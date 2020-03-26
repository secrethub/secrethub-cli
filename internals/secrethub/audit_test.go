package secrethub

import (
	"strings"
	"testing"

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
				columns: []auditTableColumn{
					{maxWidth: 10},
					{maxWidth: 10},
				},
			},
			expected: []int{50, 50},
		},
		"no columns fit": {
			formatter: columnFormatter{
				tableWidth: 12,
				columns: []auditTableColumn{
					{maxWidth: 10},
					{maxWidth: 10},
				},
			},
			expected: []int{5, 5},
		},
		"one column fits": {
			formatter: columnFormatter{
				tableWidth: 27,
				columns: []auditTableColumn{
					{maxWidth: 10},
					{maxWidth: 20},
				},
			},
			expected: []int{10, 15},
		},
		"multiple adjustments": {
			formatter: columnFormatter{
				tableWidth: 106,
				columns: []auditTableColumn{
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
				columns: []auditTableColumn{
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
				columns: []auditTableColumn{
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
				columns:              []auditTableColumn{{}, {}},
			},
			row:         []string{"foo", "bar"},
			expected:    "foo" + strings.Repeat(" ", 47) + "  " + "bar" + strings.Repeat(" ", 47),
			expectedErr: nil,
		},
		"wrapping": {
			formatter: columnFormatter{
				tableWidth:           6,
				computedColumnWidths: []int{2, 2},
				columns:              []auditTableColumn{{}, {}},
			},
			row:         []string{"foo", "bar"},
			expected:    "fo  ba\no   r ",
			expectedErr: nil,
		},
		"fits exactly": {
			formatter: columnFormatter{
				tableWidth:           8,
				computedColumnWidths: []int{3, 3},
				columns:              []auditTableColumn{{}, {}},
			},
			row:         []string{"foo", "bar"},
			expected:    "foo  bar",
			expectedErr: nil,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := tc.formatter.formatRow(tc.row)
			assert.Equal(t, err, tc.expectedErr)
			if err == nil {
				assert.Equal(t, result, tc.expected)
			}
		})
	}
}
