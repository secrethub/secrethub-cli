package cli

import (
	"fmt"
)

func (c *CommandClause) validateArgumentsCount(args []string) error {
	minimum := getRequired(c.Args)
	maximum := len(c.Args)
	if len(args) >= minimum && len(args) <= maximum {
		return nil
	}

	if minimum == maximum {
		return c.argumentError(fmt.Sprintf("requires exactly %d %s", minimum, pluralize("argument", minimum)))
	}
	return c.argumentError(fmt.Sprintf("requires between %d and %d arguments", minimum, maximum))
}

func (c *CommandClause) validateArgumentsArrCount(args []string) error {
	if len(args) == 0 {
		return c.argumentError("requires at least 1 argument")
	}
	return nil
}

func (c *CommandClause) argumentError(errorText string) error {
	return fmt.Errorf(
		"%s %s.\n"+
			"See `%s --help` for help.\n"+
			"\n"+
			"Usage: %s\n"+
			"\n"+
			"%s",
		c.fullCommand(),
		errorText,
		c.fullCommand(),
		useLine(c.Cmd, c.Args),
		c.Cmd.Short,
	)
}

func pluralize(word string, num int) string {
	if num == 1 {
		return word
	}
	return word + "s"
}
