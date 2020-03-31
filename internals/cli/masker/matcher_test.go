package masker

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/pkg/randchar"
)

func TestMultipleMatcher(t *testing.T) {
	testSequences := [][]byte{
		[]byte("first sequence"),
		[]byte("test"),
		[]byte("test but longer"),
		[]byte("another first"),
		[]byte("112"),
		[]byte("22222222223"),
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
		"repeat in match": {
			sequences: testSequences,
			inputs: [][]byte{
				[]byte("1112"),
			},
			wantMatches: []matches{
				map[int64]int{
					1: 3,
				},
			},
		},
		"repeat in match followed by a new match": {
			sequences: testSequences,
			inputs: [][]byte{
				[]byte("1112112"),
			},
			wantMatches: []matches{
				map[int64]int{
					1: 3,
					4: 3,
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

func TestMatcher_Repeats(t *testing.T) {
	N := 30
	repeats := 7

	for repeatLen := 1; repeatLen < 10; repeatLen++ {
		sequence, err := randchar.Generate(repeatLen)
		assert.OK(t, err)

		sequences := [][]byte{
			[]byte(strings.Repeat(string(sequence), repeats) + "!"),
		}
		for i := 0; i < N; i++ {
			input := []byte(strings.Repeat(string(sequence), i) + "!")
			t.Run(strconv.Itoa(i)+"/"+string(input), func(t *testing.T) {

				prefix, err := randchar.MustNewRand(randchar.Symbols).Generate(rand.Intn(20))
				assert.OK(t, err)

				input := append(prefix, input...)
				matcher := newMatcher(sequences)

				matches := matcher.write(input)

				expected := map[int64]int{}
				if i >= repeats {
					expected[int64((i-repeats)*repeatLen+len(prefix))] = repeatLen*repeats + 1
				}
				assert.Equal(t, matches, expected)
			})
		}
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
			expectedMatches: []int{}, // This case is handled by adding multiple detectors with newMatcher
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

func TestSequenceRepetitions(t *testing.T) {
	cases := map[string]struct {
		sequence string
		want     map[int]int
	}{
		"no repetition": {
			sequence: "abcd",
			want:     map[int]int{},
		},
		"single sequence": {
			sequence: "aabcd",
			want: map[int]int{
				1: 1,
			},
		},
		"single sequence multiple times": {
			sequence: "aaaabcd",
			want: map[int]int{
				1: 3,
				2: 1,
			},
		},
		"double repetition": {
			sequence: "aabcaabc",
			want: map[int]int{
				1: 1,
				4: 1,
			},
		},
		"repetition with divider": {
			sequence: "abcdabc",
			want:     map[int]int{},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := sequenceRepetitions([]byte(tc.sequence))

			assert.Equal(t, got, tc.want)
		})
	}
}

func doBench(sequences [][]byte, input []byte) int {
	m := newMatcher(sequences)
	_ = m.write(input)
	maxIndex := 0
	for _, d := range m.detectors {
		if d.index > maxIndex {
			maxIndex = d.index
		}
	}
	return maxIndex
}

func BenchmarkMatcher(b *testing.B) {
	sequences := make([][]byte, 100)
	goodSequences := make([][]byte, 100)
	badSequences := make([][]byte, 100)
	for i := range sequences {
		seq, err := randchar.Generate(1024)
		assert.OK(b, err)
		sequences[i] = seq

		goodSequences[i] = make([]byte, 512)
		copy(goodSequences[i], seq)

		badSeq, err := randchar.Generate(512)
		assert.OK(b, err)
		badSequences[i] = badSeq
	}

	startTime := time.Now()
	for -time.Until(startTime) < time.Minute {
		for i := 0; i < len(sequences); i++ {
			index := doBench(sequences, goodSequences[i])
			assert.Equal(b, index, 512)
			index = doBench(sequences, badSequences[i])
			assert.Equal(b, index < 512, true)
		}
	}

	fmt.Println("end warming up")

	b.Run("random sequences", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			doBench(sequences, badSequences[rand.Intn(len(badSequences))])
		}
	})

	b.Run("matching sequences", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			doBench(sequences, goodSequences[rand.Intn(len(goodSequences))])
		}
	})

}
