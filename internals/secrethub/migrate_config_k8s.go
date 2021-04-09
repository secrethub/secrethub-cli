package secrethub

import (
	"fmt"
	"os"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"gopkg.in/yaml.v2"
)

func (cmd *MigrateConfigK8sCommand) Run() error {
	plan, err := getPlan(cmd.planFile)
	if err != nil {
		return err
	}

	k8sSpecs := make([]itemK8sSpec, 0)

	if len(cmd.vaults) == 0 {
		for _, vault := range plan.vaults {
			for _, item := range vault.Items {
				k8sSpecs = append(k8sSpecs, itemK8sSpec{
					vaultName: vault.Name,
					itemName:  item.Name,
				})
			}
		}
	} else {
		for _, vaultName := range cmd.vaults {
			vault := plan.vaults[vaultName]
			for _, item := range vault.Items {
				k8sSpecs = append(k8sSpecs, itemK8sSpec{
					vaultName: vault.Name,
					itemName:  item.Name,
				})
			}
		}
	}

	if _, err := os.Stat(cmd.outFile); !os.IsNotExist(err) {
		proceed, err := ui.AskYesNo(cmd.io, fmt.Sprintf("%s file already exists. Do you want to overwrite it?", cmd.outFile), ui.DefaultNo)
		if err != nil {
			return fmt.Errorf("output file '%s' already exists", cmd.outFile)
		}
		if !proceed {
			fmt.Fprintln(cmd.io.Output(), "Aborting.")
			return nil
		}
	}
	file, err := os.OpenFile(cmd.outFile, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("could not open %s: %s", cmd.outFile, err)
	}
	defer file.Close()

	for i, item := range k8sSpecs {
		bytes, err := yaml.Marshal(&item)
		if err != nil {
			return fmt.Errorf("could not marshal %s/%s to YAML: %s", item.vaultName, item.itemName, err)
		}
		_, err = file.Write(bytes)
		if err != nil {
			return fmt.Errorf("could not write spec to %s: %s", cmd.outFile, err)
		}
		if i != len(k8sSpecs)-1 {
			_, err := file.Write([]byte("---\n"))
			if err != nil {
				return fmt.Errorf("could not write spec to %s: %s", cmd.outFile, err)
			}
		}
	}

	fmt.Fprintln(cmd.io.Output(), "Successfully generated secrets.yml file.")

	return nil
}

type MigrateConfigK8sCommand struct {
	io ui.IO

	planFile string
	outFile  string
	vaults   []string
}

func NewMigrateConfigK8sCommand(io ui.IO) *MigrateConfigK8sCommand {
	return &MigrateConfigK8sCommand{
		io: io,
	}
}

func (cmd *MigrateConfigK8sCommand) Register(r cli.Registerer) {
	clause := r.Command("k8s", "Create a yaml spec for your Kubernetes secrets.")

	clause.Flags().StringVar(&cmd.planFile, "plan-file", defaultPlanPath, "Path to the file used to migrate your secrets.")
	clause.Flags().StringVar(&cmd.outFile, "out-file", "secrets.yml", "The filename of the spec to create.")
	clause.Flags().StringArrayVar(&cmd.vaults, "vault", []string{}, "Include only items from these vaults.")

	clause.BindAction(cmd.Run)
}

type itemK8sSpec struct {
	itemName  string
	vaultName string
}

func (item itemK8sSpec) MarshalYAML() (interface{}, error) {
	return struct {
		APIVersion string `yaml:"apiVersion"`
		Kind       string `yaml:"kind"`
		MetaData   struct {
			Name string `yaml:"name"`
		} `yaml:"metadata"`
		Spec struct {
			ItemPath string `yaml:"itemPath"`
		} `yaml:"spec"`
	}{
		APIVersion: "onepassword.com/v1",
		Kind:       "OnePasswordItem",
		MetaData: struct {
			Name string `yaml:"name"`
		}{Name: item.itemName},
		Spec: struct {
			ItemPath string `yaml:"itemPath"`
		}{ItemPath: fmt.Sprintf("vaults/%s/items/%s", item.vaultName, item.itemName)},
	}, nil
}
