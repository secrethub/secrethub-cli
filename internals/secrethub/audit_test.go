package secrethub

import (
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
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			result := tc.formatter.columnWidths()
			assert.Equal(t, result, tc.expected)
		})
	}
}
