package masker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sync"
	"time"
)

type stream struct {
	dest        io.Writer
	buf         bytes.Buffer
	masker      *Masker
	matches     map[int64]struct{}
	index       int64
	matchLock   sync.Mutex
	maxLookback int
}

func (s *stream) Write(p []byte) (int, error) {
	n, err := s.buf.Write(p)
	if n > 0 {
		s.masker.addTrigger(s, n)
	}
	return n, err
}

func (s *stream) flushN(n int) error {
	s.matchLock.Lock()
	s.lookForMatches(n)

	transportBuf := make([]byte, n)
	nRead, err := s.buf.Read(transportBuf)
	oldIndex := s.index
	s.index += int64(nRead)
	s.matchLock.Unlock()

	if nRead != n {
		return errors.New("number of bytes in buffer lower than expected")
	}
	if err != nil {
		return fmt.Errorf("error with buffer: %v", err)
	}

	for i := 0; i < n; i++ {
		if _, exists := s.matches[oldIndex+int64(i)]; exists {
			transportBuf[i] = '*'
			delete(s.matches, oldIndex+int64(i))
		}
	}

	_, err = bytes.NewReader(transportBuf).WriteTo(s.dest)
	return err
}

func (s *stream) lookForMatches(n int) {
	searchBuf := s.buf.Bytes()
	bufLen := n + s.maxLookback
	if bufLen > len(searchBuf) {
		bufLen = len(searchBuf)
	}
	searchBuf = searchBuf[:bufLen]
	for _, seq := range s.masker.MatchSequences {

		// finding matches with a regexp is really easy but it is far from efficient.
		for _, match := range seq.FindAllIndex(searchBuf, -1) {
			for i := match[0]; i < match[1]; i++ {
				s.matches[s.index+int64(i)] = struct{}{}
			}
		}
	}
}

type trigger struct {
	length int
	stream *stream
	timer  *time.Timer
}

type Masker struct {
	BufferDelay    time.Duration
	MatchSequences []*regexp.Regexp

	lock         sync.Mutex
	triggers     []trigger
	flushAllChan chan struct{}
}

func (m *Masker) AddStream(w io.Writer) io.Writer {
	s := stream{
		dest:        w,
		masker:      m,
		matches:     map[int64]struct{}{},
		maxLookback: 1024,
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
