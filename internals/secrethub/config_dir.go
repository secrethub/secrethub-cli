package secrethub

import "github.com/secrethub/secrethub-go/pkg/secrethub/configdir"

type ConfigDir struct {
	configdir.Dir
}

func (c *ConfigDir) Type() string {
	return "configDir"
}

func (c *ConfigDir) String() string {
	return c.Dir.Path()
}

func (c *ConfigDir) Set(value string) error {
	if value != "" {
		*c = ConfigDir{Dir: configdir.New(value)}
		return nil
	}
	dir, err := configdir.Default()
	if err != nil {
		return err
	}
	*c = ConfigDir{Dir: *dir}
	return nil

}
