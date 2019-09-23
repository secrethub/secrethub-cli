package secrethub

import (
	"fmt"
	"os"
	"strconv"

	"github.com/secrethub/secrethub-cli/internals/cli/ui"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/randchar"
)

var (
	errGenerate = errio.Namespace("generate")

	// ErrInvalidRandLength is returned when an invalid length is given.
	ErrInvalidRandLength         = errGenerate.Code("invalid_rand_length").Error("The secret length must be larger than 0")
	ErrCannotUseLengthArgAndFlag = errGenerate.Code("length_arg_and_flag").Error("length cannot be provided as an argument and a flag at the same time")
)

const defaultLength = 22

// GenerateSecretCommand generates a new secret and writes to the output path.
type GenerateSecretCommand struct {
	useSymbols bool
	generator  randchar.Generator
	io         ui.IO
	lengthFlag intValue
	firstArg   string
	secondArg  string
	lengthArg  intValue
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
	clause := r.Command("generate", "Generate a random secret.")
	clause.HelpLong("By default, it uses numbers (0-9), lowercase letters (a-z) and uppercase letters (A-Z) and a length of 22.")
	clause.Arg("secret-path", "The path to write the generated secret to (<namespace>/<repo>[/<dir>]/<secret>)").Required().StringVar(&cmd.firstArg)
	clause.Flag("length", "The length of the generated secret. Defaults to "+strconv.Itoa(defaultLength)).PlaceHolder(strconv.Itoa(defaultLength)).SetValue(&cmd.lengthFlag)
	clause.Flag("symbols", "Include symbols in secret.").Short('s').BoolVar(&cmd.useSymbols)

	clause.Arg("rand-command", "").Hidden().StringVar(&cmd.secondArg)
	clause.Arg("length", "").Hidden().SetValue(&cmd.lengthArg)

	// TODO SHDEV-528: implement --clip
	// clause.Flag("clip", "Copy the secret value to the clipboard. The clipboard is automatically cleared after 45 seconds.").Short('c').BoolVar(cmd.clip)

	BindAction(clause, cmd.Run)
}

// before configures the command using the flag values.
func (cmd *GenerateSecretCommand) before() error {
	useSymbols := false

	useSymbolsEnv := os.Getenv("SECRETHUB_GENERATE_RAND_SYMBOLS")
	if useSymbolsEnv != "" {
		b, err := strconv.ParseBool(useSymbolsEnv)
		if err != nil {
			return err
		}
		useSymbols = useSymbols || b
	}

	useSymbols = useSymbols || cmd.useSymbols

	cmd.generator = randchar.NewGenerator(useSymbols)

	return nil
}

// Run generates a new secret and writes to the output path.
func (cmd *GenerateSecretCommand) Run() error {
	err := cmd.before()
	if err != nil {
		return err
	}
	return cmd.run()
}

// run generates a new secret and writes to the output path.
func (cmd *GenerateSecretCommand) run() error {
	length, err := cmd.length()
	if err != nil {
		return err
	}

	path, err := cmd.path()
	if err != nil {
		return err
	}

	if length <= 0 {
		return ErrInvalidRandLength
	}

	fmt.Fprint(cmd.io.Stdout(), "Generating secret value...\n")

	data, err := cmd.generator.Generate(length)
	if err != nil {
		return err
	}

	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	fmt.Fprint(cmd.io.Stdout(), "Writing secret value...\n")

	version, err := client.Secrets().Write(path, data)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.io.Stdout(), "Write complete! A randomly generated secret has been written to %s:%d.\n", path, version.Version)

	return nil
}

func (cmd *GenerateSecretCommand) length() (int, error) {
	if cmd.lengthArg.IsSet() && cmd.lengthFlag.IsSet() {
		return 0, ErrCannotUseLengthArgAndFlag
	}
	if cmd.lengthFlag.IsSet() {
		return cmd.lengthFlag.Get(), nil
	}
	if cmd.lengthArg.IsSet() {
		return cmd.lengthArg.Get(), nil
	}
	return defaultLength, nil
}

func (cmd *GenerateSecretCommand) path() (string, error) {
	if cmd.firstArg == "rand" {
		return cmd.secondArg, api.ValidateSecretPath(cmd.secondArg)
	}
	if cmd.secondArg != "" {
		return "", fmt.Errorf("unexpected %s", cmd.secondArg)
	}
	if cmd.lengthArg.IsSet() {
		return "", fmt.Errorf("unexpected %d", cmd.lengthArg.Get())
	}
	return cmd.firstArg, api.ValidateSecretPath(cmd.firstArg)
}

type intValue struct {
	v *int
}

func (iv *intValue) Get() int {
	if iv.v == nil {
		return 0
	}
	return *iv.v
}

func (iv *intValue) IsSet() bool {
	return iv.v != nil
}

func (iv *intValue) Set(s string) error {
	f, err := strconv.ParseFloat(s, 64)
	if err == nil {
		v := (int)(f)
		iv.v = &v
	}
	return err
}

func (iv *intValue) String() string {
	return fmt.Sprintf("%v", iv.v)
}
