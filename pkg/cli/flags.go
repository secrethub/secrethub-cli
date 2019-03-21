package cli

// FlagRegisterer allows others to register flags on it.
type FlagRegisterer interface {
	Flag(name, help string) *Flag
}
