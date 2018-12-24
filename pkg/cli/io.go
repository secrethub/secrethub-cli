package cli

import (
	"encoding/json"

	"github.com/secrethub/secrethub-go/internals/errio"
)

// PrettyJSON returns a 4-space indented JSON text.
// Can be useful for printing out structs.
func PrettyJSON(data interface{}) (string, error) {
	pretty, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return "", errio.Error(err)
	}

	return string(pretty), nil
}
