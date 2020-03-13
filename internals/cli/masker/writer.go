// Package masker provides a wrapper around an io.Writer that replaces sensitive values in its output.
package masker

import (
	"io"
	"sync"
	"time"
)

// matcher is an interface used by MaskedWriter to find matches of sequences to mask.
type matcher interface {
	Read(byte) int
	InProgress() bool
	Reset()
}

type sequenceMatcher struct {
	sequence     []byte
	currentIndex int
}

// Read takes in a new byte to match against.
// If the given byte results in a match with sequence, the number of matched bytes is returned.
func (m *sequenceMatcher) Read(in byte) int {
	if m.sequence[m.currentIndex] == in {
		m.currentIndex++

		if m.currentIndex == len(m.sequence) {
			m.currentIndex = 0
			return len(m.sequence)
		}
		return 0
	}

	m.currentIndex -= m.findShift()
	if m.sequence[m.currentIndex] == in {
		return m.Read(in)
	}
	return 0
}

// findShift checks whether we can also make a partial match by decreasing the currentIndex .
// For example, if the sequence is foofoobar, if someone inserts foofoofoobar, we still want to match.
// So after the third f is inserted, the currentIndex is decreased by 3 with the following code.
func (m *sequenceMatcher) findShift() int {
	for offset := 1; offset <= m.currentIndex; offset++ {
		ok := true
		for i := 0; i < m.currentIndex-offset; i++ {
			if m.sequence[i] != m.sequence[i+offset] {
				ok = false
				break
			}
		}
		if ok {
			return offset
		}
	}
	return m.currentIndex
}

// InProgress returns whether this sequenceMatcher is currently partially matching.
//
// For example, if the sequence is "foobar" and the registered input is "foob", InProgress() returns true.
func (m *sequenceMatcher) InProgress() bool {
	return m.currentIndex > 0
}

// Reset forgets the current match.
func (m *sequenceMatcher) Reset() {
	m.currentIndex = 0
}

type input struct {
	bytes  []byte
	target io.Writer
}

// maskByte represents a byte and whether the byte should be masked or not.
type maskByte struct {
	byte
	masked bool
}

type output struct {
	byte   maskByte
	target io.Writer
}

// Masker masks all occurrences of masks by maskString.
// If no write is made for timeout on the io.Writer, any matches in progress are reset
// and the buffer is flushed. This is to ensure that the writer does not hang on partial matches.
type Masker struct {
	maskString string
	masks      [][]byte
	timeout    time.Duration

	matchers        map[io.Writer][]matcher
	buf             []output
	incomingBytesCh chan input
	outputTimeoutCh chan struct{}
	outputCh        chan []output
	errCh           chan error
	wg              sync.WaitGroup
}

// New returns a new masker on which you can create a writer that masks all occurrences of sequences in masks with maskString.
func New(masks [][]byte, maskString string, timeout time.Duration) *Masker {
	return &Masker{
		maskString:      maskString,
		masks:           masks,
		timeout:         timeout,
		matchers:        make(map[io.Writer][]matcher),
		errCh:           make(chan error, 1),
		outputTimeoutCh: make(chan struct{}, 1),
		incomingBytesCh: make(chan input),
		outputCh:        make(chan []output),
	}
}

// MaskedWriter implements io.Writer and masks all occurrences of masks with the mask string.
type MaskedWriter struct {
	w io.Writer
	m *Masker
}

func (mw *MaskedWriter) Write(p []byte) (n int, err error) {
	return mw.m.Write(mw.w, p)
}

// NewWriter returns a new io.Writer that masks all occurrences of masks with the mask string.
func (mw *Masker) NewWriter(w io.Writer) io.Writer {
	mw.matchers[w] = make([]matcher, len(mw.masks))
	for i, mask := range mw.masks {
		mw.matchers[w][i] = &sequenceMatcher{
			sequence: mask,
		}
	}

	return &MaskedWriter{
		w: w,
		m: mw,
	}
}

// Write performs one Write operations for an io.Writer.
// It is responsible for finding any matches to mask and mark the appropriate bytes as masked.
// This function never returns an error. These can instead be caught with Flush().
func (mw *Masker) Write(w io.Writer, p []byte) (n int, err error) {
	mw.wg.Add(len(p))
	tmp := make([]byte, len(p))
	copy(tmp, p)
	mw.incomingBytesCh <- input{
		bytes:  tmp,
		target: w,
	}
	return len(p), nil
}

// process receives incoming bytes from calls to Write (over the incoming bytes channel) masks them if necessary and passes
// them through to Run (over the output channel).
//
// - When the incoming bytes do not end in a partial secret, they are directly passed to the output channel.
// - When the incoming bytes do end in a partial secret, all bytes up to the partial secret are passed to the
//   output channel. Passing on the partial secrets bytes is delayed until:
//     - New bytes come in that finish the secret, in which case the bytes are passed to the output channel and marked as masked.
//     - New bytes come in that do not finish the secret, in which case the bytes are passed to the output channel and not marked as masked.
//     - The output timeout channel receives a signal, in which case the bytes are passed to the output channel and not marked as masked.
func (mw *Masker) process() {
	for {
		select {
		case <-mw.outputTimeoutCh:
			// Only flush if there is still nothing send to the output channel.
			if len(mw.outputCh) == 0 {
				for _, matchers := range mw.matchers {
					for _, matcher := range matchers {
						matcher.Reset()
					}
				}
				mw.flushBuffer()
			}
		case p := <-mw.incomingBytesCh:
			for _, b := range p.bytes {
				matchInProgress := false
				mw.buf = append(mw.buf, output{byte: maskByte{byte: b}, target: p.target})

				for _, matcher := range mw.matchers[p.target] {
					maskLen := matcher.Read(b)
					for i := 0; i < maskLen; i++ {
						mw.buf[len(mw.buf)-1-i].byte.masked = true
					}
					matchInProgress = matchInProgress || matcher.InProgress()
				}

				if !matchInProgress {
					mw.flushBuffer()
				}
			}
		}
	}
}

func (mw *Masker) flushBuffer() {
	tmp := make([]output, len(mw.buf))
	copy(tmp, mw.buf)
	mw.outputCh <- tmp
	mw.buf = mw.buf[:0]
}

// Run writes any processed data from the output channel to the underlying io.Writer.
// If no new data is received on the output channel for timeout, the output buffer is forced flushed
// and all ongoing matches are reset.
//
// This should be run in a separate goroutine.
func (mw *Masker) Run() {
	go mw.process()
	masking := false
	for {
		select {
		case output := <-mw.outputCh:
			for _, b := range output {
				var err error
				if b.byte.masked {
					if !masking {
						_, err = b.target.Write([]byte(mw.maskString))
						if err != nil {
							mw.errCh <- err
							return
						}
					}
					masking = true
				} else {
					_, err = b.target.Write([]byte{b.byte.byte})
					if err != nil {
						mw.errCh <- err
						return
					}
					masking = false
				}
			}
			mw.wg.Add(-len(output))
		case <-time.After(mw.timeout):
			// send to the timeout channel if not already done so.
			select {
			case mw.outputTimeoutCh <- struct{}{}:
			default:
			}
		}
	}
}

// Flush is called to make sure that all output is written to the underlying io.Writer.
// Returns any errors caused by the writing.
func (mw *Masker) Flush() error {
	go func() {
		mw.wg.Wait()
		mw.errCh <- nil
	}()
	return <-mw.errCh
}
