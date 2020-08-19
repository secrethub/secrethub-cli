package secrethub

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/secrethub/secrethub-go/internals/errio"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
	"github.com/secrethub/secrethub-cli/internals/winrm"
	"github.com/spf13/cobra"
)

// DefaultServiceConfigFilemode configures the filemode used for writing service configuration files.
// When changing this, make sure to update the description and default values of the flags.
var DefaultServiceConfigFilemode os.FileMode = 0440

// Errors
var (
	errService = errio.Namespace("service")

	ErrNoValueProvided = errService.Code("no_value_provided").Error("request value was not provided")

	ErrUnknownGroupDocker = errService.Code("unknown_group_docker").Error("could not find the user group docker. Make sure Docker is installed.")
	ErrUserNotRoot        = errService.Code("not_root").Error("this command can only be executed as root")
	ErrChownFailed        = errService.Code("chown_failed").ErrorPref("cannot change owner of %s: %s")

	ErrUnknownAuthType      = errService.Code("unknown_auth_type").Error("authentication type must be basic or cert")
	ErrCouldNotReadCA       = errService.Code("ca_read_error").ErrorPref("could not read CA file: %s")
	ErrCouldNotReadCert     = errService.Code("cert_read_error").ErrorPref("could not read Cert file: %s")
	ErrCouldNotReadKey      = errService.Code("key_read_error").ErrorPref("could not read Key file: %s")
	ErrIncorrectWinRMScheme = errService.Code("incorrect_winrm_scheme").ErrorPref("Only http or https allowed. The scheme supplied: %s")
)

// ServiceDeployWinRmCommand creates a service and installs the configuration using WinRM.
type ServiceDeployWinRmCommand struct {
	resourceURI cli.URLArgValue
	authType    string
	username    string
	password    string
	clientCert  string
	clientKey   string
	caCert      string
	noVerify    bool
	io          ui.IO
}

// NewServiceDeployWinRmCommand creates a new ServiceDeployWinRmCommand.
func NewServiceDeployWinRmCommand(io ui.IO) *ServiceDeployWinRmCommand {
	return &ServiceDeployWinRmCommand{
		io: io,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *ServiceDeployWinRmCommand) Register(r command.Registerer) {
	clause := r.Command("winrm", "Read a service account configuration from stdin and deploy it to a running instance with WinRM. The instance needs to be reachable, have WinRM enabled, and have PowerShell installed.")
	clause.Cmd.Args = cobra.ExactValidArgs(1)
	//clause.Arg("resource-uri", "Hostname, optional connection protocol and port of the host ([http[s]://]<host>[:<port>]). This defaults to https and port 5986.").Required().URLVar(&cmd.resourceURI)
	clause.StringVar(&cmd.authType, "auth-type", "basic", "Authentication type (basic/cert)", true, false)
	_ = clause.Cmd.RegisterFlagCompletionFunc("auth-type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"basic", "cert"}, cobra.ShellCompDirectiveDefault
	})
	clause.StringVar(&cmd.username, "username", "", "The username used for logging in when authentication type is basic. Is asked if not supplied.", true, false)
	clause.StringVar(&cmd.password, "password", "", "The password used for logging in when authentication type is basic. Is asked if not supplied.", true, false)
	clause.StringVar(&cmd.clientCert, "client-cert", "", "Path to client certificate used for certificate authentication.", true, false)
	clause.StringVar(&cmd.clientKey, "client-key", "", "Path to client key used for certificate authentication.", true, false)
	clause.StringVar(&cmd.caCert, "ca-cert", "", "Path to CA certificate used to verify server TLS certificate.", true, false)
	clause.BoolVar(&cmd.noVerify, "insecure-no-verify-cert", false, "Do not verify server TLS certificate (insecure).", true, false)

	command.BindAction(clause, []cli.ArgValue{&cmd.resourceURI}, cmd.Run)
}

// Run creates a service and installs the configuration using WinRM.
func (cmd *ServiceDeployWinRmCommand) Run() error {
	var err error

	var port int
	if cmd.resourceURI.Param.Port() != "" {
		port, err = strconv.Atoi(cmd.resourceURI.Param.Port())
		if err != nil {
			return err
		}
	}

	// Retrieve the WinRM port, default to 5986.
	if port == 0 {
		if cmd.resourceURI.Param.Scheme == "http" {
			port = 5985
		} else {
			port = 5986
		}
	}

	var caCert []byte
	if cmd.caCert != "" {
		caCert, err = ioutil.ReadFile(cmd.caCert)
		if err != nil {
			return ErrCouldNotReadCA(err)
		}
	}

	// check the schema
	https, err := cmd.checkWinRMTLS()
	if err != nil {
		return err
	}

	skipVerifyCert := cmd.checkWinRMVerifyCert()

	config := &winrm.Config{
		Host: cmd.resourceURI.Param.Hostname(),
		Port: port,

		HTTPS: https,

		SkipVerifyCert: skipVerifyCert,

		CaCert: caCert,
	}

	var client *winrm.Client
	// Construct the credential portion of the WinRM config.
	switch cmd.authType {
	case "basic":
		if cmd.username == "" {
			cmd.username, err = ui.Ask(cmd.io, "What username do you want to use to connect?\n")
			if err != nil {
				return err
			}
		}

		if cmd.password == "" {
			cmd.password, err = ui.AskSecret(cmd.io, fmt.Sprintf("What is the password for user %s?\n", cmd.username))
			if err != nil {
				return err
			}
		}

		client, err = winrm.NewBasicClient(config, cmd.username, cmd.password)
		if err != nil {
			return err
		}
	case "cert":
		var clientCert, clientKey []byte
		if cmd.clientCert != "" {
			clientCert, err = ioutil.ReadFile(cmd.clientCert)
			if err != nil {
				return ErrCouldNotReadCert(err)
			}
		}

		if cmd.clientKey != "" {
			clientKey, err = ioutil.ReadFile(cmd.clientKey)
			if err != nil {
				return ErrCouldNotReadKey(err)
			}
		}

		client, err = winrm.NewCertClient(config, clientCert, clientKey)
		if err != nil {
			return err
		}
	default:
		return ErrUnknownAuthType
	}

	// Verify the server certificate if we're using TLS.
	// If it is not trusted, the user is asked to confirm the server's identity.
	if https {
		err := client.GetServerCert(cmd.io)
		if err != nil {
			return err
		}
	}

	// Get the path to place the credential file in.
	destinationPath := fmt.Sprintf("$HOME\\%s\\%s", defaultProfileDirName, defaultCredentialFilename)

	deployer := newWindowsDeployer(client, destinationPath)

	if !cmd.io.IsInputPiped() {
		return ErrNoDataOnStdin
	}

	credential, err := ioutil.ReadAll(cmd.io.Input())
	if err != nil {
		return err
	}

	// Copy the config to the host.
	fmt.Fprintln(cmd.io.Output(), "Deploying configuration...")
	err = deployer.configure(credential)
	if err != nil {
		return err
	}

	fmt.Fprintln(cmd.io.Output(), "Deploy complete! The service account can now be used to connect to SecretHub from the host.")

	return nil
}

// checkWinRMTLS checks if the given schema corresponds to the given CLI flags.
func (cmd *ServiceDeployWinRmCommand) checkWinRMTLS() (bool, error) {
	if cmd.resourceURI.Param.Scheme == "http" {
		fmt.Fprintln(cmd.io.Output(), "WARNING: insecure no tls flag is set! We recommend to always use TLS.")
		return false, nil
	}

	if cmd.resourceURI.Param.Scheme == "https" {
		return true, nil
	}

	if cmd.resourceURI.Param.Scheme == "" {
		return true, nil
	}

	return true, ErrIncorrectWinRMScheme(cmd.resourceURI.Param.Scheme)
}

// checkWinRMVerifyCert checks if the given schema corresponds to the given CLI flags.
func (cmd *ServiceDeployWinRmCommand) checkWinRMVerifyCert() bool {
	if cmd.noVerify {
		fmt.Fprintln(cmd.io.Output(), "WARNING: insecure no verify cert flag is set! We recommend to always verify the certificate.")
		return true
	}

	return false
}

// deployer is an interface that can be used to install the secrets client
// and copy a service configuration to a target machine.
type deployer interface {
	configure([]byte) error
}

// windowsDeployer deploy a secrets service to a Windows host.
type windowsDeployer struct {
	conn *winrm.Client
	path string
}

// newWindowsDeployer creates a windowsDeployer using a WinRM connection.
func newWindowsDeployer(conn *winrm.Client, path string) deployer {
	wd := windowsDeployer{
		conn: conn,
		path: path,
	}

	return wd
}

// configure copies a service credential to a Windows host.
func (wd windowsDeployer) configure(token []byte) error {
	r := bytes.NewBuffer(token)
	copyProgress := make(chan int)

	err := wd.conn.CopyFile(r, wd.path, copyProgress)
	if err != nil {
		return err
	}

	return nil
}
