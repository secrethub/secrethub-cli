package onepassword

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
)

type OPClient interface {
	IsV2() bool
	CreateVault(name string) error
	CreateItem(vault string, template *ItemTemplate, title string) error
	SetField(vault, item, field, value string) error
	GetFields(vault, item string) (map[string]string, error)
	ExistsVault(vaultName string) (bool, error)
	ExistsItemInVault(vault string, itemName string) (bool, error)
}

func GetOPClient() (OPClient, error) {
	out, err := execOP("--version")
	if err != nil {
		return nil, err
	}

	version := strings.TrimSpace(string(out))

	if strings.HasPrefix(version, "2.") {
		return &OPV2Client{
			version: version,
			isV2:    true,
		}, nil
	}
	return &OPV1Client{
		version: version,
		isV2:    false,
	}, nil
}

type OPV1Client struct {
	version string
	isV2    bool
}

func (op *OPV1Client) IsV2() bool {
	return op.isV2
}

func (op *OPV1Client) CreateVault(name string) error {
	_, err := execOP("create", "vault", name)
	if err != nil {
		return fmt.Errorf("could not create vault '%s': %s", name, err)
	}
	return nil
}

func (op *OPV1Client) CreateItem(vault string, template *ItemTemplate, title string) error {
	jsonTemplate, err := json.Marshal(template)
	if err != nil {
		return err
	}

	encodedTemplate := base64.RawURLEncoding.EncodeToString(jsonTemplate)

	_, err = execOP("create", "item", "apicredential", "--vault="+vault, encodedTemplate, "title="+title)
	return err
}

func (op *OPV1Client) SetField(vault, item, field, value string) error {
	_, err := execOP("edit", "item", item, fmt.Sprintf(`%s=%s`, field, value), "--vault="+vault)
	if err != nil {
		return fmt.Errorf("could not set field '%s'.'%s'.'%s'", vault, item, field)
	}
	return nil
}

// GetFields returns a title-to-value map of the fields from the first section of the given 1Password item.
// The rest of the fields are ignored as the migration tool only stores information in the first
// section of each item.
func (op *OPV1Client) GetFields(vault, item string) (map[string]string, error) {
	opItem := struct {
		Details ItemTemplate `json:"details"`
	}{}
	opItemJSON, err := execOP("get", "item", item, "--vault="+vault)
	if err != nil {
		return nil, fmt.Errorf("could not get item '%s'.'%s' from 1Password: %s", vault, item, err)
	}
	err = json.Unmarshal(opItemJSON, &opItem)
	if err != nil {
		return nil, fmt.Errorf("unexpected format of 1Password item in `op get item` command output: %s", err)
	}

	fields := make(map[string]string, len(opItem.Details.Sections[0].Fields))
	for _, field := range opItem.Details.Sections[0].Fields {
		fields[field.Title] = field.Value
	}
	return fields, nil
}

func NewItemTemplate() *ItemTemplate {
	return &ItemTemplate{
		Sections: []sectionTemplate{
			{
				Name:  "",
				Title: "",
			},
		},
	}
}

type ItemTemplate struct {
	Sections []sectionTemplate `json:"sections"`
}

type sectionTemplate struct {
	Name   string              `json:"name"`
	Title  string              `json:"title"`
	Fields []itemFieldTemplate `json:"fields"`
}

func (tpl *ItemTemplate) AddField(name, value string, concealed bool) {
	designation := "concealed"
	if !concealed {
		designation = "string"
	}

	tpl.Sections[0].Fields = append(tpl.Sections[0].Fields, itemFieldTemplate{
		Designation: designation,
		Name:        name,
		Title:       name,
		Value:       value,
	})
}

type itemFieldTemplate struct {
	Designation string `json:"k"`
	Name        string `json:"n"`
	Title       string `json:"t"`
	Value       string `json:"v"`
}

func execOP(args ...string) ([]byte, error) {
	command := exec.Command("op1", args...)
	command.Stderr = os.Stderr
	var out bytes.Buffer
	command.Stdout = &out

	err := command.Run()
	if err != nil {
		return nil, fmt.Errorf("1password: op %s: %s", strings.Join(args, " "), err)
	}

	return out.Bytes(), nil
}

func EnsureSignedIn() error {
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "OP_SESSION") {
			return nil
		}
	}

	return fmt.Errorf("OP_SESSION environment variable not found, run `eval $(op signin)` to set one")
}

func opConfigDirPath() (string, error) {
	xdgConfigHome, _ := os.LookupEnv("XDG_CONFIG_HOME")
	home, _ := homedir.Dir()

	// Inspect possible config directories in reverse order of priority.
	// This code has been taken from the op cli's source code.
	configDirPaths := []string{}
	if home != "" {
		// Legacy home
		configDirPaths = append(configDirPaths, filepath.Join(home, ".op"))
	}
	if xdgConfigHome != "" {
		// Legacy xdg
		configDirPaths = append(configDirPaths, filepath.Join(xdgConfigHome, ".op"))
	}
	if home != "" {
		// New home
		configDirPaths = append(configDirPaths, filepath.Join(home, ".config", "op"))
	}
	if xdgConfigHome != "" {
		// New xdg
		configDirPaths = append(configDirPaths, filepath.Join(xdgConfigHome, "op"))
	}

	for _, configDir := range configDirPaths {
		fileInfo, err := os.Stat(configDir)
		if err == nil && fileInfo.IsDir() {
			return configDir, nil
		}
	}

	// If we reach this point then none of those directories exist (op is executed
	// for the first time). Default to the last entry in the list.
	if len(configDirPaths) > 0 {
		return configDirPaths[len(configDirPaths)-1], nil
	}

	return "", fmt.Errorf("unable to determine location of config directory")
}

func GetSignInAddress() (string, error) {
	path, err := opConfigDirPath()
	if err != nil {
		return "", err
	}
	bytes, err := ioutil.ReadFile(filepath.Join(path, "config"))
	if err != nil {
		return "", fmt.Errorf("could not read 1password config file at %s", path)
	}

	config := struct {
		LatestSignin string `json:"latest_signin"`
		Accounts     []struct {
			Shorthand string `json:"shorthand"`
			URL       string `json:"url"`
		} `json:"accounts"`
	}{}

	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return "", fmt.Errorf("unexpected format of 1password config file at %s", path)
	}

	for _, account := range config.Accounts {
		if account.Shorthand == config.LatestSignin {
			return account.URL, nil
		}
	}

	return "", fmt.Errorf("unexpected format of 1password config file at %s: missing account entry for latest used account", path)
}

func (op *OPV1Client) ExistsVault(vaultName string) (bool, error) {
	vaultsBytes, err := execOP("list", "vaults")
	if err != nil {
		return false, fmt.Errorf("could not list vaults: %s", err)
	}

	vaultsJSON := make([]struct {
		UUID string `json:"uuid"`
		Name string `json:"name"`
	}, 0)

	err = json.Unmarshal(vaultsBytes, &vaultsJSON)
	if err != nil {
		return false, fmt.Errorf("unexpected format of `op list vaults`: %s", vaultsBytes)
	}

	for _, vault := range vaultsJSON {
		if vault.Name == vaultName {
			return true, nil
		}
	}

	return false, nil
}

func (op *OPV1Client) ExistsItemInVault(vault string, itemName string) (bool, error) {
	itemsBytes, err := execOP("list", "items", "--vault", vault)
	if err != nil {
		return false, fmt.Errorf("could not list items in vault %s: %s", vault, err)
	}

	itemsJSON := make([]struct {
		Overview struct {
			Title string `json:"title"`
		} `json:"overview"`
	}, 0)

	err = json.Unmarshal(itemsBytes, &itemsJSON)
	if err != nil {
		return false, fmt.Errorf("unexpected format of `op list items`: %s", itemsBytes)
	}

	for _, item := range itemsJSON {
		if item.Overview.Title == itemName {
			return true, nil
		}
	}

	return false, nil
}
