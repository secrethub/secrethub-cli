package ui

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/secrethub/secrethub-go/internals/assert"
)

func TestConfirmCaseInsensitive(t *testing.T) {
	cases := map[string]struct {
		expectedConfirmation []string
		promptIn             string
		expected             bool
	}{
		"confirmed, one choice": {
			[]string{"answer"},
			"answer",
			true,
		},
		"not confirmed, one choice": {
			[]string{"answer"},
			"otheranswer",
			false,
		},
		"confirmed, first choice": {
			[]string{"answer1", "answer2"},
			"answer1",
			true,
		},
		"confirmed, second choice": {
			[]string{"answer1", "answer2"},
			"answer2",
			true,
		},
		"not confirmed, two choices": {
			[]string{"answer1", "answer2"},
			"answer3",
			false,
		},
		"confirmed, lowercase": {
			[]string{"ANSWER"},
			"answer",
			true,
		},
		"confirmed, uppercase": {
			[]string{"answer"},
			"ANSWER",
			true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			io := NewFakeIO()
			io.PromptIn.Buffer = bytes.NewBufferString(tc.promptIn)

			// Run
			actual, err := ConfirmCaseInsensitive(io, "question", tc.expectedConfirmation...)

			// Assert
			assert.Equal(t, err, nil)
			assert.Equal(t, actual, tc.expected)
			assert.Equal(t, io.PromptOut.String(), "question: ")
		})
	}
}

func TestAskYesNo(t *testing.T) {
	cases := map[string]struct {
		question      string
		defaultAnswer ConfirmationType
		in            []string
		expected      bool
		out           string
	}{
		"default yes": {
			question:      "question",
			defaultAnswer: DefaultYes,
			in:            []string{"\n"},
			expected:      true,
			out:           "question [Y/n]: ",
		},
		"default no": {
			question:      "question",
			defaultAnswer: DefaultNo,
			in:            []string{"\n"},
			expected:      false,
			out:           "question [y/N]: ",
		},
		"default none": {
			question:      "question",
			defaultAnswer: DefaultNone,
			in:            []string{"\n", "\n", "\n"},
			expected:      false,
			out: "question [y/n]: " +
				"question [y/n]: " +
				"question [y/N]: ",
		},
		"n": {
			question:      "question",
			defaultAnswer: DefaultNone,
			in:            []string{"n\n"},
			expected:      false,
			out:           "question [y/n]: ",
		},
		"N": {
			question:      "question",
			defaultAnswer: DefaultNone,
			in:            []string{"N\n"},
			expected:      false,
			out:           "question [y/n]: ",
		},
		"NO": {
			question:      "question",
			defaultAnswer: DefaultNone,
			in:            []string{"NO\n"},
			expected:      false,
			out:           "question [y/n]: ",
		},
		"no": {
			question:      "question",
			defaultAnswer: DefaultNone,
			in:            []string{"no\n"},
			expected:      false,
			out:           "question [y/n]: ",
		},
		"No": {
			question:      "question",
			defaultAnswer: DefaultNone,
			in:            []string{"No\n"},
			expected:      false,
			out:           "question [y/n]: ",
		},
		"y": {
			question:      "question",
			defaultAnswer: DefaultNone,
			in:            []string{"y\n"},
			expected:      true,
			out:           "question [y/n]: ",
		},
		"Y": {
			question:      "question",
			defaultAnswer: DefaultNone,
			in:            []string{"Y\n"},
			expected:      true,
			out:           "question [y/n]: ",
		},
		"yes": {
			question:      "question",
			defaultAnswer: DefaultNone,
			in:            []string{"yes\n"},
			expected:      true,
			out:           "question [y/n]: ",
		},
		"YES": {
			question:      "question",
			defaultAnswer: DefaultNone,
			in:            []string{"YES\n"},
			expected:      true,
			out:           "question [y/n]: ",
		},
		"Yes": {
			question:      "question",
			defaultAnswer: DefaultNone,
			in:            []string{"Yes\n"},
			expected:      true,
			out:           "question [y/n]: ",
		},
		"invalid default yes": {
			question:      "question",
			defaultAnswer: DefaultYes,
			in:            []string{"Yesshouldnotwork\n", "n\n"},
			expected:      false,
			out: "question [Y/n]: " +
				"question [Y/n]: ",
		},
		"invalid default no": {
			question:      "question",
			defaultAnswer: DefaultNo,
			in:            []string{"noshouldnotwork\n", "y\n"},
			expected:      true,
			out: "question [y/N]: " +
				"question [y/N]: ",
		},
		"invalid default none": {
			question:      "question",
			defaultAnswer: DefaultNone,
			in:            []string{"invalid\n", "invalid\n", "invalid\n"},
			expected:      false,
			out: "question [y/n]: " +
				"question [y/n]: " +
				"question [y/N]: ",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			io := NewFakeIO()
			io.PromptIn.Reads = tc.in

			// Run
			actual, err := AskYesNo(io, tc.question, tc.defaultAnswer)

			// Assert
			assert.Equal(t, err, nil)
			assert.Equal(t, actual, tc.expected)
			assert.Equal(t, io.PromptOut.String(), tc.out)
		})
	}
}

func TestChoose(t *testing.T) {
	cases := map[string]struct {
		question   string
		getOptions func() ([]Option, bool, error)
		addOwn     bool

		in []string

		expected string
		out      string
	}{
		"directly add own": {
			question: "foo?",
			addOwn:   true,
			in:       []string{"bar\n"},
			expected: "bar",
			out:      "foo? Press [ENTER] for options.\n",
		},
		"choose option of first batch": {
			question: "foo?",
			getOptions: func() ([]Option, bool, error) {
				return []Option{
					{Value: "foo", Display: "foo"},
					{Value: "bar", Display: "bar"},
					{Value: "baz", Display: "baz"},
				}, true, nil
			},

			in: []string{"\n", "2\n"},

			expected: "bar",
			out: "foo? Press [ENTER] for options.\n" +
				"1) foo\n" +
				"2) bar\n" +
				"3) baz\n" +
				"Type a number of choice or enter a value: \n",
		},
		"choose option of second batch": {
			question: "foo?",

			in: []string{"\n", "\n", "7\n"},

			expected: "Option 7",
			out: "foo? Press [ENTER] for options.\n" +
				"1) Option 1\n" +
				"2) Option 2\n" +
				"3) Option 3\n" +
				"4) Option 4\n" +
				"5) Option 5\n" +
				"Type a number of choice or enter a value ([ENTER] for more options): \n" +
				"6) Option 6\n" +
				"7) Option 7\n" +
				"8) Option 8\n" +
				"9) Option 9\n" +
				"10) Option 10\n" +
				"Type a number of choice or enter a value ([ENTER] for more options): \n",
		},
		"options formatted": {
			question: "foo?",
			getOptions: func() ([]Option, bool, error) {
				return []Option{
					{Value: "foo", Display: "foobar\tbaz"},
					{Value: "bar", Display: "bar\tbaz"},
					{Value: "baz", Display: "baz\tbar"},
				}, true, nil
			},

			in:       []string{"\n", "2\n"},
			expected: "bar",
			out: "foo? Press [ENTER] for options.\n" +
				"1) foobar    baz\n" +
				"2) bar       baz\n" +
				"3) baz       bar\n" +
				"Type a number of choice or enter a value: \n",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			io := NewFakeIO()
			io.PromptIn.Reads = tc.in

			if tc.getOptions == nil {
				og := optionGetter{}
				tc.getOptions = og.Get
			}

			// Run
			actual, err := Choose(io, tc.question, tc.getOptions, tc.addOwn, "value")

			// Assert
			assert.Equal(t, err, nil)
			assert.Equal(t, actual, tc.expected)
			assert.Equal(t, io.PromptOut.String(), tc.out)
		})
	}
}

type optionGetter struct {
	n int
}

func (og *optionGetter) Get() ([]Option, bool, error) {
	res := make([]Option, 5)
	for i := 0; i < 5; i++ {
		option := fmt.Sprintf("Option %d", og.n+i+1)
		res[i] = Option{
			Value:   option,
			Display: option,
		}
	}
	og.n += 5
	return res, false, nil
}
