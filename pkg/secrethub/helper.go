package secrethub

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/secrethub/secrethub-go/internals/api"
)

// pluralize returns the plural or single string depending on the number of items.
func pluralize(single string, plural string, items int) string {
	if items == 1 {
		return fmt.Sprintf("1 %s", single)
	}
	return fmt.Sprintf("%d %s", items, plural)
}

var (
	red = color.New(color.FgRed, color.Bold)
)

// colorizeByStatus adds optional color to a given message based on status.
func colorizeByStatus(status string, msg interface{}) interface{} {
	switch status {
	case api.StatusFlagged:
		return red.Sprint(msg)
	default:
		return msg
	}
}
