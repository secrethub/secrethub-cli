package secrethub

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"sync"
	"text/tabwriter"

	"golang.org/x/sync/errgroup"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/pkg/secretpath"

	"github.com/joho/godotenv"
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

	keys := make([]string, 0, len(envVar))
	for k := range envVar {
		keys = append(keys, k)
	}
	unPrefixedLocationsMap := envkeysToPaths(keys)
	locationsMap := make(map[string]string, len(unPrefixedLocationsMap))
	for key, path := range unPrefixedLocationsMap {
		locationsMap[key] = secretpath.Join(cmd.path.Value(), path)
	}

	if cmd.interactive {
		editor, err := newEditor()
		if err != nil {
			return err
		}
		buildFile(locationsMap, editor)
		edited, err := editor.openAndWait()
		if err != nil {
			return err
		}

		locationsMap, err = buildMap(edited)
		if err != nil {
			return err
		}
	}

	if !cmd.force {
		alreadyExist := make(map[string]struct{})
		var m sync.Mutex
		errGroup, _ := errgroup.WithContext(context.Background())
		for _, path := range locationsMap {
			errGroup.Go(func(path string) func() error {
				return func() error {
					exists, err := client.Secrets().Exists(path)
					if err != nil {
						return err
					}
					if exists {
						m.Lock()
						alreadyExist[path] = struct{}{}
						m.Unlock()
					}
					return nil
				}
			}(path))
		}
		err = errGroup.Wait()
		if err != nil {
			return err
		}

		if len(alreadyExist) > 0 {
			_, promptOut, err := cmd.io.Prompts()
			if err != nil {
				errMessage := "secrets already exist at the following locations: "
				for location := range alreadyExist {
					errMessage += location + ", "
				}
				errMessage = errMessage[:len(errMessage)-2]
				return fmt.Errorf(errMessage)
			}

			fmt.Fprintln(promptOut, "secrets already exist at the following locations:")
			for location := range alreadyExist {
				fmt.Fprintln(promptOut, location)
			}

			confirmed, err := ui.AskYesNo(cmd.io, fmt.Sprintf("This import process will overwrite these secrets. Do you wish to continue?"), ui.DefaultNo)

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

	errGroup, _ := errgroup.WithContext(context.Background())
	for envVarKey, secretPath := range locationsMap {
		errGroup.Go(func(envVarKey, secretPath string) func() error {
			return func() error {
				envVarValue, ok := envVar[envVarKey]
				if !ok {
					return fmt.Errorf("key not found in .env file: %s", envVarKey)
				}

				err = client.Dirs().CreateAll(secretpath.Parent(secretPath))
				if err != nil {
					return fmt.Errorf("creating parent directories for %s: %s", secretPath, err)
				}

				_, err = client.Secrets().Write(secretPath, []byte(envVarValue))
				if err != nil {
					return err
				}

				return nil
			}
		}(envVarKey, secretPath))
	}
	err = errGroup.Wait()
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(cmd.io.Output(), "Transfer complete! The secrets have been written to %s.\n", cmd.path.String())
	if err != nil {
		return err
	}

	return nil
}

type editor struct {
	file *os.File
}

func newEditor() (editor, error) {
	tmpFile, err := ioutil.TempFile(os.TempDir(), "secrethub-")
	if err != nil {
		return editor{}, nil
	}
	return editor{
		file: tmpFile,
	}, nil
}

// openAndWait opens the editors file in an editor and waits
// for the user to exit the editor.
// It returns a reader to read the edited contents of the file.
func (e editor) openAndWait() (io.Reader, error) {
	defer func() {
		_ = os.Remove(e.file.Name())
	}()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "editor"
	}

	cmd := exec.Command(editor, e.file.Name())

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		return nil, err
	}
	err = cmd.Wait()
	if err != nil {
		return nil, err
	}

	return os.Open(e.file.Name())
}

func (e editor) Write(in []byte) (int, error) {
	return e.file.Write(in)
}

func buildFile(locationsMap map[string]string, w io.Writer) {
	tabWriter := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)

	for envVarKey, secretPath := range locationsMap {
		_, _ = fmt.Fprintf(tabWriter, "%s\t=>\t%s\n", envVarKey, secretPath)
	}
	_ = tabWriter.Flush()

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "# Environment variables on the left will be stored in SecretHub at the given path on the right.")
	_, _ = fmt.Fprintln(w, "# You can remove or comment out lines for environment variables you do not want to import.")
	_, _ = fmt.Fprintln(w, "# You can change the path where the secrets are stored for the variables you want to keep.")
	_, _ = fmt.Fprintln(w, "# For example, you can group variables in a directory.")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "# When everything is to your liking, you can save the file and exit the editor to continue.")
}

func buildMap(input io.Reader) (map[string]string, error) {
	scanner := bufio.NewScanner(input)
	locationsMap := make(map[string]string)

	i := 0
	for scanner.Scan() {
		i++
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#") && line != "" {
			split := strings.SplitN(line, "=>", 2)
			if len(split) != 2 {
				if strings.Contains(line, "=>") {
					return nil, fmt.Errorf("could not parse prompt at line %d: '=>' should be followed by a secret path", i)
				}
				return nil, fmt.Errorf("could not parse prompt at line %d: missing '=>'", i)
			}
			locationsMap[strings.TrimSpace(split[0])] = strings.TrimSpace(split[1])
		}
	}
	return locationsMap, nil
}

type envKeyToPath struct {
	key  string
	tail []string
}

// envkeysToPaths maps environment variable keys to paths on SecretHub in
// which to store the secrets the corresponding environment variables
// contain.
// See Test_envkeysToPaths for examples on how envkeysToPaths maps the keys
// to paths.
func envkeysToPaths(envkeys []string) map[string]string {
	keys := make([]envKeyToPath, len(envkeys))
	for i, envkey := range envkeys {
		keys[i] = envKeyToPath{
			key:  envkey,
			tail: strings.Split(strings.ToLower(envkey), "_"),
		}
	}
	res, _ := splittedEnvKeysToPaths(keys)
	return res
}

func splittedEnvKeysToPaths(keys []envKeyToPath) (map[string]string, bool) {
	byHeads := make(map[string][]envKeyToPath)
	for _, key := range keys {
		if len(key.tail) == 0 {
			// If there's no tail, that means one key is completely equal to part of another key.
			// e.g. STRIPE_API, STRIPE_API_KEY
			// In this edge-case we create secrets "api" and "api-key" and we don't create a directory called "api".
			res := make(map[string]string, len(keys))
			for _, key = range keys {
				res[key.key] = strings.Join(key.tail, "-")
			}
			return res, true
		}
		byHeads[key.tail[0]] = append(byHeads[key.tail[0]], envKeyToPath{key: key.key, tail: key.tail[1:]})
	}

	res := make(map[string]string)
	for head, keys := range byHeads {
		if len(keys) > 1 {
			paths, oneDir := splittedEnvKeysToPaths(keys)
			for key, path := range paths {
				if oneDir {
					res[key] = head
					if path != "" {
						// If all secrets starting with this prefix are already in a single directory,
						// we don't want to put that directory into another directory, but instead use
						// a longer name for that directory. For example, we don't want MY_APP prefix
						// to convert to my/app/ directories, but to one single my-app directory.
						res[key] += "-" + path
					}
				} else {
					res[key] = secretpath.Join(head, path)
				}
			}
		} else {
			res[keys[0].key] = strings.Join(append([]string{head}, keys[0].tail...), "-")
		}
	}
	return res, len(byHeads) == 1
}
