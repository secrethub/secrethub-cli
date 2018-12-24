package ui

import (
	"testing"

	"bytes"

	"github.com/keylockerbv/secrethub/testutil"
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
			testutil.Compare(t, err, nil)
			testutil.Compare(t, actual, tc.expected)
			testutil.Compare(t, io.PromptOut.String(), "question: ")
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
			testutil.Compare(t, err, nil)
			testutil.Compare(t, actual, tc.expected)
			testutil.Compare(t, io.PromptOut.String(), tc.out)
		})
	}
}
