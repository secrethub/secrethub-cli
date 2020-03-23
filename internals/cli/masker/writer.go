package masker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

type stream struct {
	dest    io.Writer
	buf     bytes.Buffer
	masker  *Masker
	matcher *multipleMatcher
	matches Matches
	index   int64
}

func (s *stream) Write(p []byte) (int, error) {
	s.matches = s.matches.Join(s.matcher.Write(p))

	n, err := s.buf.Write(p)
	if n > 0 {
		s.masker.addTrigger(s, n)
	}
	return n, err
}

func (s *stream) flushN(n int) error {
	transportBuf := make([]byte, n)
	nRead, err := s.buf.Read(transportBuf)

	if nRead != n {
		return errors.New("number of bytes in buffer lower than expected")
	}
	if err != nil {
		return fmt.Errorf("error with buffer: %v", err)
	}

	for i := 0; i < n; i++ {
		if length, exists := s.matches[s.index+int64(i)]; exists {
			maskLowerIndex := i
			maskUpperIndex := i + length

			// If the match exceeds, add a new match at the beginning of the next flush.
			if maskUpperIndex > n {
				s.matches = s.matches.Add(s.index+int64(n), maskUpperIndex-n)
				maskUpperIndex = n
			}
			for maskIndex := maskLowerIndex; maskIndex < maskUpperIndex; maskIndex++ {
				transportBuf[maskIndex] = '*'
			}
			delete(s.matches, s.index+int64(i))
		}
	}

	s.index += int64(n)

	_, err = bytes.NewReader(transportBuf).WriteTo(s.dest)
	return err
}

type trigger struct {
	length int
	stream *stream
	timer  *time.Timer
}

type Masker struct {
	BufferDelay    time.Duration
	MatchSequences [][]byte

	lock         sync.Mutex
	triggers     []trigger
	flushAllChan chan struct{}
}

func (m *Masker) AddStream(w io.Writer) io.Writer {
	s := stream{
		dest:    w,
		masker:  m,
		matches: Matches{},
		matcher: newMultipleMatcher(m.MatchSequences),
	}
	return &s
}

func (m *Masker) Run(ctx context.Context) {
	m.flushAllChan = make(chan struct{})
	for {
		select {
		case <-m.flushAllChan:
			m.flushAll()
			return
		default:
		}

		if len(m.triggers) == 0 {
			select {
			case <-time.After(m.BufferDelay / 2):
				continue
			case <-m.flushAllChan:
				m.flushAll()
				return
			}
		}

		m.lock.Lock()
		trigger := m.triggers[0]
		m.lock.Unlock()

		select {
		case <-trigger.timer.C:
			err := trigger.stream.flushN(trigger.length)
			if err != nil {
				m.handleErr(err)
			}
		}

		m.lock.Lock()
		m.triggers = m.triggers[1:]
		m.lock.Unlock()
	}
}

func (m *Masker) flushAll() {
	for _, t := range m.triggers {
		err := t.stream.flushN(t.length)
		if err != nil {
			m.handleErr(err)
		}
	}
	m.triggers = m.triggers[:0]
	m.flushAllChan <- struct{}{}
}

func (m *Masker) Wait() {
	m.flushAllChan <- struct{}{}
	<-m.flushAllChan
}

func (m *Masker) addTrigger(s *stream, l int) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.triggers = append(m.triggers, trigger{
		length: l,
		stream: s,
		timer:  time.NewTimer(m.BufferDelay),
	})
}

func (m *Masker) handleErr(err error) {
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
}
