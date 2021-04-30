package secrethub

import (
	"errors"
	"fmt"
	"os"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
)

var ErrDone = errors.New("done")

// AccountDeleteCommand is a command to inspect account details.
type AccountDeleteCommand struct {
	io              ui.IO
	newClient       newClientFunc
	credentialStore CredentialConfig
}

// NewAccountDeleteCommand creates a new AccountDeleteCommand.
func NewAccountDeleteCommand(io ui.IO, newClient newClientFunc, credentialStore CredentialConfig) *AccountDeleteCommand {
	return &AccountDeleteCommand{
		io:              io,
		newClient:       newClient,
		credentialStore: credentialStore,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *AccountDeleteCommand) Register(r cli.Registerer) {
	clause := r.Command("delete", "Delete your SecretHub account.")

	clause.BindAction(cmd.Run)
}

// Run handles the command with the options as specified in the command.
func (cmd *AccountDeleteCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}
	var account *api.Account
	account, err = client.Accounts().Me()
	if err != nil {
		return err
	}

	orgList, err := client.Orgs().ListMine()
	if err != nil {
		return err
	}
	if len(orgList) > 0 {
		fmt.Fprintf(cmd.io.Output(), "You are a member of %d orgs. In order to delete your account you will need to either leave these orgs or delete them.\n", len(orgList))
		err = cmd.leaveOrRmOrgs(client, orgList)
		if err != nil {
			return err
		}
	}

	var confirmed bool
	confirmed, err = ui.ConfirmCaseInsensitive(
		cmd.io,
		fmt.Sprintf("[DANGER ZONE] This action cannot be undone. "+
			"This will permanently delete the account named %s along with all its repositories and secrets. "+
			"Please type in the name of the account to confirm", account.Name),
		account.Name.String(),
	)
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Fprintln(cmd.io.Output(), "Name does not match. Aborting.")
		return nil
	}

	err = client.Accounts().Delete(account.AccountID)
	if err != nil {
		return err
	}
	if cmd.credentialStore.Source() == CredentialSourceConfigDir {
		err := os.Remove(cmd.credentialStore.ConfigDir().Credential().Path())
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not delete credential file: %v", err)
		}
	}
	fmt.Fprintln(cmd.io.Output(), "Your account was successfully deleted.")
	return nil
}

func (cmd *AccountDeleteCommand) leaveOrRmOrgs(client secrethub.ClientInterface, orgs []*api.Org) error {
	me, err := client.Me().GetUser()
	if err != nil {
		return err
	}

	for i, org := range orgs {
		fmt.Fprintf(cmd.io.Output(), "[%d/%d] %s\n", i, len(orgs), org.Name)
		orgMembers, err := client.Orgs().Members().List(org.Name)
		if err != nil {
			return err
		}

		admin := false
		adminCount := 0
		for _, member := range orgMembers {
			if member.Role == api.OrgRoleAdmin {
				adminCount++
				if member.User.AccountID == me.AccountID {
					admin = true
				}
			}
		}
		var options []func() error
		var optionNames []string
		leave := func() error {
			_, err = client.Orgs().Members().Revoke(org.Name, me.Username, nil)
			return err
		}
		deleteOrg := func() error {
			return cmd.deleteOrg(client, org.Name)
		}
		transferOwnership := func() error {
			err = cmd.transferAdminRole(client, org, orgMembers)
			if err != nil {
				return err
			}
			return leave()
		}

		if admin {
			optionNames = append(optionNames, "Delete it")
			options = append(options, deleteOrg)
		}
		if !admin || adminCount > 1 {
			optionNames = append(optionNames, "Leave it")
			options = append(options, leave)
		}
		if admin && adminCount == 1 && len(orgMembers)-adminCount > 1 {
			optionNames = append(optionNames, "Make someone else admin and leave it")
			options = append(options, transferOwnership)
		}

		optionNames = append(optionNames, "Abort")
		options = append(options, func() error {
			return ErrDone
		})
		choice, err := ui.Choose(cmd.io, fmt.Sprintf("What would you like to do with the org named '%s'?", org.Name), optionNames, 3)
		if err != nil {
			return err
		}
		err = options[choice]()
		if err == ErrDone {
			fmt.Fprintf(cmd.io.Output(), "Aborting...\n")
			return nil
		} else if err != nil {
			return err
		}
	}
	return nil
}

func (cmd *AccountDeleteCommand) deleteOrg(client secrethub.ClientInterface, org string) error {
	confirmed, err := ui.ConfirmCaseInsensitive(
		cmd.io,
		fmt.Sprintf(
			"[DANGER ZONE] This action cannot be undone. "+
				"This will permanently delete the %s organization, repositories, and remove all team associations. "+
				"Please type in the name of the organization to confirm",
			org,
		),
		org,
	)
	if err != nil {
		return err
	}

	if !confirmed {
		fmt.Fprintln(cmd.io.Output(), "Name does not match. Aborting.")
		return nil
	}
	return client.Orgs().Delete(org)
}

func (cmd *AccountDeleteCommand) transferAdminRole(client secrethub.ClientInterface, org *api.Org, members []*api.OrgMember) error {
	var memberNames []string
	for _, member := range members {
		if member.Role == api.OrgRoleMember {
			memberNames = append(memberNames, fmt.Sprintf("%s (%s)", member.User.Username, member.User.FullName))
		}
	}
	choice, err := ui.Choose(cmd.io, fmt.Sprintf("Who sould become the new admin of '%s'?", org.Name), memberNames, 3)
	if err != nil {
		return err
	}
	_, err = client.Orgs().Members().Update(org.Name, memberNames[choice], api.OrgRoleAdmin)
	return err
}
