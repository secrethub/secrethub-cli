package secrethub

import (
	"fmt"
	"strings"

	raven "github.com/getsentry/raven-go"
	"github.com/keylockerbv/secrethub-cli/pkg/ui"
	"github.com/keylockerbv/secrethub-cli/pkg/cli"
	logging "github.com/op/go-logging"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
)

const (
	// ApplicationName is the name of the command-line application.
	ApplicationName = "secrethub"
	logFormat       = `%{color}%{level:.4s} ▶ %{color:reset} %{message}`
	logModule       = "log"
)

var (
	// logger handles logging statements at different levels
	logger = logging.MustGetLogger(logModule)
)

// Errors
var (
	errMain       = errio.Namespace(ApplicationName)
	ErrParseError = errio.Namespace("cli").Code("parse_error")

	ErrMustBeUser        = errMain.Code("must_be_user").Error("you must be a user to perform this command")
	ErrCannotFindHomeDir = errMain.Code("cannot_find_home_dir").ErrorPref(
		"cannot find your home directory: %s\n\n" + // some dirty magic to still end up with a format string.
			fmt.Sprintf(
				"Usually this means your home directory is on a different volume/disk, which is common for e.g. managed workstations. "+
					"The CLI automatically attempts to create a %s config directory in your home folder. "+
					"Use the --config-dir flag or %s_CONFIG_DIR environment variable to place the configuration files in a custom location.",
				defaultProfileDirName,
				strings.ToUpper(ApplicationName),
			),
	)
	ErrInvalidConfigDirFlag       = errMain.Code("invalid_config_dir").Errorf("the path to the SecretHub configuration directory must be an absolute path and is configured with the --config-dir flag or %s_CONFIG_DIR environment variable", strings.ToUpper(ApplicationName))
	ErrRepoNameTooShort           = errMain.Code("repo_name_too_short").Error("repository names should be at least 3 characters long")
	ErrOtherNamespaceNotSupported = errMain.Code("other_namespace_not_supported").Error("creating a repository for another namespace is not yet supported")
	ErrMissingFlags               = errMain.Code("missing_flags").Error("force flag set but not all required flags set")
	ErrForceWithoutPipe           = errMain.Code("force_without_pipe").Error("force flag can only be used in conjunction with piped input")
	ErrCannotDoWithoutForce       = errMain.Code("cannot_do_without_force").Error(
		"cannot perform this action without confirmation or a --force flag.\n\n" +
			"This usually happens when you run the command in a non-Unix terminal and pipe either the input or output of the command. " +
			"If you are sure you want to perform this action, run the same command with the --force or -f flag.")
	ErrSecretAlreadyExists      = errMain.Code("already_exists").Error("the secret already exists. To overwrite it, run the same command with the --force or -f flag")
	ErrSecretNotFound           = errMain.Code("secret_not_found").ErrorPref("the secret %s does not exist")
	ErrSecretVersionNotFound    = errMain.Code("version_not_found").ErrorPref("version %s of secret %s does not exist")
	ErrResourceNotFound         = errMain.Code("resource_not_found").ErrorPref("the resource at path %s does not exist")
	ErrCannotAuditSecretVersion = errMain.Code("cannot_audit_version").Error("auditing a specific version of a secret is not yet supported")
	ErrCannotAuditDir           = errMain.Code("cannot_audit_dir").Error("auditing a specific directory is not yet supported")
	ErrInvalidAuditActor        = errMain.Code("invalid_audit_actor").Error("received an invalid audit actor")
	ErrInvalidAuditSubject      = errMain.Code("invalid_audit_subject").Error("received an invalid audit subject")
	ErrNoValidRepoOrDirPath     = errMain.Code("no_repo_or_dir").Error("no valid path to a repository or a directory was given")
	ErrNoValidRepoOrSecretPath  = errMain.Code("no_repo_or_secret").Error("no valid path to a repository or a secret was given")
	ErrCannotWrite              = errMain.Code("cannot_write").ErrorPref("cannot write to file at %s: %s")
	ErrCannotGetWorkingDir      = errMain.Code("cannot_get_working_dir").ErrorPref("cannot get the working directory: %s")
	ErrNoDataOnStdin            = errMain.Code("no_data_on_stdin").Error("expected data on stdin but none found")
	ErrFlagsConflict            = errMain.Code("flags_conflict").ErrorPref("these flags cannot be used together: %s")
	ErrFileAlreadyExists        = errMain.Code("file_already_exists").Error("file already exists")
)

// App is the secrethub command-line application.
type App struct {
	credentialStore CredentialStore
	clientFactory   ClientFactory
	cli             *cli.App
	io              ui.IO
	logger          *logging.Logger
}

// Registerer allows others to register commands on it.
type Registerer interface {
	Command(cmd string, help string) *cli.CommandClause
}

// newClientFunc creates a ClientAdapater.
type newClientFunc func() (secrethub.Client, error)

// NewApp creates a new command-line application.
func NewApp() *App {
	io := ui.NewUserIO()
	store := NewCredentialStore(io)
	help := "The SecretHub Command-Line Interface (CLI) is a unified tool to manage your secrets inside SecretHub.\n\n" +
		"The CLI is configurable through command-line flags and environment variables. " +
		"Options set on the command-line take precedence over those set in the environment. " +
		"The format for environment variables is `SECRETHUB_[COMMAND_]FLAG_NAME`."
	return &App{
		cli:             cli.NewApp(ApplicationName, help).Version(secrethub.ClientVersion),
		credentialStore: store,
		clientFactory:   NewClientFactory(store),
		io:              io,
		logger:          cli.NewLogger(logFormat, logModule, false),
	}
}

// Run builds the command-line application, parses the arguments,
// configures global behavior and executes the command given by the args.
func (app *App) Run(args []string) error {
	// Construct the CLI
	err := raven.SetDSN(cli.ClientSentryDSN)
	if err != nil {
		return errio.Error(err)
	}

	RegisterDebugFlag(app.cli, app.logger)
	RegisterMlockFlag(app.cli)
	RegisterColorFlag(app.cli)
	app.credentialStore.Register(app.cli)
	app.clientFactory.Register(app.cli)
	app.registerCommands()

	app.cli.UsageTemplate(DefaultUsageTemplate)

	// Parse also executes the command when parsing is successful.
	_, err = app.cli.Parse(args)
	return err
}

// registerCommands initializes all commands and registers them on the app.
func (app *App) registerCommands() {
	NewACLCommand(app.io, app.clientFactory.NewClient).Register(app.cli)
	NewAccountCommand(app.io, app.clientFactory.NewClient, app.credentialStore).Register(app.cli)

	auditCommand := app.cli.Command("audit", "Show the audit log of all actions on a repository or a secret.")
	NewAuditCommand(app.io, app.clientFactory.NewClient).Register(auditCommand)
	NewAuditRepoCommand(app.io, app.clientFactory.NewClient).Register(auditCommand)
	NewAuditSecretCommand(app.io, app.clientFactory.NewClient).Register(auditCommand)

	NewClearCommand(app.io).Register(app.cli)
	NewClearClipboardCommand(app.logger).Register(app.cli)
	NewConfigCommand(app.io, app.credentialStore).Register(app.cli)
	NewGenerateSecretCommand(app.io, app.clientFactory.NewClient).Register(app.cli)
	NewInjectCommand(app.io, app.clientFactory.NewClient).Register(app.cli)
	NewInspectCommand(app.io, app.clientFactory.NewClient).Register(app.cli)
	NewLsCommand(app.io, app.clientFactory.NewClient).Register(app.cli)
	NewMkDirCommand(app.io, app.clientFactory.NewClient).Register(app.cli)
	NewOrgCommand(app.io, app.clientFactory.NewClient).Register(app.cli)
	NewPrintEnvCommand(app.cli, app.io).Register(app.cli)
	NewReadCommand(app.io, app.clientFactory.NewClient).Register(app.cli)
	NewRepoCommand(app.io, app.clientFactory.NewClient).Register(app.cli)
	NewRmCommand(app.io, app.clientFactory.NewClient).Register(app.cli)
	NewRunCommand(app.clientFactory.NewClient).Register(app.cli)
	NewSetCommand(app.io, app.clientFactory.NewClient).Register(app.cli)
	NewSignUpCommand(app.io, app.clientFactory.NewClient, app.credentialStore).Register(app.cli)
	NewTreeCommand(app.io, app.clientFactory.NewClient).Register(app.cli)
	NewWriteCommand(app.io, app.clientFactory.NewClient).Register(app.cli)
	NewKeyringClearCommand().Register(app.cli)
	NewServiceCommand(app.io, app.clientFactory.NewClient).Register(app.cli)
}
