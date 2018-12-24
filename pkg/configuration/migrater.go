package configuration

import (
	"github.com/keylockerbv/secrethub-cli/pkg/ui"
	"github.com/secrethub/secrethub-go/internals/errio"
)

var (
	// ErrVersionNotReachable is given when the migrater cannot reach the desired version
	ErrVersionNotReachable = errConfig.Code("not_reachable").Error("desired config version not reachable")
)

// Migration defines a function that is used to convert a configuration in ConfigMap format from VersionFrom to VersionTo
type Migration struct {
	VersionFrom int
	VersionTo   int
	UpdateFunc  func(ui.IO, ConfigMap) (ConfigMap, error)
}

// MigrateConfigTo attempts to convert a config in ConfigMap format from versionFrom to versionTo
// A list of Migrations has to be passed that form at least a path from versionFrom to versionTo
// Function may fail if the migration path contains dead ends that do not lead to versionTo
// Setting checkOnly to true will not perform the actual migration and can be used to check whether there is a valid
// migration path from one version to another
func MigrateConfigTo(io ui.IO, config ConfigMap, versionFrom int, versionTo int, migrations []Migration, checkOnly bool) (ConfigMap, error) {
	var err error
	version := versionFrom
	for version != versionTo {
		migrated := false
		for _, m := range migrations {
			if m.VersionFrom == version {
				if !checkOnly {
					config, err = m.migrate(io, config)
				}
				version = m.VersionTo
				migrated = true
				break
			}
		}

		if err != nil {
			return config, errio.Error(err)
		}
		if !migrated {
			return config, ErrVersionNotReachable
		}

	}

	config["version"] = versionTo

	return config, nil
}

func (m *Migration) migrate(io ui.IO, src ConfigMap) (ConfigMap, error) {
	return m.UpdateFunc(io, src)
}
