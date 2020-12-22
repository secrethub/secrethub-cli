package secrethub

import (
	"bytes"
	"os"

	"github.com/secrethub/secrethub-cli/internals/cli"
)

type CompletionCommand struct {
	shell  cli.StringValue
	clause *cli.CommandClause
}

// NewCompletionCommand is a command that, when executed, generates a completion script
// for a specific shell, based on the argument it is provided with. It is able to generate
// completions for Bash, ZSh, Fish and PowerShell.
func NewCompletionCommand() *CompletionCommand {
	return &CompletionCommand{}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *CompletionCommand) Register(r cli.Registerer) {
	cmd.clause = r.Command("autocomplete", "Generate completion script")
	cmd.clause.Cmd.DisableFlagsInUseLine = true
	cmd.clause.Cmd.ValidArgs = []string{"bash", "zsh", "fish", "powershell"}
	cmd.clause.BindAction(cmd.run)
	cmd.clause.BindArguments([]cli.Argument{{Value: &cmd.shell, Name: "shell", Required: true}})
}

func (cmd *CompletionCommand) run() error {
	switch cmd.shell.Param {
	case "bash":
		//options := []string{"yes", "y", ""}
		//ok := false
		//answer, err := ui.Ask(ui.NewUserIO(), "In order to install autocompletion, sudo privileges are needed. Do you agree to give permission? [yes/no]\n")
		//if err != nil {
		//	return err
		//}
		//for _, a := range options {
		//	if a == answer {
		//		ok = true
		//	}
		//}
		//if !ok {
		//	return nil
		//}
		buf := new(bytes.Buffer)
		buf.Write([]byte("#!/usr/bin/env bash\n"))
		err := cmd.clause.Cmd.Root().GenBashCompletion(buf)
		if err != nil {
			return err
		}
		f, err := os.OpenFile("contrib/completion/bash/secrethub", os.O_RDWR|os.O_CREATE, 0755)
		if err != nil {
			return err
		}
		_, err = f.Write(buf.Bytes())
		if err != nil {
			return err
		}
		f.Close()
		//err = exec.Command("sudo", "mv", "secrethub", "/etc/bash_completion.d/secrethub").Run()
		//if err != nil {
		//	return err
		//}
		//err = exec.Command("chmod", "+x", "/etc/bash_completion.d/secrethub").Run()
		//if err != nil {
		//	return err
		//}
		//fmt.Println("Generation complete. From the next terminal onward you will have autocompletion.")
		//fmt.Println("Please execute the following command To have autocompletion on the current command:\n  source /etc/bash_completion.d/secrethub")
	case "zsh":
		buf := new(bytes.Buffer)
		err := cmd.clause.Cmd.Root().GenZshCompletion(buf)
		if err != nil {
			return err
		}
		f, err := os.OpenFile("contrib/completion/zsh/_secrethub", os.O_RDWR|os.O_CREATE, 0755)
		if err != nil {
			return err
		}
		_, err = f.Write(buf.Bytes())
		if err != nil {
			return err
		}
		f.Close()
		//exec.Command("zsh", "-c", fmt.Sprint("echo", "\"autoload -U compinit; compinit\"", ">>", "~/.zshrc"))
		//path, err := exec.Command("zsh", "-c", fmt.Sprintf("\"echo %q\"", "${fpath[1]}")).Output()
		//if err != nil {
		//	return err
		//}
		//fmt.Println(path)
		//_, err = os.Create(string(path) + "/_secrethub")
		//if err != nil {
		//	return err
		//}
		//f.Write(buf.Bytes())
		//defer f.Close()
	case "fish":
		buf := new(bytes.Buffer)
		err := cmd.clause.Cmd.Root().GenFishCompletion(buf, true)
		if err != nil {
			return err
		}
		f, err := os.OpenFile("contrib//completion/fish/secrethub.fish", os.O_RDWR|os.O_CREATE, 0755)
		if err != nil {
			return err
		}
		_, err = f.Write(buf.Bytes())
		if err != nil {
			return err
		}
		f.Close()
	case "powershell":
		buf := new(bytes.Buffer)
		err := cmd.clause.Cmd.Root().GenPowerShellCompletion(buf)
		if err != nil {
			return err
		}
		f, err := os.OpenFile("contrib/completion/powershell/secrethub", os.O_RDWR|os.O_CREATE, 0755)
		if err != nil {
			return err
		}
		_, err = f.Write(buf.Bytes())
		if err != nil {
			return err
		}
		f.Close()
	}
	return nil
}
