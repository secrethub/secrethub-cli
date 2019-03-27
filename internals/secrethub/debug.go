package secrethub

// RegisterDebugFlag registers a debug flag.
func RegisterDebugFlag(r FlagRegisterer) {
	r.Flag("debug", "Enable debug mode.").Short('D')
}
