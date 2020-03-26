package masker

// Matches represents a set of sequence matches. The key is the index at which the match is found and the value is the
// length of the match. The index corresponds to the index of the byte in the BufferedIndex of the stream.
type Matches map[int64]int

// Add a new match to the map if it does not yet exist or the existing match has a shorter length.
func (m Matches) Add(index int64, length int) Matches {
	existing, exists := m[index]
	if !exists || existing < length {
		m[index] = length
	}
	return m
}

// multipleMatcher combines multiple sequenceMatchers to check for matches of secrets against any of them.
type multipleMatcher struct {
	matchers     []*sequenceMatcher
	currentIndex int64
}

// newMultipleMatcher returns a new multipleMatcher that contains a sequenceMatcher for all given sequences.
func newMultipleMatcher(sequences [][]byte) *multipleMatcher {
	res := &multipleMatcher{
		matchers: make([]*sequenceMatcher, len(sequences)),
	}
	for i, seq := range sequences {
		res.matchers[i] = &sequenceMatcher{sequence: seq}
	}
	return res
}

// Write takes in a slice of bytes and returns all matches found by any of its sequenceMatchers.
func (mb *multipleMatcher) Write(in []byte) Matches {
	res := Matches{}
	for i, b := range in {
		for _, matcher := range mb.matchers {
			match := matcher.WriteByte(b)
			if match {
				res = res.Add(mb.currentIndex+int64(i-len(matcher.sequence)+1), len(matcher.sequence))
			}
		}
	}
	mb.currentIndex += int64(len(in))
	return res
}

// sequenceMatcher takes in bytes to check whether there is any match with the given sequence.
type sequenceMatcher struct {
	sequence     []byte
	currentIndex int
}

// WriteByte takes in a new byte to match against.
// Returns true if the given byte results in a match with sequence
func (m *sequenceMatcher) WriteByte(in byte) bool {
	if m.sequence[m.currentIndex] == in {
		m.currentIndex++

		if m.currentIndex == len(m.sequence) {
			m.currentIndex = 0
			return true
		}
		return false
	}

	m.currentIndex -= m.findShift()
	if m.sequence[m.currentIndex] == in {
		return m.WriteByte(in)
	}
	return false
}

// findShift checks whether we can also make a partial Match by decreasing the currentIndex .
// For example, if the sequence is foofoobar, if someone inserts foofoofoobar, we still want to Match.
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
