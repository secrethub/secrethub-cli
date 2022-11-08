package onepassword

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
)

type OPCLI interface {
	IsV2() bool
	CreateVault(name string) error
	CreateItem(vault string, template ItemTemplate, title string) error
	SetField(vault, item, field, value string) error
	GetFields(vault, item string) (map[string]string, error)
	ExistsVault(vaultName string) (bool, error)
	ExistsItemInVault(vault string, itemName string) (bool, error)
}

func GetOPClient() (OPCLI, error) {
	out, err := execOP("--version")
	if err != nil {
		return nil, err
	}

	version := strings.TrimSpace(string(out))

	if strings.HasPrefix(version, "2.") {
		return &OPV2CLI{}, nil
	} else if strings.HasPrefix(version, "1.") {
		return &OPV1CLI{}, nil
	}
	return nil, fmt.Errorf("1password: op version not recognized")
}

func NewItemTemplate(client OPCLI) ItemTemplate {
	if client.IsV2() {
		return &v2ItemTemplate{}
	}
	return &v1ItemTemplate{
		Sections: []sectionTemplate{
			{
				Name:  "",
				Title: "",
			},
		},
	}
}

type ItemTemplate interface {
	AddField(name, value string, concealed bool)
}

type v1ItemTemplate struct {
	Sections []sectionTemplate `json:"sections"`
}

type sectionTemplate struct {
	Name   string              `json:"name"`
	Title  string              `json:"title"`
	Fields []itemFieldTemplate `json:"fields"`
}

func (tpl *v1ItemTemplate) AddField(name, value string, concealed bool) {
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
	command := exec.Command("op", args...)
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
