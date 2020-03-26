package masker

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/secrethub/secrethub-go/internals/assert"
)

func TestMultipleMatcher(t *testing.T) {
	testSequences := [][]byte{
		[]byte("first sequence"),
		[]byte("test"),
		[]byte("test but longer"),
		[]byte("another first"),
	}

	cases := map[string]struct {
		sequences   [][]byte
		inputs      [][]byte
		wantMatches []matches
	}{
		"no matches": {
			sequences: testSequences,
			inputs: [][]byte{
				[]byte("12345678"),
			},
			wantMatches: []matches{
				nil,
			},
		},
		"single input single match": {
			sequences: testSequences,
			inputs: [][]byte{
				[]byte("123test"),
			},
			wantMatches: []matches{
				map[int64]int{
					3: 4,
				},
			},
		},
		"single input double match": {
			sequences: testSequences,
			inputs: [][]byte{
				[]byte("123test89test"),
			},
			wantMatches: []matches{
				map[int64]int{
					3: 4,
					9: 4,
				},
			},
		},
		"subset match": {
			sequences: testSequences,
			inputs: [][]byte{
				[]byte("12test but longer"),
			},
			wantMatches: []matches{
				map[int64]int{
					2: 15,
				},
			},
		},
		"overlappig matches": {
			sequences: testSequences,
			inputs: [][]byte{
				[]byte("12another first sequence"),
			},
			wantMatches: []matches{
				map[int64]int{
					2:  13,
					10: 14,
				},
			},
		},
		"double write single match": {
			sequences: testSequences,
			inputs: [][]byte{
				[]byte("123"),
				[]byte("4test"),
			},
			wantMatches: []matches{
				nil,
				map[int64]int{
					4: 4,
				},
			},
		},
		"match across 2 writes": {
			sequences: testSequences,
			inputs: [][]byte{
				[]byte("123te"),
				[]byte("st"),
			},
			wantMatches: []matches{
				nil,
				map[int64]int{
					3: 4,
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			matcher := newMatcher(tc.sequences)

			for i, input := range tc.inputs {
				t.Run(strconv.Itoa(i), func(t *testing.T) {
					gotMatches := matcher.write(input)
					assert.Equal(t, gotMatches, tc.wantMatches[i])
				})
			}
		})
	}
}

func TestSequenceMatcher(t *testing.T) {
	tests := []struct {
		matchString     string
		input           string
		expectedMatches []int
	}{
		{
			matchString:     "test",
			input:           "test",
			expectedMatches: []int{0},
		},
		{
			matchString:     "test",
			input:           "ttest",
			expectedMatches: []int{1},
		},
		{
			matchString:     "test",
			input:           "testtest",
			expectedMatches: []int{0, 4},
		},
		{
			matchString:     "testtest",
			input:           "test",
			expectedMatches: nil,
		},
		{
			matchString:     "foofoobar",
			input:           "foofoofoobar",
			expectedMatches: []int{3},
		},
		{
			matchString:     "test",
			input:           "123 testtest",
			expectedMatches: []int{4, 8},
		},
		{
			matchString:     "test",
			input:           "t est",
			expectedMatches: nil,
		},
		{
			matchString:     "test",
			input:           "tesat",
			expectedMatches: nil,
		},
		{
			matchString:     "test",
			input:           "tesT",
			expectedMatches: nil,
		},
		{
			matchString:     "t",
			input:           "ttattt",
			expectedMatches: []int{0, 1, 3, 4, 5},
		},
		{
			matchString:     "tt",
			input:           "ttattt",
			expectedMatches: []int{0, 3},
		},
	}

	for _, tc := range tests {
		name := fmt.Sprintf("%s in %s", tc.matchString, tc.input)

		t.Run(name, func(t *testing.T) {
			matcher := sequenceDetector{sequence: []byte(tc.matchString)}
			var matches []int
			for i, b := range []byte(tc.input) {
				match := matcher.writeByte(b)
				if match {
					matches = append(matches, i-len(tc.matchString)+1)
				}
			}
			assert.Equal(t, matches, tc.expectedMatches)
		})
	}

}
