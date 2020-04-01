package secrethub

import (
	"bytes"
	"strings"
	"testing"

	"github.com/secrethub/secrethub-cli/internals/secrethub/fakes"

	"github.com/secrethub/secrethub-go/internals/assert"
)

func Test_columnFormatter_columnWidths(t *testing.T) {
	cases := map[string]struct {
		formatter columnFormatter
		expected  []int
	}{
		"all columns fit": {
			formatter: columnFormatter{
				tableWidth: 102,
				columns: []tableColumn{
					{maxWidth: 10},
					{maxWidth: 10},
				},
			},
			expected: []int{50, 50},
		},
		"no columns fit": {
			formatter: columnFormatter{
				tableWidth: 12,
				columns: []tableColumn{
					{maxWidth: 10},
					{maxWidth: 10},
				},
			},
			expected: []int{5, 5},
		},
		"one column fits": {
			formatter: columnFormatter{
				tableWidth: 27,
				columns: []tableColumn{
					{maxWidth: 10},
					{maxWidth: 20},
				},
			},
			expected: []int{10, 15},
		},
		"multiple adjustments": {
			formatter: columnFormatter{
				tableWidth: 106,
				columns: []tableColumn{
					{maxWidth: 27},
					{maxWidth: 26},
					{maxWidth: 25},
					{maxWidth: 20},
				},
			},
			expected: []int{27, 26, 25, 20},
		},
		"no max width for some all fit": {
			formatter: columnFormatter{
				tableWidth: 64,
				columns: []tableColumn{
					{maxWidth: 15},
					{},
					{maxWidth: 15},
				},
			},
			expected: []int{15, 30, 15},
		},
		"no max width for some not all fit": {
			formatter: columnFormatter{
				tableWidth: 64,
				columns: []tableColumn{
					{maxWidth: 50},
					{},
					{maxWidth: 10},
				},
			},
			expected: []int{25, 25, 10},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			result := tc.formatter.columnWidths()
			assert.Equal(t, result, tc.expected)
		})
	}
}

func Test_columnFormatter_formatRow(t *testing.T) {
	cases := map[string]struct {
		formatter   columnFormatter
		row         []string
		expected    string
		expectedErr error
	}{
		"all cells fit": {
			formatter: columnFormatter{
				tableWidth:           102,
				computedColumnWidths: []int{50, 50},
				columns:              []tableColumn{{}, {}},
			},
			row:         []string{"foo", "bar"},
			expected:    "foo" + strings.Repeat(" ", 47) + "  " + "bar" + strings.Repeat(" ", 47) + "\n",
			expectedErr: nil,
		},
		"wrapping": {
			formatter: columnFormatter{
				tableWidth:           6,
				computedColumnWidths: []int{2, 2},
				columns:              []tableColumn{{}, {}},
			},
			row:         []string{"foo", "bar"},
			expected:    "fo  ba\no   r \n",
			expectedErr: nil,
		},
		"fits exactly": {
			formatter: columnFormatter{
				tableWidth:           8,
				computedColumnWidths: []int{3, 3},
				columns:              []tableColumn{{}, {}},
			},
			row:         []string{"foo", "bar"},
			expected:    "foo  bar\n",
			expectedErr: nil,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := tc.formatter.formatRow(tc.row)
			assert.Equal(t, err, tc.expectedErr)
			if err == nil {
				assert.Equal(t, string(result), tc.expected)
			}
		})
	}
}

func TestFallbackPager_Write(t *testing.T) {
	cases := map[string]struct {
		pager       *fallbackPager
		param       string
		expected    string
		expectedErr error
	}{
		"no lines left": {
			pager:       &fallbackPager{linesLeft: 0},
			expectedErr: errPagerNotFound,
			param:       "test\n",
			expected:    "",
		},
		"last line": {
			pager:       &fallbackPager{linesLeft: 1},
			expectedErr: errPagerNotFound,
			param:       "test\n",
			expected:    "test\n",
		},
		"print more": {
			pager:       &fallbackPager{linesLeft: 2},
			param:       "test1\ntest2\ntest3\ntest4",
			expected:    "test1\ntest2\n",
			expectedErr: errPagerNotFound,
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
