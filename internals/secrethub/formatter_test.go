package secrethub

import (
	"strings"
	"testing"

	"github.com/secrethub/secrethub-go/internals/assert"
)

func Test_columnFormatter_columnWidths(t *testing.T) {
	cases := map[string]struct {
		formatter tableFormatter
		expected  []int
	}{
		"all columns fit": {
			formatter: tableFormatter{
				tableWidth: 102,
				columns: []tableColumn{
					{maxWidth: 10},
					{maxWidth: 10},
				},
			},
			expected: []int{50, 50},
		},
		"no columns fit": {
			formatter: tableFormatter{
				tableWidth: 12,
				columns: []tableColumn{
					{maxWidth: 10},
					{maxWidth: 10},
				},
			},
			expected: []int{5, 5},
		},
		"one column fits": {
			formatter: tableFormatter{
				tableWidth: 27,
				columns: []tableColumn{
					{maxWidth: 10},
					{maxWidth: 20},
				},
			},
			expected: []int{10, 15},
		},
		"multiple adjustments": {
			formatter: tableFormatter{
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
			formatter: tableFormatter{
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
			formatter: tableFormatter{
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
		formatter tableFormatter
		row       []string
		expected  string
	}{
		"all cells fit": {
			formatter: tableFormatter{
				tableWidth:           102,
				computedColumnWidths: []int{50, 50},
				columns:              []tableColumn{{}, {}},
			},
			row:      []string{"foo", "bar"},
			expected: "foo" + strings.Repeat(" ", 47) + "  " + "bar" + strings.Repeat(" ", 47) + "\n",
		},
		"wrapping": {
			formatter: tableFormatter{
				tableWidth:           6,
				computedColumnWidths: []int{2, 2},
				columns:              []tableColumn{{}, {}},
			},
			row:      []string{"foo", "bar"},
			expected: "fo  ba\no   r \n",
		},
		"fits exactly": {
			formatter: tableFormatter{
				tableWidth:           8,
				computedColumnWidths: []int{3, 3},
				columns:              []tableColumn{{}, {}},
			},
			row:      []string{"foo", "bar"},
			expected: "foo  bar\n",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			result := tc.formatter.formatRow(tc.row)
			assert.Equal(t, string(result), tc.expected)
		})
	}
}
