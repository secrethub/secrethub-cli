package secrethub

import (
	"fmt"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
)

// OrgPurchaseCommand prints instructions on purchasing a SecretHub subscription.
type OrgPurchaseCommand struct {
	io ui.IO
}

// NewOrgPurchaseCommand creates a new OrgPurchaseCommand.
func NewOrgPurchaseCommand(io ui.IO) *OrgPurchaseCommand {
	return &OrgPurchaseCommand{
		io: io,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *OrgPurchaseCommand) Register(r Registerer) {
	clause := r.Command("purchase", "Purchase a SecretHub subscription.")

	BindAction(clause, cmd.Run)
}

// Run prints instructions on purchasing a SecretHub subscription.
func (cmd OrgPurchaseCommand) Run() error {
	fmt.Fprintf(cmd.io.Stdout(), "An organization subscription for SecretHub can be purchased through the billing dashboard.\n\n")
	fmt.Fprintf(cmd.io.Stdout(), "For more information, check out:\nhttps://secrethub.io/docs/organizations/upgrade/\n\n")

	return nil
}
