package ui

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/secrethub/secrethub-go/internals/assert"

	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"
)

func TestAskWithDefault(t *testing.T) {
	question := "foo?"
	defaultValue := "bar"
	defaultOutput := "foo? [" + defaultValue + "] "
	cases := map[string]struct {
		in          []string
		expected    string
		expectedOut string
	}{
		"value entered": {
			in:          []string{"foobar\n"},
			expected:    "foobar",
			expectedOut: defaultOutput,
		},
		"no value entered": {
			in:          []string{"\n"},
			expected:    defaultValue,
			expectedOut: defaultOutput,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			io := fakeui.NewIO()
			io.PromptIn.Reads = tc.in

			// Run
			actual, err := AskWithDefault(io, question, defaultValue)

			// Assert
			assert.OK(t, err)
			assert.Equal(t, actual, tc.expected)

			assert.Equal(t, io.PromptOut.String(), tc.expectedOut)
		})
	}
}

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
			io := fakeui.NewIO()
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
			io := fakeui.NewIO()
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
	question := "foo?"
	defaultOptions := []string{
		"option 1",
		"second option",
	}
	defaultOutput := "foo?\n" +
		"  1) option 1\n" +
		"  2) second option\n" +
		"Give the number of an option: "

	cases := map[string]struct {
		in          []string
		options     []string
		n           int
		expected    int
		expectedErr error
		expectedOut string
	}{
		"first option": {
			in:          []string{"1\n"},
			options:     defaultOptions,
			n:           3,
			expected:    0,
			expectedOut: defaultOutput,
		},
		"retry": {
			in:          []string{"a\n", "1\n"},
			options:     defaultOptions,
			n:           3,
			expected:    0,
			expectedOut: defaultOutput + "\nInvalid input: not a valid number\nPlease try again.\nGive the number of an option: ",
		},
		"filter out )": {
			in:          []string{"1)\n"},
			options:     defaultOptions,
			n:           3,
			expected:    0,
			expectedOut: defaultOutput,
		},
		"out of bounds lower": {
			in:          []string{"0\n"},
			options:     defaultOptions,
			n:           1,
			expectedErr: errors.New("out of bounds"),
			expectedOut: defaultOutput + "\nInvalid input: out of bounds\n",
		},
		"out of bounds upper": {
			in:          []string{"3\n"},
			options:     defaultOptions,
			n:           1,
			expectedErr: errors.New("out of bounds"),
			expectedOut: defaultOutput + "\nInvalid input: out of bounds\n",
		},
		"not a number": {
			in:          []string{"abc\n"},
			options:     defaultOptions,
			n:           1,
			expectedErr: errors.New("not a valid number"),
			expectedOut: defaultOutput + "\nInvalid input: not a valid number\n",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			io := fakeui.NewIO()
			io.PromptIn.Reads = tc.in

			// Run
			actual, err := Choose(io, question, tc.options, tc.n)

			// Assert
			assert.Equal(t, err, tc.expectedErr)
			if tc.expectedErr == nil {
				assert.Equal(t, actual, tc.expected)
			}
			assert.Equal(t, io.PromptOut.String(), tc.expectedOut)
		})
	}
}

func TestChooseDynamicOptions(t *testing.T) {
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
			out:      "foo? (press [ENTER] for options)\n",
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
			out: "foo? (press [ENTER] for options)\n" +
				"1) foo\n" +
				"2) bar\n" +
				"3) baz\n" +
				"Type the number of an option or type a value:\n",
		},
		"choose option of second batch": {
			question: "foo?",

			in: []string{"\n", "\n", "7\n"},

			expected: "Option 7",
			out: "foo? (press [ENTER] for options)\n" +
				"1) Option 1\n" +
				"2) Option 2\n" +
				"3) Option 3\n" +
				"4) Option 4\n" +
				"5) Option 5\n" +
				"Type the number of an option or type a value (press [ENTER] for more options):\n" +
				"6) Option 6\n" +
				"7) Option 7\n" +
				"8) Option 8\n" +
				"9) Option 9\n" +
				"10) Option 10\n" +
				"Type the number of an option or type a value (press [ENTER] for more options):\n",
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
			out: "foo? (press [ENTER] for options)\n" +
				"1) foobar    baz\n" +
				"2) bar       baz\n" +
				"3) baz       bar\n" +
				"Type the number of an option or type a value:\n",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Setup
			io := fakeui.NewIO()
			io.PromptIn.Reads = tc.in

			if tc.getOptions == nil {
				og := optionGetter{}
				tc.getOptions = og.Get
			}

			// Run
			actual, err := ChooseDynamicOptions(io, tc.question, tc.getOptions, tc.addOwn, "value")

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
