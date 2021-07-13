package secrethub

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
)

var regexpSecretsRef = regexp.MustCompile(`([^A-Za-z0-9_\.-]|^)(secrethub:\/\/[A-Za-z0-9_\.\-]{2,}\/[A-Za-z0-9_\.\-]{2,}\/[A-Za-z0-9_\.\-\/]{2,})([^A-Za-z0-9_\.-]|$)`)

func (cmd *MigrateConfigReferencesCommand) Run() error {
	plan, err := getPlan(cmd.planFile)
	if err != nil {
		return err
	}

	refMapping := newReferenceMapping(plan)
	for _, filepath := range cmd.inFiles {
		replaceCount, err := migrateReferences(filepath, filepath, refMapping)
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.io.Output(), "Updated %s with %d op:// references\n", filepath, len(replaceCount))
	}

	return nil
}

func migrateReferences(inFile string, outFile string, mapping referenceMapping) ([]string, error) {
	raw, err := ioutil.ReadFile(inFile)
	if err != nil {
		return nil, ErrReadFile(inFile, err)
	}

	var hits, misses []string
	output := regexpSecretsRef.ReplaceAllStringFunc(string(raw), func(match string) string {
		submatches := regexpSecretsRef.FindStringSubmatch(match)[1:]

		matchIndexRef := 1
		secretHubRef := submatches[matchIndexRef]

		opRef, ok := mapping[secretHubRef]
		if !ok {
			misses = append(misses, secretHubRef)
			return secretHubRef
		}

		hits = append(hits, opRef)

		submatches[matchIndexRef] = opRef
		return strings.Join(submatches, "")
	})

	if len(misses) != 0 {
		return nil, fmt.Errorf("no 1Password equivalent present in your migration plan for the following secrets:\n- %s", strings.Join(misses, "\n- "))
	}

	err = os.WriteFile(outFile, []byte(output), 0666)
	if err != nil {
		return nil, err
	}

	return hits, nil
}

type MigrateConfigReferencesCommand struct {
	io ui.IO

	inFiles  cli.StringListValue
	planFile string
}

func NewMigrateConfigReferencesCommand(io ui.IO) *MigrateConfigReferencesCommand {
	return &MigrateConfigReferencesCommand{
		io: io,
	}
}

func (cmd *MigrateConfigReferencesCommand) Register(r cli.Registerer) {
	clause := r.Command("references", "Migrate secrethub:// references in configuration code to 1Password op:// references.")
	clause.Flags().StringVar(&cmd.planFile, "plan-file", defaultPlanPath, "Path to the file used to migrate your secrets.")
	clause.BindArgumentsArr(cli.Argument{Value: &cmd.inFiles, Name: "in-file", Required: true, Placeholder: "<filepath>...", Description: "The paths to one or more files you'd like to migrate."})

	clause.BindAction(cmd.Run)
}
