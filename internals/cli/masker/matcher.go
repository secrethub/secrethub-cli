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
func (m *matcher) write(in []byte) matches {
	res := matches{}
	for i, b := range in {
		for _, matcher := range m.matchers {
			match := matcher.writeByte(b)
			if match {
				res = res.add(m.currentIndex+int64(i-len(matcher.sequence)+1), len(matcher.sequence))
			}
		}
	}
	m.currentIndex += int64(len(in))
	return res
}

// sequenceDetector detects if a sequence is present in the bytes it receives.
type sequenceDetector struct {
	sequence []byte
	index    int
}

// writeByte takes in a new byte to match against.
// Returns true if the given byte results in a match with sequence
func (d *sequenceDetector) writeByte(in byte) bool {
	if d.sequence[d.index] == in {
		d.index++

		if d.index == len(d.sequence) {
			d.index = 0
			return true
		}
		return false
	}

	d.index -= d.findShift()
	if d.sequence[d.index] == in {
		return d.writeByte(in)
	}
	return false
}

// findShift checks whether we can also make a partial match by shifting the detector's index, returning the number
// of positions the index should be shifted. If no partial match can be made by shifting, the current index is returned.
// For example, if the sequence to match is "foobar" and "foofoobar" is encountered in the stream we still want to
// trigger a positive match. So after the second "f" character is encountered, findShift returns 3 to indicate the
// index should be decreased by 3.
func (d *sequenceDetector) findShift() int {
	for offset := 1; offset <= d.index; offset++ {
		found := true
		for i := 0; i < d.index-offset; i++ {
			if d.sequence[i] != d.sequence[i+offset] {
				found = false
				break
			}
		}
		if found {
			return offset
		}
	}
	return d.index
}
