package secrethub

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
)

var regexpSecretTemplatePath = regexp.MustCompile(`[A-Za-z0-9_\.\-\$\{\}]{2,}\/[A-Za-z0-9_\.\-\$\{\}]{2,}\/[A-Za-z0-9_\.\-\$\{\}\/]{2,}`)
var regexpSecretTemplateTags = regexp.MustCompile(`{{(\s)*?(` + regexpSecretTemplatePath.String() + `)(\s)*?}}`)

func (cmd *MigrateConfigTemplatesCommand) Run() error {
	plan, err := getPlan(cmd.planFile)
	if err != nil {
		return err
	}

	vars, err := parseVarPossibilities(cmd.vars)
	if err != nil {
		return err
	}

	refMapping := newReferenceMapping(plan)
	_, err = refMapping.addVarPossibilities(vars)
	if err != nil {
		return err
	}
	refMapping.stripSecretHubURIScheme()

	for _, filepath := range cmd.inFiles {
		file, err := os.Open(filepath)
		if err != nil {
			return ErrReadFile(filepath, err)
		}
		defer file.Close()

		replaceCount, err := migrateTemplateTags(file, file, refMapping, "{{ %s }}")
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.io.Output(), "Updated %s with %d op:// references\n", filepath, len(replaceCount))
	}

	return nil
}

func migrateTemplateTags(inFile io.Reader, outFile io.Writer, mapping referenceMapping, formatString string) ([]string, error) {
	raw, err := io.ReadAll(inFile)
	if err != nil {
		return nil, ErrReadFile(inFile, err)
	}

	var hits, misses []string
	output := regexpSecretTemplateTags.ReplaceAllStringFunc(string(raw), func(templateTag string) string {
		path := regexpSecretTemplatePath.FindString(templateTag)
		if path == "" {
			misses = append(misses, templateTag)
			return ""
		}

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

		return nil, fmt.Errorf(errMsg)
	}

	_, err = io.WriteString(outFile, output)
	if err != nil {
		return nil, err
	}

	return hits, nil
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

func parseVarPossibilities(unparsed map[string]string) (map[string][]string, error) {
	result := make(map[string][]string)
	for varname, optionsStr := range unparsed {
		options := strings.Split(optionsStr, ",")
		for i, v := range options {
			options[i] = strings.TrimSpace(v)
		}

		result[varname] = options
	}

	return result, nil
}
