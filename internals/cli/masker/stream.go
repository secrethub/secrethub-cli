package masker

import (
	"bytes"
	"io"
	"sync"
	"time"
)

// stream is a buffered io.Writer that masks all secrets written on it.
type stream struct {
	dest          io.Writer
	buf           indexedBuffer
	registerFrame func(*stream, time.Duration, int)

	matcher     *matcher
	matches     matches
	matchesLock sync.Mutex
}

// Write implements the io.Writer interface for the stream.
// The written frame is stored in the buffer and it is registered in the Masker to make sure it is flushed from
// the buffer after the constant buffer delay has passed.
// The bytes are also passed to the secret matcher to check for any matches with secrets.
func (s *stream) Write(p []byte) (int, error) {
	// Save the current time to compensate for the time taken to match for secrets.
	referenceTime := time.Now()

	n, err := s.buf.write(p)

	for index, length := range s.matcher.write(p[:n]) {
		s.addMatch(index, length)
	}

	if n > 0 {
		s.registerFrame(s, time.Until(referenceTime), n)
	}

	return n, err
}

// addMatch adds the match of a secret at the given index and with the given length to the map of matches.
// If the associated bytes have already been written to the destination, the match is ignored to avoid storing matches
// that are never being processed by flush().
func (s *stream) addMatch(index int64, length int) {
	s.matchesLock.Lock()
	defer s.matchesLock.Unlock()

	if index >= s.buf.currentIndex {
		s.matches = s.matches.add(index, length)
	}
}

// flush n bytes from the buffer and mask any secrets that have been matched.
func (s *stream) flush(n int) error {
	startIndex := s.buf.currentIndex
	endIndex := startIndex + int64(n)

	// Increment the frameIndex before processing matches to avoid adding new matches in the processed frame.
	for i := startIndex; i < endIndex; i++ {
		s.matchesLock.Lock()
		length, exists := s.matches[i]
		s.matchesLock.Unlock()

		if exists {
			// Get any unprocessed bytes before this match to the destination.
			bytesBeforeMatch, err := s.buf.writeUpToIndex(s.dest, i)
			if err != nil {
				return err
			}

			// Only write the redaction text if there were bytes between this match and the previous match
			// or this is the first flush for the buffer.
			if bytesBeforeMatch > 0 || s.buf.currentIndex == 0 {
				_, err = s.dest.Write([]byte("<redacted by SecretHub>"))
				if err != nil {
					return err
				}
			}

			// Drop all bytes until the end of the mask.
			_, err = s.buf.writeUpToIndex(io.Discard, i+int64(length))
			if err != nil {
				return err
			}

			delete(s.matches, i)
		}
	}

	// Write all bytes after the last match.
	_, err := s.buf.writeUpToIndex(s.dest, endIndex)
	if err != nil {
		return err
	}

	return nil
}

// indexedBuffer is a goroutine safe buffer that assigns every byte that is written to it with an incrementing index.
type indexedBuffer struct {
	buffer       bytes.Buffer
	mutex        sync.Mutex
	currentIndex int64
}

func (b *indexedBuffer) write(p []byte) (n int, err error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.buffer.Write(p)
}

// writeUpToIndex pops all bytes in the buffer up to the given index and writes them to the given writer.
// The number of bytes written and any errors encountered are returned
func (b *indexedBuffer) writeUpToIndex(w io.Writer, index int64) (int, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if index < b.currentIndex {
		return 0, nil
	}
	n := int(index - b.currentIndex)
	b.currentIndex = index
	bufferSlice := b.buffer.Next(n)
	return w.Write(bufferSlice)
}
