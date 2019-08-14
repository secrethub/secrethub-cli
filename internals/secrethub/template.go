package secrethub

import (
	"github.com/secrethub/secrethub-cli/internals/secrethub/tpl"
)

func getTemplateParser(raw []byte, version string) (tpl.Parser, error) {
	switch version {
	case "auto":
		if tpl.IsV1Template(raw) {
			return tpl.NewV1Parser(), nil
		}
		return tpl.NewParser(), nil
	case "1", "v1":
		return tpl.NewV1Parser(), nil
	case "2", "v2":
		return tpl.NewV2Parser(), nil
	case "latest":
		return tpl.NewParser(), nil
	default:
		return nil, ErrUnknownTemplateVersion(version)
	}
}
