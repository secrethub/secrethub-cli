package secrethub

import (
	"fmt"
	"github.com/secrethub/secrethub-cli/internals/cli"

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

// Registeclause.Cmd.Rootters the command, arguments and flags on the provided Registerer.
func (cmd *OrgPurchaseCommand) Register(r cli.Registerer) {
	clause := r.Command("purchase", "Purchase a SecretHub subscription.")

	clause.BindAction(cmd.Run)
	clause.BindArguments(nil)
}

// Run prints instructions on purchasing a SecretHub subscription.
func (cmd OrgPurchaseCommand) Run() error {
	fmt.Fprintf(cmd.io.Output(), "An organization subscription for SecretHub can be purchased through the billing dashboard.\n\n")
	fmt.Fprintf(cmd.io.Output(), "For more information, check out:\nhttps://secrethub.io/docs/organizations/upgrade/\n\n")

	return nil
}
