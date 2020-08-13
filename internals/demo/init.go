package demo

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secretpath"
)

type newClientFunc func() (secrethub.ClientInterface, error)

const defaultDemoRepo = "demo"

type InitCommand struct {
	repo      pathValue
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
	clause := r.CreateCommand("init", "Create the secrets necessary to connect with the demo application.")
	clause.HelpLong("demo init creates a repository with the username and password needed to connect to the demo API.")

	clause.Flags().VarPF(&cmd.repo, "repo", "", "The path of the repository to create. Defaults to a "+defaultDemoRepo+" repo in your personal namespace.")

	command.BindAction(clause, nil, cmd.Run)
}

// Run handles the command with the options as specified in the command.
func (cmd *InitCommand) Run() error {
	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	var repoPath string
	var username string
	if cmd.repo.path == "" {
		me, err := client.Me().GetUser()
		if err != nil {
			return err
		}
		username = me.Username
		repoPath = secretpath.Join(me.Username, defaultDemoRepo)
	} else {
		username = secretpath.Namespace(cmd.repo.path.Value())
		repoPath = cmd.repo.path.Value()
	}

	_, err = client.Repos().Create(repoPath)
	if err == api.ErrRepoAlreadyExists {
		demoRepo, err := cmd.isDemoRepo(client, repoPath)
		if err != nil {
			return err
		}
		if demoRepo {
			return nil
		}
		return fmt.Errorf("repo %s already exists and is not a demo repo, use --repo to specify another repo to use", repoPath)
	} else if err != nil {
		return err
	}

	usernamePath := secretpath.Join(repoPath, "username")
	_, err = client.Secrets().Write(usernamePath, []byte(username))
	if err != nil {
		return err
	}

	h := hmac.New(sha256.New, []byte("this-is-no-good-way-to-generate-a-password-that-is-why-we-only-use-it-for-demo-purposes"))
	password := base64.RawStdEncoding.EncodeToString(h.Sum([]byte(username)))[:20]

	passwordPath := secretpath.Join(repoPath, "password")
	_, err = client.Secrets().Write(passwordPath, []byte(password))
	if err != nil {
		return err
	}

	fmt.Printf("Created the following secrets:\n%s\n%s\n", usernamePath, passwordPath)

	return nil
}

// isDemoRepo checks whether the repo on the given path is a demo repository.
// It returns true iff the repository contains exactly two secrets named username and password.
func (cmd *InitCommand) isDemoRepo(client secrethub.ClientInterface, repoPath string) (bool, error) {
	repo, err := client.Repos().Get(repoPath)
	if err != nil {
		return false, err
	}
	if repo.SecretCount != 2 {
		return false, nil
	}

	usernamePath := secretpath.Join(repoPath, "username")
	exists, err := client.Secrets().Exists(usernamePath)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	passwordPath := secretpath.Join(repoPath, "password")
	exists, err = client.Secrets().Exists(passwordPath)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}
	return true, nil
}

type pathValue struct {
	path api.RepoPath
}

func (u *pathValue) String() string {
	return u.path.String()
}

func (u *pathValue) Set(s string) error {
	var err error
	u.path, err = api.NewRepoPath(s)
	if err != nil {
		return err
	}
	return nil
}

func (u *pathValue) Type() string {
	return "pathValue"
}
