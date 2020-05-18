package pager

import (
	"bytes"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/secrethub/fakes"
	"gotest.tools/assert"
)

func TestFallbackPager_Write(t *testing.T) {
	cases := map[string]struct {
		pager       *fallbackPager
		param       string
		expected    string
		expectedErr error
	}{
		"no lines left": {
			pager:       &fallbackPager{linesLeft: 0},
			expectedErr: ErrPagerNotFound,
			param:       "test\n",
			expected:    "",
		},
		"last line": {
			pager:       &fallbackPager{linesLeft: 1},
			expectedErr: ErrPagerNotFound,
			param:       "test\n",
			expected:    "test\n",
		},
		"print more": {
			pager:       &fallbackPager{linesLeft: 2},
			param:       "test1\ntest2\ntest3\ntest4",
			expected:    "test1\ntest2\n",
			expectedErr: ErrPagerNotFound,
		},
		"more lines left": {
			pager:       &fallbackPager{linesLeft: 3},
			expectedErr: nil,
			param:       "test\n",
			expected:    "test\n",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			buffer := bytes.Buffer{}
			tc.pager.writer = &fakes.Pager{Buffer: &buffer}
			_, err := tc.pager.Write([]byte(tc.param))
			assert.Equal(t, err, tc.expectedErr)
			assert.Equal(t, buffer.String(), tc.expected)
		})
	}
}
