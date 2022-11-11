package onepassword

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

type OPV1CLI struct{}

func (op *OPV1CLI) IsV2() bool {
	return false
}

func (op *OPV1CLI) CreateVault(name string) error {
	_, err := execOP("create", "vault", name)
	if err != nil {
		return fmt.Errorf("could not create vault '%s': %s", name, err)
	}
	return nil
}

func (op *OPV1CLI) CreateItem(vault string, template ItemTemplate, title string) error {
	jsonTemplate, err := json.Marshal(template)
	if err != nil {
		return err
	}

	encodedTemplate := base64.RawURLEncoding.EncodeToString(jsonTemplate)

	_, err = execOP("create", "item", "apicredential", "--vault="+vault, encodedTemplate, "title="+title)
	return err
}

func (op *OPV1CLI) SetField(vault, item, field, value string) error {
	_, err := execOP("edit", "item", item, fmt.Sprintf(`%s=%s`, field, value), "--vault="+vault)
	if err != nil {
		return fmt.Errorf("could not set field '%s'.'%s'.'%s'", vault, item, field)
	}
	return nil
}

// GetFields returns a title-to-value map of the fields from the first section of the given 1Password item.
// The rest of the fields are ignored as the migration tool only stores information in the first
// section of each item.
func (op *OPV1CLI) GetFields(vault, item string) (map[string]string, error) {
	opItem := struct {
		Details v1ItemTemplate `json:"details"`
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

func (op *OPV1CLI) ExistsVault(vaultName string) (bool, error) {
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

func (op *OPV1CLI) ExistsItemInVault(vault string, itemName string) (bool, error) {
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
