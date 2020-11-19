package secrethub

import (
	"fmt"
	"strings"

	"github.com/joho/godotenv"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
	"github.com/secrethub/secrethub-go/internals/api"
)

// ImportDotEnvCommand handles the migration of secrets from .env files to SecretHub.
type ImportDotEnvCommand struct {
	io          ui.IO
	path        api.DirPath
	interactive bool
	force       bool
	dotenvFile  string
	newClient   newClientFunc
}

// NewImportDotEnvCommand creates a new ImportDotEnvCommand.
func NewImportDotEnvCommand(io ui.IO, newClient newClientFunc) *ImportDotEnvCommand {
	return &ImportDotEnvCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command and its sub-commands on the provided Registerer.
func (cmd *ImportDotEnvCommand) Register(r command.Registerer) {
	clause := r.Command("dotenv", "Import secrets from `.env` files. Outputs a `secrethub.env` file, containing references to your secrets in SecretHub.")
	clause.Arg("dir-path", "path to a directory on SecretHub in which to store the imported secrets").PlaceHolder(dirPathPlaceHolder).SetValue(&cmd.path)
	clause.Flag("interactive", "Interactive mode. Edit the paths to where the secrets should be written.").Short('i').BoolVar(&cmd.interactive)
	clause.Flag("env-file", "The location of the .env file. Defaults to `.env`.").Default(".env").ExistingFileVar(&cmd.dotenvFile)
	registerForceFlag(clause).BoolVar(&cmd.force)
	command.BindAction(clause, cmd.Run)
}

func (cmd *ImportDotEnvCommand) Run() error {
	var envVar map[string]string
	locationsMap := make(map[string]string)

	envVar, err := godotenv.Read(cmd.dotenvFile)
	if err != nil {
		return err
	}

	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	if cmd.interactive {
		locationsMap, err = openEditor(cmd.path.Value(), getMapKeys(envVar))
		if err != nil {
			return err
		}
	} else {
		for key := range envVar {
			locationsMap[key] = cmd.path.Value() + "/" + strings.ToLower(key)
		}
	}

	if !cmd.force {
		for key := range envVar {
			exists, err := client.Secrets().Exists(locationsMap[key])
			if err != nil {
				return err
			}
			if exists {
				confirmed, err := ui.AskYesNo(cmd.io, fmt.Sprintf("A secret at location %s already exists. "+
					"This import process will overwrite this secret. Do you wish to continue?", locationsMap[key]), ui.DefaultNo)

				if err != nil {
					return err
				}

				if !confirmed {
					_, err = fmt.Fprintln(cmd.io.Output(), "Aborting.")
					if err != nil {
						return err
					}
					return nil
				}
			}
		}
	}

	for key, value := range envVar {
		_, err = client.Secrets().Write(locationsMap[key], []byte(value))
		if err != nil {
			return err
		}
	}

	_, err = fmt.Fprintf(cmd.io.Output(), "Transfer complete! The secrets have been written to %s.\n", cmd.path.String())
	if err != nil {
		return err
	}

	return nil
}
