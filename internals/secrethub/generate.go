package secrethub

import (
	"fmt"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/randchar"
)

var (
	errGenerate = errio.Namespace("generate")

	// ErrInvalidRandLength is returned when an invalid length is given.
	ErrInvalidRandLength = errGenerate.Code("invalid_rand_length").Error("The secret length must be larger than 0")
)

// GenerateSecretCommand generates a new secret and writes to the output path.
type GenerateSecretCommand struct {
	useSymbols bool
	generator  randchar.Generator
	io         ui.IO
	length     int
	path       api.SecretPath
	newClient  newClientFunc
}

// NewGenerateSecretCommand creates a new GenerateSecretCommand.
func NewGenerateSecretCommand(io ui.IO, newClient newClientFunc) *GenerateSecretCommand {
	return &GenerateSecretCommand{
		io:        io,
		newClient: newClient,
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *GenerateSecretCommand) Register(r Registerer) {
	generateCommand := r.Command("generate", "").Hidden()

	clause := generateCommand.Command("rand", "Generate a random secret. By default, it uses numbers (0-9), lowercase letters (a-z) and uppercase letters (A-Z) and a length of 22.")
	clause.Arg("secret-path", "The path to write the generated secret to (<namespace>/<repo>[/<dir>]/<secret>)").Required().SetValue(&cmd.path)
	clause.Arg("length", "The length of the generated secret. Defaults to 22.").Default("22").IntVar(&cmd.length)
	clause.Flag("symbols", "Include symbols in secret.").Short('s').BoolVar(&cmd.useSymbols)

	// TODO SHDEV-528: implement --clip
	// clause.Flag("clip", "Copy the secret value to the clipboard. The clipboard is automatically cleared after 45 seconds.").Short('c').BoolVar(cmd.clip)

	BindAction(clause, cmd.Run)
}

// before configures the command using the flag values.
func (cmd *GenerateSecretCommand) before() {
	cmd.generator = randchar.NewGenerator(cmd.useSymbols)
}

// Run generates a new secret and writes to the output path.
func (cmd *GenerateSecretCommand) Run() error {
	cmd.before()
	return cmd.run()
}

// run generates a new secret and writes to the output path.
func (cmd *GenerateSecretCommand) run() error {
	if cmd.length <= 0 {
		return ErrInvalidRandLength
	}

	fmt.Fprint(cmd.io.Stdout(), "Generating secret value...\n")

	data, err := cmd.generator.Generate(cmd.length)
	if err != nil {
		return err
	}

	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	fmt.Fprint(cmd.io.Stdout(), "Writing secret value...\n")

	version, err := client.Secrets().Write(cmd.path.Value(), data)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.io.Stdout(), "Write complete! A randomly generated secret has been written to %s:%d.\n", cmd.path, version.Version)

	return nil
}
