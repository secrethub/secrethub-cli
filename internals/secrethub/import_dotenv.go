package secrethub

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
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
	envVar, err := godotenv.Read(cmd.dotenvFile)
	if err != nil {
		return err
	}

	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	locationsMap := make(map[string]string)
	if cmd.interactive {
		mappingString, err := openEditor(buildFile(cmd.path.Value(), getMapKeys(envVar)))
		if err != nil {
			return err
		}
		locationsMap = buildMap(mappingString)
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

// openEditor opens an editor with the provided input as contents,
// lets the user edit those contents with the editor and returns
// the edited contents.
// Note that this functions is blocking for user input.
func openEditor(input string) (string, error) {
	fpath := os.TempDir() + "secretPaths.txt"
	f, err := os.Create(fpath)
	if err != nil {
		return "", err
	}

	_, err = f.WriteString(input)
	if err != nil {
		return "", err
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "editor"
	}

	cmd := exec.Command(editor, fpath)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		return "", err
	}
	err = cmd.Wait()
	if err != nil {
		return "", err
	}

	reading, err := ioutil.ReadFile(fpath)
	if err != nil {
		return "", err
	}

	err = os.Remove(fpath)
	if err != nil {
		return "", err
	}

	return string(reading), nil
}

func buildFile(path string, secretPaths []string) string {
	output := "Choose the paths to where your secrets will be written:\n"

	for _, secretPath := range secretPaths {
		output += fmt.Sprintf("%s => %s/%s\n", secretPath,
			path, strings.ToLower(secretPath))
	}
	return output
}

func buildMap(input string) map[string]string {
	scanner := bufio.NewScanner(strings.NewReader(input))
	scanner.Scan()
	locationsMap := make(map[string]string)

	for scanner.Scan() {
		line := scanner.Text()
		split := strings.Split(line, "=>")
		locationsMap[strings.TrimSpace(split[0])] = strings.TrimSpace(split[1])
	}
	return locationsMap
}

func getMapKeys(stringMap map[string]string) []string {
	keys := make([]string, 0, len(stringMap))

	for k := range stringMap {
		keys = append(keys, k)
	}
	return keys
}
