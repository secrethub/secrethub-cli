package secrethub

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"

	"github.com/secrethub/secrethub-go/pkg/secretpath"
)

// ImportThycoticCommand handles importing secrets from Thycotic.
type ImportThycoticCommand struct {
	io        ui.IO
	newClient newClientFunc
	file      string
}

// NewImportThycoticCommand creates a new ImportThycoticCommand.
func NewImportThycoticCommand(io ui.IO, newClient newClientFunc) *ImportThycoticCommand {
	return &ImportThycoticCommand{
		io:        io,
		newClient: newClient,
	}
}

func (cmd *ImportThycoticCommand) Run() error {
	if !strings.HasSuffix(cmd.file, ".csv") {
		return fmt.Errorf("currently only csv imports are supported")
	}

	r, err := os.Open(cmd.file)
	if err != nil {
		return fmt.Errorf("could not open file: %s", err)
	}

	csvReader := csv.NewReader(r)
	header, err := csvReader.Read()
	if err != nil {
		return fmt.Errorf("could not read from csv file: %s", err)
	}

	if len(header) < 1 || header[0] != "SecretName" {
		return fmt.Errorf("first column of csv file should contain the SecretName")
	}

	if len(header) < 2 || header[1] != "FolderPath" {
		return fmt.Errorf("second column of csv file should contain the FolderPath")
	}

	if len(header) < 3 {
		return fmt.Errorf("nothing to import; there should be at least one column next to SecretName and FolderPath, that contains secrets to import")
	}

	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	dirs := map[string]struct{}{}
	secrets := map[string][]byte{}

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("could not read record: %s", err)
		}

		dirPath := record[1]
		if strings.ContainsAny(dirPath, "/") {
			return fmt.Errorf("path %s contains '/' character, which is not allowed; paths should be separated with \\", dirPath)
		}
		dirPath = strings.ReplaceAll(dirPath, "\\", "/")

		dirPath = secretpath.Join(dirPath, record[0])

		dirPath = strings.ReplaceAll(dirPath, " ", "_")

		dirs[dirPath] = struct{}{}

		for i, field := range record[1:] {
			secretPath := secretpath.Join(dirPath, strings.ReplaceAll(header[i], " ", "_"))
			_, exists := secrets[secretPath]
			if exists {
				return fmt.Errorf("secret '%s' encountered twice", secretPath)
			}
			secrets[secretPath] = []byte(field)
		}
	}

	for dirPath := range dirs {
		err = client.Dirs().CreateAll(dirPath)
		if err != nil {
			return fmt.Errorf("could not create directory '%s': %s", dirPath, err)
		}
	}

	for secretPath, value := range secrets {
		_, err = client.Secrets().Write(secretPath, value)
		if err != nil {
			return fmt.Errorf("could not write secret '%s': %s", secretPath, err)
		}
	}

	fmt.Fprintf(cmd.io.Stdout(), "Successfully imported %d directories containing %d secrets\n", len(dirs), len(secrets))

	return nil
}

// Register registers the command and its sub-commands on the provided Registerer.
func (cmd *ImportThycoticCommand) Register(r command.Registerer) {
	clause := r.Command("thycotic", "Import secrets from Thycotic.")
	clause.Arg("file", "Path to CSV export of your Thycotic secrets.").Required().StringVar(&cmd.file)
	command.BindAction(clause, cmd.Run)
}
