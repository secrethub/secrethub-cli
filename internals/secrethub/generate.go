package secrethub

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/secrethub/secrethub-cli/internals/cli/clip"
	"github.com/secrethub/secrethub-cli/internals/cli/ui"
	"github.com/secrethub/secrethub-cli/internals/secrethub/command"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/errio"
	"github.com/secrethub/secrethub-go/pkg/randchar"

	"github.com/docker/go-units"
)

var (
	errGenerate = errio.Namespace("generate")

	// ErrInvalidRandLength is returned when an invalid length is given.
	ErrInvalidRandLength         = errGenerate.Code("invalid_rand_length").Error("The secret length must be larger than 0")
	ErrCannotUseLengthArgAndFlag = errGenerate.Code("length_arg_and_flag").Error("length cannot be provided as an argument and a flag at the same time")
	ErrCouldNotFindCharSet       = errGenerate.Code("charset_not_found").ErrorPref("could not find charset: %s")
	ErrMinFlagInvalidInteger     = errGenerate.Code("min_flag_invalid_int").ErrorPref("second part of --min flag is not an integer: %s")
	ErrInvalidMinFlag            = errGenerate.Code("min_flag_invalid").ErrorPref("min flag must be of the form <charset name>:<minimum count>, invalid min flag: %s")
)

const defaultLength = 22

// GenerateSecretCommand generates a new secret and writes to the output path.
type GenerateSecretCommand struct {
	symbolsFlag         boolValue
	generator           randchar.Generator
	io                  ui.IO
	lengthFlag          intValue
	firstArg            string
	secondArg           string
	lengthArg           intValue
	charsetFlag         charsetValue
	mins                minRuleValue
	copyToClipboard     bool
	clearClipboardAfter time.Duration
	clipper             clip.Clipper
	newClient           newClientFunc
}

// NewGenerateSecretCommand creates a new GenerateSecretCommand.
func NewGenerateSecretCommand(io ui.IO, newClient newClientFunc) *GenerateSecretCommand {
	return &GenerateSecretCommand{
		io:                  io,
		newClient:           newClient,
		clearClipboardAfter: defaultClearClipboardAfter,
		clipper:             clip.NewClipboard(),
	}
}

// Register registers the command, arguments and flags on the provided Registerer.
func (cmd *GenerateSecretCommand) Register(r command.Registerer) {
	clause := r.Command("generate", "Generate a random secret.")
	clause.Arg("secret-path", "The path to write the generated secret to").Required().PlaceHolder(secretPathPlaceHolder).StringVar(&cmd.firstArg)
	clause.Flag("length", "The length of the generated secret. Defaults to "+strconv.Itoa(defaultLength)).PlaceHolder(strconv.Itoa(defaultLength)).Short('l').SetValue(&cmd.lengthFlag)
	clause.Flag("min", "<charset>:<n> Ensure that the resulting password contains at least n characters from the given character set. Note that adding constraints reduces the strength of the secret. When possible, avoid any constraints.").SetValue(&cmd.mins)
	clause.Flag("clip", "Copy the generated value to the clipboard. The clipboard is automatically cleared after "+units.HumanDuration(cmd.clearClipboardAfter)+".").Short('c').BoolVar(&cmd.copyToClipboard)
	clause.Flag("charset", "Define the set of characters to randomly generate a password from. Options are all, alphanumeric, numeric, lowercase, uppercase, letters, symbols and human-readable. Multiple character sets can be combined by supplying them in a comma separated list. Defaults to alphanumeric.").Default("alphanumeric").HintOptions("all", "alphanumeric", "numeric", "lowercase", "uppercase", "letters", "symbols", "human-readable").SetValue(&cmd.charsetFlag)
	clause.Flag("symbols", "Include symbols in secret.").Short('s').Hidden().SetValue(&cmd.symbolsFlag)
	clause.Arg("rand-command", "").Hidden().StringVar(&cmd.secondArg)
	clause.Arg("length", "").Hidden().SetValue(&cmd.lengthArg)

	command.BindAction(clause, cmd.Run)
}

// before configures the command using the flag values.
func (cmd *GenerateSecretCommand) before() error {
	useSymbols, err := cmd.useSymbols()
	if err != nil {
		return err
	}

	charset := cmd.charsetFlag.v
	if useSymbols {
		charset = charset.Add(randchar.Symbols)
	}

	cmd.generator, err = randchar.NewRand(charset, cmd.mins.v...)
	if err != nil {
		return err
	}

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

	data, err := cmd.generator.Generate(length)
	if err != nil {
		return err
	}

	client, err := cmd.newClient()
	if err != nil {
		return err
	}

	version, err := client.Secrets().Write(path, data)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.io.Stdout(), "A randomly generated secret has been written to %s:%d.\n", path, version.Version)

	if cmd.copyToClipboard {
		err = WriteClipboardAutoClear(data, cmd.clearClipboardAfter, cmd.clipper)
		if err != nil {
			return err
		}

		fmt.Fprintf(
			cmd.io.Stdout(),
			"The generated value has been copied to the clipboard. It will be cleared after %s.\n",
			units.HumanDuration(cmd.clearClipboardAfter),
		)
	}

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

func (cmd *GenerateSecretCommand) useSymbols() (bool, error) {
	if cmd.symbolsFlag.IsSet() {
		return cmd.symbolsFlag.Get(), nil
	}

	useSymbolsEnv := os.Getenv("SECRETHUB_GENERATE_RAND_SYMBOLS")
	if useSymbolsEnv != "" {
		b, err := strconv.ParseBool(useSymbolsEnv)
		if err != nil {
			return false, err
		}
		return b, nil
	}

	return false, nil
}

type minRuleValue struct {
	v []randchar.Option
}

func (ov *minRuleValue) String() string {
	return ""
}

func (ov *minRuleValue) Set(flagValue string) error {
	elements := strings.Split(flagValue, ":")
	if len(elements) != 2 {
		return ErrInvalidMinFlag(flagValue)
	}

	count, err := strconv.Atoi(elements[1])
	if err != nil {
		return ErrMinFlagInvalidInteger(elements[1])
	}

	charset, found := randchar.CharsetByName(elements[0])
	if !found {
		return ErrCouldNotFindCharSet(elements[0])
	}

	ov.v = append(ov.v, randchar.Min(count, charset))
	return nil
}

func (ov *minRuleValue) IsCumulative() bool {
	return true
}

type charsetValue struct {
	v randchar.Charset
}

func (cv *charsetValue) String() string {
	return ""
}

func (cv *charsetValue) Set(flagValue string) error {
	charsetNames := strings.Split(flagValue, ",")
	for _, charsetName := range charsetNames {
		charset, ok := randchar.CharsetByName(charsetName)
		if !ok {
			return ErrCouldNotFindCharSet(charsetName)
		}
		cv.v = cv.v.Add(charset)
	}
	return nil
}

func (cv *charsetValue) IsCumulative() bool {
	return true
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

type boolValue struct {
	v *bool
}

func (iv *boolValue) Get() bool {
	if iv.v == nil {
		return false
	}
	return *iv.v
}

func (iv *boolValue) IsSet() bool {
	return iv.v != nil
}

func (iv *boolValue) Set(s string) error {
	b, err := strconv.ParseBool(s)
	if err == nil {
		iv.v = &b
	}
	return err
}

func (iv *boolValue) String() string {
	return fmt.Sprintf("%v", iv.v)
}
