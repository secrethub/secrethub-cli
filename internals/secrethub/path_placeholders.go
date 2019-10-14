package secrethub

const (
	repoPathPlaceHolder                  = "<namespace>/<repo>"
	dirPathPlaceHolder                   = repoPathPlaceHolder + "/<dir>[/<dir> ...]"
	optionalDirPathPlaceHolder           = repoPathPlaceHolder + "[/<dir> ...]"
	secretPathPlaceHolder                = optionalDirPathPlaceHolder + "/<secret>"
	secretPathOptionalVersionPlaceHolder = secretPathPlaceHolder + "[:<version>]"
	anyPathPlaceHolder                   = "<namespace>[/<repo>[/<dir> ...][/<secret>[:<version>]]]"
)
