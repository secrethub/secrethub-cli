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

var regexpSecretTemplatePath = regexp.MustCompile(`[A-Za-z0-9_\.\-\$\{\}]{2,}\/[A-Za-z0-9_\.\-\$\{\}]{2,}\/[A-Za-z0-9_\.\-\$\{\}\/]{2,}`)
var regexpSecretTemplateTags = regexp.MustCompile(`{{\s*?(` + regexpSecretTemplatePath.String() + `)\s*?}}`)

func (cmd *MigrateConfigTemplatesCommand) Run() error {
	plan, err := getPlan(cmd.planFile)
	if err != nil {
		return err
	}

	vars := parseVarPossibilities(cmd.vars)
	refMapping := newReferenceMapping(plan)
	err = refMapping.addVarPossibilities(vars)
	if err != nil {
		return err
	}

	refMapping.stripSecretHubURIScheme()

	for _, filepath := range cmd.inFiles {
		inFileContents, err := ioutil.ReadFile(filepath)
		if err != nil {
			return ErrReadFile(filepath, err)
		}

		output, replaceCount, err := migrateTemplateTags(string(inFileContents), refMapping, "{{ %s }}")
		if err != nil {
			return err
		}

		inFileInfo, err := os.Stat(filepath)
		if err != nil {
			return ErrReadFile(filepath, err)
		}

		err = ioutil.WriteFile(filepath, []byte(output), inFileInfo.Mode())
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.io.Output(), "Updated %s with %d op:// references\n", filepath, replaceCount)
	}

	return nil
}

func migrateTemplateTags(inFileContents string, mapping referenceMapping, formatString string) (string, int, error) {
	var hits, misses []string
	output := regexpSecretTemplateTags.ReplaceAllStringFunc(inFileContents, func(templateTag string) string {
		path := regexpSecretTemplateTags.FindStringSubmatch(templateTag)[1]

		opRef, ok := mapping[path]
		if !ok {
			misses = append(misses, path)
			return path
		}

		hits = append(hits, opRef)
		return fmt.Sprintf(formatString, opRef)
	})

	if len(misses) != 0 {
		errMsg := fmt.Sprintf("no 1Password equivalent present in your migration plan for the following secrets:\n- %s", strings.Join(misses, "\n- "))

		possiblyMissingVar := false
		for _, miss := range misses {
			if strings.Contains(miss, "$") {
				possiblyMissingVar = true
				break
			}
		}
		if possiblyMissingVar {
			errMsg += "\nDid you specify every possible value for your template variables? E.g. --var varname1=a,b,c,d --var varname2=x,y,z"
		}

		return "", 0, fmt.Errorf(errMsg)
	}

	return output, len(hits), nil
}

type MigrateConfigTemplatesCommand struct {
	io ui.IO

	inFiles cli.StringListValue

	planFile string
	vars     map[string]string
}

func NewMigrateConfigTemplatesCommand(io ui.IO) *MigrateConfigTemplatesCommand {
	return &MigrateConfigTemplatesCommand{
		io: io,
	}
}

func (cmd *MigrateConfigTemplatesCommand) Register(r cli.Registerer) {
	clause := r.Command("templates", "Migrate config file templates by turning SecretHub paths into 1Password op:// references.")
	clause.Flags().StringVar(&cmd.planFile, "plan-file", defaultPlanPath, "Path to the file used to migrate your secrets.")
	clause.Flags().StringToStringVarP(&cmd.vars, "var", "v", nil, "Define the possible values for a template variable, e.g. --var env=dev,staging,prod --var region=us-east-1,eu-west-1")
	clause.BindArgumentsArr(cli.Argument{Value: &cmd.inFiles, Name: "in-file", Required: true, Placeholder: "<config-file-path>...", Description: "The paths to one or more config template files you'd like to migrate."})

	clause.BindAction(cmd.Run)
}

func parseVarPossibilities(unparsed map[string]string) map[string][]string {
	result := make(map[string][]string)
	for varname, optionsStr := range unparsed {
		options := strings.Split(optionsStr, ",")
		for i, v := range options {
			options[i] = strings.TrimSpace(v)
		}

		result[varname] = options
	}

	return result
}
