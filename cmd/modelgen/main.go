package main

import (
	"fmt"

	"github.com/secrethub/secrethub-cli/internals/secrethub"
	"gopkg.in/yaml.v2"
)

func main() {
	model := secrethub.NewApp().Version(secrethub.Version, secrethub.Commit).Model()
	yml, err := yaml.Marshal(model)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(yml))
}
