package secrethub

import (
	"strconv"

	"github.com/fatih/color"
	"github.com/secrethub/secrethub-go/internals/errio"
)

// noColorFlag configures the global behaviour to disable colored output.
type noColorFlag bool

// init disables colored output based on the value of the flag.
func (f noColorFlag) init() {
	color.NoColor = bool(f)
}

// RegisterColorFlag registers a color flag that configures whether colored output is used.
func RegisterColorFlag(r FlagRegisterer) {
	flag := noColorFlag(false)
	r.Flag("no-color", "Disable colored output.").SetValue(&flag)
}

// String implements the flag.Value interface.
func (f noColorFlag) String() string {
	return strconv.FormatBool(bool(f))
}

// Set disables colors when the given value is false.
func (f *noColorFlag) Set(value string) error {
	b, err := strconv.ParseBool(value)
	if err != nil {
		return errio.Error(err)
	}
	*f = noColorFlag(b)
	f.init()
	return nil
}

// IsBoolFlag makes the flag a boolean flag when used in a Kingpin application.
// Thus, the flag can be used without argument (--color or --no-color).
func (f noColorFlag) IsBoolFlag() bool {
	return true
}
