// +build !production

package fakes

import "time"

// TimeFormatter is a mock of the TimeFormatter interface
type TimeFormatter struct {
	Response string
}

// Format returns the mocked response.
func (tf *TimeFormatter) Format(t time.Time) string {
	return tf.Response
}
