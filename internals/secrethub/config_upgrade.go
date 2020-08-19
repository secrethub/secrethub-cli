package secrethub

import "github.com/secrethub/secrethub-cli/internals/secrethub/command"

// Errors
var (
	ErrConfigUpgradeDropped = errMain.Code("config_upgrade_dropped").Error("This command no longer exists. config update-passphrase can be used to change the passphrase of your credential. To upgrade old configuration files, use a CLI with a version <= v0.25")
)

type ConfigUpgradeCommand struct{}

func NewConfigUpgradeCommand() *ConfigUpgradeCommand {
	return &ConfigUpgradeCommand{}
}

func (cmd *ConfigUpgradeCommand) Register(r command.Registerer) {
	clause := r.Command("upgrade", "Upgrade your .secrethub configuration directory. This can be useful to migrate to a newer version of the configuration files.").Hidden()
	command.BindAction(clause, nil, cmd.Run)
}

func (cmd *ConfigUpgradeCommand) Run() error {
	return ErrConfigUpgradeDropped
}
