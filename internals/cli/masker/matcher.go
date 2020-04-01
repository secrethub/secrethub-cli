package masker

import (
	"bytes"
	"crypto/subtle"
)

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
	detectors    []*sequenceDetector
	currentIndex int64
}

// newMatcher returns a new matcher that contains a sequenceDetector for all given sequences.
func newMatcher(sequences [][]byte) *matcher {
	res := &matcher{
		detectors: make([]*sequenceDetector, 0, len(sequences)),
	}

	for _, sequence := range sequences {
		res.detectors = append(res.detectors, &sequenceDetector{
			sequence: sequence,
			offset:   0,
		})
		// Add detectors for any repetitions in the sequence.
		// For example, if the sequence is "aaab", we also require a detector for "aaaab" and "aaaaab".
		for length, count := range sequenceRepetitions(sequence) {
			for i := 1; i <= count; i++ {
				prefixedSequence := make([]byte, len(sequence)+length*i)
				copy(prefixedSequence, sequence[:length*i])
				copy(prefixedSequence[length*i:], sequence)
				res.detectors = append(res.detectors, &sequenceDetector{
					sequence: prefixedSequence,
					offset:   length * i,
				})
			}
		}
	}
	return res
}

// write takes in a slice of bytes and returns all matches found by any of its detectors.
func (m *matcher) write(in []byte) matches {
	res := matches{}
	for i, b := range in {
		for _, detector := range m.detectors {
			match := detector.writeByte(b)
			if match {
				res = res.add(m.currentIndex+int64(i-detector.length()+1), detector.length())
			}
		}
	}
	m.currentIndex += int64(len(in))
	return res
}

// sequenceDetector detects if a sequence is present in the bytes it receives.
type sequenceDetector struct {
	sequence []byte
	offset   int
	index    int
}

// length returns the length of the sequence corrected for its offset.
func (d *sequenceDetector) length() int {
	return len(d.sequence) - d.offset
}

// writeByte takes in a new byte to match against.
// Returns true if the given byte results in a match with sequence
// The implementation tries to reduce the effect of the input on the execution duration as much as possible
// to limit the information that can be derived from measuring the execution time of the masking functionality.
func (d *sequenceDetector) writeByte(in byte) bool {
	// Implementation of the following code that limits data-dependency as much as possible.
	//
	// 	if d.sequence[d.index] == in {
	// 		d.index++
	//
	//		if d.index == len(d.sequence) {
	//			d.index = 0
	//			return true
	//		}
	//		return false
	//	} else if d.sequence[0] == in{
	//		d.index = d.offset + 1
	//	} else {
	//      d.index = 0
	//  }
	//	return false

	newIndex := d.index

	correctInput := subtle.ConstantTimeByteEq(d.sequence[newIndex], in)
	newIndex = subtle.ConstantTimeSelect(correctInput, newIndex+1, newIndex)
	sequenceComplete := subtle.ConstantTimeEq(int32(len(d.sequence)), int32(newIndex))

	newIndexIfNotCorrectInput := subtle.ConstantTimeSelect(subtle.ConstantTimeByteEq(d.sequence[0], in), d.offset+1, 0)
	newIndex = subtle.ConstantTimeSelect(correctInput, newIndex, newIndexIfNotCorrectInput)

	d.index = subtle.ConstantTimeSelect(sequenceComplete, 0, newIndex)

	return sequenceComplete == 1
}

// sequenceRepetitions finds all repetitions of bytes in the start of the sequence and returns a map of the length of
// repeated sequences as the key and the number of repetitions as the value.
// Example, if the input is aabaabaab, the "a" is repeated 1 time and "aab" is repeated 2 times,
// so the result is: {1:1, 3:2}
func sequenceRepetitions(seq []byte) map[int]int {
	res := map[int]int{}
	for i := 1; i < len(seq); i++ {
		count := 0
		for j := 1; i*(j+1) <= len(seq); j++ {
			if bytes.Equal(seq[0:i], seq[i*j:i*(j+1)]) {
				count++
			} else {
				break
			}
		}
		if count > 0 {
			res[i] = count
		}
	}
	return res
}
