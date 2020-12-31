package secrethub

const (
	repoPathPlaceHolder                  = "<namespace>/<repo>"
	dirPathPlaceHolder                   = repoPathPlaceHolder + "/<dir>[/<dir> ...]"
	dirPathsPlaceHolder                  = dirPathPlaceHolder + "..."
	optionalDirPathPlaceHolder           = repoPathPlaceHolder + "[/<dir> ...]"
	secretPathPlaceHolder                = optionalDirPathPlaceHolder + "/<secret>"
	secretPathOptionalVersionPlaceHolder = secretPathPlaceHolder + "[:<version>]"
	generalPathPlaceHolder               = repoPathPlaceHolder + "/<path>"
	optionalSecretPathPlaceHolder        = repoPathPlaceHolder + "[[/<dir> ...]/<secret>]"
)
