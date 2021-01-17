package cli

import (
	"fmt"
)

func (c *CommandClause) argumentError(args []string) error {
	if len(args) >= getRequired(c.Args) && len(args) <= len(c.Args) {
		return nil
	}
	errorText, minimum, maximum := "", getRequired(c.Args), len(c.Args)

	if c.Args[0].Value.String() == new(StringListValue).String() {
		errorText += fmt.Sprintf(`"%s" requires at least %d %s.`, c.fullCommand(), minimum, pluralize("argument", minimum))
	} else if minimum == maximum {
		errorText += fmt.Sprintf(`"%s" requires exactly %d %s.`, c.fullCommand(), minimum, pluralize("argument", minimum))
	} else {
		errorText += fmt.Sprintf(`"%s" requires between %d and %d arguments.`, c.fullCommand(), minimum, maximum)
	}

	errorText += fmt.Sprintf("\nSee `%s --help` for help.\n\nUsage: %s\n\n%s", c.fullCommand(), useLine(c.Cmd, c.Args), c.Cmd.Short)

	return fmt.Errorf(errorText)
}

func pluralize(word string, num int) string {
	if num == 1 {
		return word
	}
	return word + "s"
}
