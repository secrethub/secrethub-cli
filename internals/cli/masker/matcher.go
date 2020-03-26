package masker

// matches represents a set of sequence matches. The key is the index at which the match is found and the value is the
// length of the match. The index corresponds to the index of the byte in the BufferedIndex of the stream.
type matches map[int64]int

// add a new match to the map if it does not yet exist or the existing match has a shorter length.
func (m matches) add(index int64, length int) matches {
	existing, exists := m[index]
	if !exists || existing < length {
		m[index] = length
	}
	return m
}

// matcher combines multiple sequenceMatchers to check for matches of secrets against any of them.
type matcher struct {
	matchers     []*sequenceDetector
	currentIndex int64
}

// newMatcher returns a new matcher that contains a sequenceDetector for all given sequences.
func newMatcher(sequences [][]byte) *matcher {
	res := &matcher{
		matchers: make([]*sequenceDetector, len(sequences)),
	}
	for i, seq := range sequences {
		res.matchers[i] = &sequenceDetector{sequence: seq}
	}
	return res
}

// write takes in a slice of bytes and returns all matches found by any of its sequenceDetectors.
func (mb *matcher) write(in []byte) matches {
	res := matches{}
	for i, b := range in {
		for _, matcher := range mb.matchers {
			match := matcher.writeByte(b)
			if match {
				res = res.add(mb.currentIndex+int64(i-len(matcher.sequence)+1), len(matcher.sequence))
			}
		}
	}
	mb.currentIndex += int64(len(in))
	return res
}

// sequenceDetector detects if a sequence is present in the bytes it receives.
type sequenceDetector struct {
	sequence []byte
	index    int
}

// writeByte takes in a new byte to match against.
// Returns true if the given byte results in a match with sequence
func (m *sequenceDetector) writeByte(in byte) bool {
	if m.sequence[m.index] == in {
		m.index++

		if m.index == len(m.sequence) {
			m.index = 0
			return true
		}
		return false
	}

	m.index -= m.findShift()
	if m.sequence[m.index] == in {
		return m.writeByte(in)
	}
	return false
}

// findShift finds a shift > 0 for which the sequenceDetector has the longest match with its sequence.
// If no partial match can be made by shifting the sequence, the current index is returned.
func (m *sequenceDetector) findShift() int {
	for offset := 1; offset <= m.index; offset++ {
		ok := true
		for i := 0; i < m.index-offset; i++ {
			if m.sequence[i] != m.sequence[i+offset] {
				ok = false
				break
			}
		}
		if ok {
			return offset
		}
	}
	return m.index
}
