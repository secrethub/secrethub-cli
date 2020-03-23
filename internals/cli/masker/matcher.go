package masker

type Matches map[int64]int

func (m Matches) Add(index int64, length int) Matches {
	existing, exists := m[index]
	if !exists || existing < length {
		m[index] = length
	}
	return m
}

func (m Matches) Join(other Matches) Matches {
	res := m
	for key, val := range other {
		res = res.Add(key, val)
	}
	return res
}

type multipleMatcher struct {
	matchers     []*sequenceMatcher
	currentIndex int64
}

func newMultipleMatcher(sequences [][]byte) *multipleMatcher {
	res := &multipleMatcher{
		matchers: make([]*sequenceMatcher, len(sequences)),
	}
	for i, seq := range sequences {
		res.matchers[i] = &sequenceMatcher{sequence: seq}
	}
	return res
}

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

type sequenceMatcher struct {
	sequence     []byte
	currentIndex int
}

// WriteByte takes in a new byte to Match against.
// Returns true if the given byte results in a Match with sequence
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
