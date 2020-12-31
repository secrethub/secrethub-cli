package cli

import "fmt"

func (c *CommandClause) argumentError(args []string) error {
	if len(args) >= getRequired(c.Args) && len(args) <= len(c.Args) {
		return nil
	}
	errorText, minimum, maximum := "", getRequired(c.Args), len(c.Args)

	if minimum == maximum {
		errorText += fmt.Sprintf("`secrethub "+c.fullCommand()+"` requires exactly %d argument(s).", minimum)
	} else {
		errorText += fmt.Sprintf("`secrethub "+c.fullCommand()+"` requires between %d and %d arguments.", minimum, maximum)
	}
	errorText += "\n\nSee `secrethub " + c.fullCommand() + " --help` for help.\n\n" + useLine(c.Cmd, c.Args)
	errorText += "\n\n" + c.Cmd.Short

	return fmt.Errorf(errorText)
}
