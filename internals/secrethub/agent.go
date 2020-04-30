package secrethub

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/secrethub/secrethub-cli/internals/agent"
	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"
)

// AccountCommand handles operations on SecretHub accounts.
type AgentCommand struct {
	io              ui.IO
	credentialStore CredentialConfig
	logger          cli.Logger

	kill    bool
	restart bool
	daemon  bool
}

// NewAccountCommand creates a new AccountCommand.
func NewAgentCommand(io ui.IO, credentialStore CredentialConfig, logger cli.Logger) *AgentCommand {
	return &AgentCommand{
		io:              io,
		credentialStore: credentialStore,
		logger:          logger,
	}
}

// Register registers the command and its sub-commands on the provided Registerer.
func (cmd *AgentCommand) Register(r command.Registerer) {
	clause := r.Command("agent", "Manage your personal account.").Hidden()

	clause.Flag("kill", "Kill a running agent").BoolVar(&cmd.kill)
	clause.Flag("restart", "Restart currently running agent").BoolVar(&cmd.restart)
	clause.Flag("daemon", "Start agent as a daemon").Short('d').BoolVar(&cmd.daemon)

	command.BindAction(clause, cmd.Run)
}

// Run handles the command with the options as specified in the command.
func (cmd *AgentCommand) Run() error {
	agent := cmd.agent()

	if cmd.kill || cmd.restart {
		err := agent.Kill()
		if err != nil {
			return fmt.Errorf("stop agent: %v", err)
		}
		if cmd.kill {
			return nil
		}
	}

	if cmd.daemon {
		executable, err := os.Executable()
		if err != nil {
			return fmt.Errorf("cannot find executable: %v", err)
		}
		execCmd := exec.Command(executable, "agent")
		execCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		err = execCmd.Start()
		if err != nil {
			return fmt.Errorf("cannot start daemon: %v", err)
		}
		return nil
	}

	return agent.Start()
}

func (cmd *AgentCommand) agent() *agent.Server {
	return agent.New(cmd.credentialStore.ConfigDir().Path(), Version, cmd.logger)
}
