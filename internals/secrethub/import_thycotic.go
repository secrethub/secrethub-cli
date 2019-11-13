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

	if len(header) < 2 {
		return fmt.Errorf("csv should have at least 2 columns")
	}

	if header[0] != "SecretName" {
		return fmt.Errorf("first column of csv file should contain the SecretName")
	}

	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("could not read record: %s", err)
		}

		dirPath := record[0]
		if strings.ContainsAny(dirPath, "/") {
			return fmt.Errorf("path %s contains '/' character, which is not allowed; paths should be separated with \\", dirPath)
		}
		dirPath = strings.ReplaceAll(dirPath, "\\", "/")

		err = client.Dirs().CreateAll(dirPath)
		if err != nil {
			return fmt.Errorf("could not create directory: %s", err)
		}

		for i, field := range record[1:] {
			secretPath := secretpath.Join(dirPath, header[i])
			_, err = client.Secrets().Write(secretPath, []byte(field))
			if err != nil {
				return fmt.Errorf("could not write secret: %s", err)
			}
		}
	}

	return nil
}

// Register registers the command and its sub-commands on the provided Registerer.
func (cmd *ImportThycoticCommand) Register(r command.Registerer) {
	clause := r.Command("thycotic", "Import secrets from Thycotic.")
	clause.Arg("file", "Path to CSV export of your Thycotic secrets.").Required().StringVar(&cmd.file)
	command.BindAction(clause, cmd.Run)
}
