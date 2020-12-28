package secrethub

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/secrethub/secrethub-cli/internals/cli"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
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
	cmd.clause = r.Command("completion", "Installs completion script").Hidden()
	cmd.clause.Cmd.DisableFlagsInUseLine = true
	cmd.clause.Flags().BoolP("n", "n", false, "generate script now")
	cmd.clause.Flags().BoolP("yes", "y", false, "confirm that you allow sudo privileges")
	cmd.clause.Cmd.ValidArgs = []string{"bash", "zsh", "fish", "powershell"}
	cmd.clause.BindAction(cmd.run)
	cmd.clause.BindArguments([]cli.Argument{{Value: &cmd.shell, Name: "shell", Required: true, Description: "The type of shell for which you want to generate completion script"}})
}

func (cmd *CompletionCommand) run() error {
	switch cmd.shell.Param {
	case "bash":
		if cmd.clause.Flag("n").Changed {
			_ = cmd.clause.Cmd.Root().GenBashCompletion(os.Stdout)
			return nil
		}

		if runtime.GOOS == "darwin" {
			if _, err := os.Stat("/usr/local/etc/bash_completion.d/secrethub"); !os.IsNotExist(err) {
				fmt.Println("You already have installed autocompletion for bash.")
				return nil
			}
		} else if runtime.GOOS == "linux" {
			if _, err := os.Stat("/usr/share/bash-completion/completions/secrethub"); !os.IsNotExist(err) {
				fmt.Println("You already have installed autocompletion for bash.")
				return nil
			} else if _, err := os.Stat("/etc/bash_completion.d/secrethub"); !os.IsNotExist(err) {
				fmt.Println("You already have installed autocompletion for bash.")
				return nil
			}
		}
		if !cmd.clause.Flag("yes").Changed && !allowSudo() {
			return nil
		}

		if runtime.GOOS == "darwin" {
			err := exec.Command("/bin/sh", "-c", "secrethub completion -n bash | sudo tee /usr/local/etc/bash_completion.d/secrethub").Run()
			if err != nil {
				return err
			}
		} else if runtime.GOOS == "linux" {
			err := exec.Command("/bin/sh", "-c", "secrethub completion -n bash | sudo tee /etc/bash_completion.d/secrethub").Run()
			if err != nil {
				return err
			}
		}
		fmt.Println("Generation complete. From the next terminal onward you will have autocompletion.")
		fmt.Println("To enable it on the current one, run the following command:\n source <(secrethub completion -n bash)")
	case "zsh":
		if cmd.clause.Flag("n").Changed {
			_ = cmd.clause.Cmd.Root().GenZshCompletion(os.Stdout)
			return nil
		}

		dirPath, err := exec.Command("zsh", "-c", "echo ${fpath[1]}").Output()
		if err != nil {
			return err
		}
		f := string(dirPath[:len(dirPath)-1]) + "/_secrethub"
		if _, err = os.Stat(f); !os.IsNotExist(err) {
			fmt.Println("You already have installed autocompletion for zsh.")
			return nil
		}
		if !cmd.clause.Flag("yes").Changed && !allowSudo() {
			return nil
		}
		err = exec.Command("zsh", "-c", "secrethub completion -n zsh | sudo tee "+f).Run()
		if err != nil {
			return err
		}
		fmt.Println("Installation complete. Please restart your terminal to take effect.")
	case "fish":
		if cmd.clause.Flag("n").Changed {
			_ = cmd.clause.Cmd.Root().GenFishCompletion(os.Stdout, true)
			return nil
		}
		dirPath, err := exec.Command("fish", "-c", "if test -d ~/.config/fish/completions echo ~/.config/fish/completions end").Output()
		if err != nil {
			return err
		}
		if string(dirPath) == "" {
			err = exec.Command("fish", "-c", "mkdir -p ~/.config/fish/completions").Run()
			if err != nil {
				return err
			}
			dirPath, err = exec.Command("fish", "-c", "echo ~/.config/fish/completions").Output()
			if err != nil {
				return err
			}
		}
		f := string(dirPath[:len(dirPath)-1]) + "/secrethub.fish"
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			fmt.Println("You already have installed autocompletion for bash.")
			return nil
		}
		err = exec.Command("fish", "-c", "secrethub completion -n fish >> "+f).Run()
		if err != nil {
			return err
		}
		fmt.Println("Generation complete. From the next terminal onward you will have autocompletion.")
		fmt.Println("To enable it on the current one, run the following command:\n secrethub completion -n fish | source")
	case "powershell":
		if cmd.clause.Flag("n").Changed {
			_ = cmd.clause.Cmd.Root().GenPowerShellCompletion(os.Stdout)
			return nil
		}

		ps, _ := exec.LookPath("powershell.exe")
		o, err := exec.Command(ps, "Test-Path", "$PROFILE").Output()
		if err != nil {
			return err
		}

		if a, _ := strconv.ParseBool(string(o)); a {
			//create profile
			_ = exec.Command(ps, "New-Item", "-Type", "File", "-Force", "$PROFILE").Run()
		}

		profilePathByte, _ := exec.Command(ps, "$PROFILE").Output()
		profileFile := string(profilePathByte)[:len(string(profilePathByte))-2]
		profilePath := strings.TrimSuffix(profileFile, "Microsoft.PowerShell_profile.ps1")

		if _, err = os.Stat(profilePath + "secrethub.ps1"); !os.IsNotExist(err) {
			fmt.Println("You already have installed autocompletion for powershell.")
			return nil
		}

		err = exec.Command(ps, "secrethub", "completion", "-n", "powershell", "|", "Out-File", "-FilePath", profilePath+"secrethub.ps1").Run()
		if err != nil {
			return err
		}

		f, err := os.OpenFile(profileFile, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			return err
		}
		scanner := bufio.NewScanner(f)
		hasMenuTab := false
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "Set-PSReadlineKeyHandler -Chord Tab -Function MenuComplete") {
				hasMenuTab = true
				break
			}
		}
		_, err = f.WriteString("\n. " + profilePath + "secrethub.ps1\n")
		if err != nil {
			return err
		}
		if !hasMenuTab {
			_, err = f.WriteString("Set-PSReadlineKeyHandler -Chord Tab -Function MenuComplete\n")
			if err != nil {
				return err
			}
		}
		f.Close()
		fmt.Println("Installation complete. Please restart your PowerShell window to take effect.")
	}
	return nil
}

func allowSudo() bool {
	options := []string{"yes", "y", ""}
	ok := false
	answer, _ := ui.Ask(ui.NewUserIO(), "In order to install autocompletion, sudo privileges are needed. Do you agree to give permission? [yes/no]\n")
	for _, a := range options {
		if a == answer {
			ok = true
		}
	}
	return ok
}
