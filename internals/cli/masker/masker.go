package masker

import (
	"io"
	"time"
)

// Masker handles the creation and synchronization of streams that have all their writes scanned for secrets and
// have them redacted if any matches are found. Output on all streams is buffered to increase the chance of finding
// secrets if they are spread across multiple writes.
//
// Usage:
// 1. Create a new Masker using New()
// 2. Add one more streams using AddStream()
// 3. Run the Run() method in a separate goroutine
// 4. After everything has been written to the io.Writers, flush all buffers using Flush()
type Masker struct {
	BufferDelay time.Duration

	sequences    [][]byte
	frames       chan frame
	flushAllChan chan struct{}
	err          error
}

// New creates a new Masker that scans all streams for the given sequences and masks them.
func New(sequences [][]byte) *Masker {
	return &Masker{
		BufferDelay:  time.Millisecond * 100,
		sequences:    sequences,
		frames:       make(chan frame, 1024),
		flushAllChan: make(chan struct{}),
	}
}

// AddStream takes in an io.Writer to mask secrets on and returns an io.Writer that has secrets on its output masked.
func (m *Masker) AddStream(w io.Writer) io.Writer {
	s := stream{
		dest:          w,
		registerFrame: m.registerFrame,
		matches:       Matches{},
		matcher:       newMultipleMatcher(m.sequences),
	}
	return &s
}

// Run continuously flushes the input buffer for each frame for which the buffer delay has passed.
// This method should be run in a separate goroutine.
// If a struct is passed on flushAllChan, all pending frames are flushed to the output and the method returns.
func (m *Masker) Run() {
	for {
		select {
		case <-m.flushAllChan:
			for t := range m.frames {
				err := t.stream.flush(t.length)
				if err != nil {
					m.handleErr(err)
				}
			}
			m.flushAllChan <- struct{}{}
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

// Flush all pending frames and wait for this to complete.
// This should be run after all input has been written to the io.Writers of the streams.
func (m *Masker) Flush() error {
	m.flushAllChan <- struct{}{}
	close(m.frames)
	<-m.flushAllChan

	return m.err
}

// registerFrame adds a new frame to the frames channel with a timeout of BufferDelay.
// After this timer has passed, the frame will be flushed to the output.
func (m *Masker) registerFrame(s *stream, l int) {
	m.frames <- frame{
		length: l,
		stream: s,
		timer:  time.NewTimer(m.BufferDelay),
	}
}

func (m *Masker) handleErr(err error) {
	if err != nil && m.err == nil {
		m.err = err
	}
}

type frame struct {
	length int
	stream *stream
	timer  *time.Timer
}
