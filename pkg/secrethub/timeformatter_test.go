package secrethub

import (
	"testing"
	"time"

	"github.com/keylockerbv/secrethub/testutil"
)

func TestTimeFormatter_Format(t *testing.T) {
	testutil.Unit(t)

	tzAmsterdam, _ := time.LoadLocation("Europe/Amsterdam")

	cases := map[string]struct {
		tf       timeFormatter
		time     time.Time
		expected string
	}{
		"human readable time": {
			tf:       timeFormatter(false),
			time:     time.Now().Add(-1 * time.Hour),
			expected: "About an hour ago",
		},
		"timestamp UTC": {
			tf:       timeFormatter(true),
			time:     time.Date(2018, 1, 1, 1, 1, 1, 1, time.UTC),
			expected: "2018-01-01T01:01:01Z",
		},
		"timestamp Amsterdam": {
			tf:       timeFormatter(true),
			time:     time.Date(2018, 1, 1, 1, 1, 1, 1, tzAmsterdam),
			expected: "2018-01-01T01:01:01+01:00",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Act
			actual := tc.tf.Format(tc.time)

			// Assert
			testutil.Compare(t, actual, tc.expected)
		})
	}
}
