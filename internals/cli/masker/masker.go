package masker

import (
	"io"
	"time"
)

// Masker handles the creation and synchronization of streams that have all their writes scanned for secrets and
// have them redacted if any matches are found. Masking of secrets is a best effort attempt. Output on all streams is
// buffered to increase the chance of finding secrets if they are spread across multiple writes, but it cannot be
// guaranteed that these secrets are masked. The duration bytes spend in the buffer is constant.
//
// Usage:
// 1. Create a new Masker using New()
// 2. Add one more streams using AddStream()
// 3. Run the Start() method in a separate goroutine
// 4. After everything has been written to the io.Writers, flush all buffers using Stop()
type Masker struct {
	bufferDelay time.Duration
	sequences   [][]byte
	frames      chan frame
	stopChan    chan struct{}
	err         error
}

// Options for configuring masking behavior.
type Options struct {
	// DisableBuffer completely disables the buffering of the masker. This increases output responsiveness
	// but also increases the chance of a secret not being masked.
	DisableBuffer bool

	// BufferDelay is the constant duration for which input to a stream is buffered. A higher value increases
	// the chance of secrets being detected for masking. Especially when writes have a variable delay between them,
	// for example in the case data arrives over an unstable network connection.
	// Defaults to 50ms if not set.
	BufferDelay time.Duration

	// FrameBufferLength is the number of frames that can be in the buffer simultaneously.
	// If the frame buffer is full, writing to a stream blocks until there is space.
	FrameBufferLength int
}

// New creates a new Masker that scans all streams for the given sequences and masks them.
func New(sequences [][]byte, opts *Options) *Masker {
	masker := &Masker{
		bufferDelay: time.Millisecond * 50,
		sequences:   sequences,
		stopChan:    make(chan struct{}),
	}
	frameChanlength := 1024
	if opts != nil {
		if opts.DisableBuffer {
			masker.bufferDelay = 0
			frameChanlength = 0
		} else {
			if opts.BufferDelay > 0 {
				masker.bufferDelay = opts.BufferDelay
			}
			if opts.FrameBufferLength > 0 {
				frameChanlength = opts.FrameBufferLength
			}
		}

	}
	masker.frames = make(chan frame, frameChanlength)

	return masker
}

// AddStream takes in an io.Writer to mask secrets on and returns an io.Writer that has secrets on its output masked.
func (m *Masker) AddStream(w io.Writer) io.Writer {
	s := stream{
		dest:          w,
		registerFrame: m.registerFrame,
		matches:       matches{},
		matcher:       newMatcher(m.sequences),
	}
	return &s
}

// Start continuously flushes the input buffer for each frame for which the buffer delay has passed.
// This method blocks until Stop() is called.
func (m *Masker) Start() {
	for {
		select {
		case <-m.stopChan:
			for t := range m.frames {
				err := t.stream.flush(t.length)
				if err != nil {
					m.handleErr(err)
				}
			}
			m.stopChan <- struct{}{}
			return
		case trigger := <-m.frames:
			<-trigger.timer.C

			err := trigger.stream.flush(trigger.length)
			if err != nil {
				m.handleErr(err)
			}
		}
	}
}

// Stop all pending frames and wait for this to complete.
// This should be run after all input has been written to the io.Writers of the streams.
// Calling Write() on a stream after calling Stop() will lead to a panic.
func (m *Masker) Stop() error {
	m.stopChan <- struct{}{}
	close(m.frames)
	<-m.stopChan

	return m.err
}

// registerFrame adds a new frame to the frames channel with a timeout of bufferDelay plus the given offset.
// After this timer has passed, the frame will be flushed to the output.
func (m *Masker) registerFrame(s *stream, offset time.Duration, l int) {
	m.frames <- frame{
		length: l,
		stream: s,
		timer:  time.NewTimer(offset + m.bufferDelay),
	}
}

func (m *Masker) handleErr(err error) {
	if err != nil && m.err == nil {
		m.err = err
	}
}

// frame represent a set of bytes in the buffer of a stream that were written in a single call of Write().
// The bytes are written to the destination after the timer has expired.
type frame struct {
	length int
	stream *stream
	timer  *time.Timer
}
