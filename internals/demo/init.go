package demo

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secretpath"
)

type newClientFunc func() (secrethub.ClientInterface, error)

const defaultDemoRepo = "demo"

type InitCommand struct {
	repo api.RepoPath

	io        ui.IO
	newClient newClientFunc
}

func NewInitCommand(io ui.IO, newClient newClientFunc) *InitCommand {
	return &InitCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *InitCommand) Register(r command.Registerer) {
	clause := r.Command("init", "Create the secrets necessary to connect with the demo application.")
	clause.HelpLong("demo init creates a repository with the username and password needed to connect to the demo API.")

	clause.Flag("repo", "The path of the repository to create. Defaults to a "+defaultDemoRepo+" repo in your personal namespace.").SetValue(&cmd.repo)

	command.BindAction(clause, cmd.Run)
}

// Run handles the command with the options as specified in the command.
func (cmd *InitCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	var repoPath string
	var username string
	if cmd.repo == "" {
		me, err := client.Me().GetUser()
		if err != nil {
			return err
		}
		username = me.Username
		repoPath = secretpath.Join(me.Username, defaultDemoRepo)
	} else {
		username = secretpath.Namespace(cmd.repo.Value())
		repoPath = cmd.repo.Value()
	}

	_, err = client.Repos().Create(repoPath)
	if err == api.ErrRepoAlreadyExists && cmd.repo == "" {
		return errors.New("demo repo already exists, use --repo to specify another repo to use")
	} else if err != nil {
		return err
	}

	_, err = client.Secrets().Write(secretpath.Join(repoPath, "username"), []byte(username))
	if err != nil {
		return err
	}

	h := hmac.New(sha256.New, []byte("this-is-no-good-way-to-generate-a-password-that-is-why-we-only-use-it-for-demo-purposes"))
	password := base64.RawStdEncoding.EncodeToString(h.Sum([]byte(username)))[:20]

	_, err = client.Secrets().Write(secretpath.Join(repoPath, "password"), []byte(password))
	if err != nil {
		return err
	}

	return nil
}
