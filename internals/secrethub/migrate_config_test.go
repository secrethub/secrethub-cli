package secrethub

import (
	"testing"

	"github.com/secrethub/secrethub-go/internals/assert"
)

func TestMigrateTemplates(t *testing.T) {
	for name, tc := range map[string]struct {
		in          string
		expected    string
		expectedErr bool
		mapping     map[string]string
		vars        map[string][]string
	}{
		"json": {
			in: `
			{
				"db_host": "db.internal",
				"db_user": "{{ org/repo/dir/user }}",
				"db_password": "{{ org/repo/dir/password }}",
				"db_port": 5432
			}
			`,
			expected: `
			{
				"db_host": "db.internal",
				"db_user": "{{ op://vault/item/user }}",
				"db_password": "{{ op://vault/item/password }}",
				"db_port": 5432
			}
			`,
			mapping: map[string]string{
				"secrethub://org/repo/dir/user":     "op://vault/item/user",
				"secrethub://org/repo/dir/password": "op://vault/item/password",
			},
		},
		"json no whitespaces": {
			in: `
			{
				"db_host": "db.internal",
				"db_user": "{{org/repo/dir/user}}",
				"db_password": "{{org/repo/dir/password}}",
				"db_port": 5432
			}
			`,
			expected: `
			{
				"db_host": "db.internal",
				"db_user": "{{ op://vault/item/user }}",
				"db_password": "{{ op://vault/item/password }}",
				"db_port": 5432
			}
			`,
			mapping: map[string]string{
				"secrethub://org/repo/dir/user":     "op://vault/item/user",
				"secrethub://org/repo/dir/password": "op://vault/item/password",
			},
		},
		"json one whitespaces": {
			in: `
			{
				"db_host": "db.internal",
				"db_user": "{{org/repo/dir/user }}",
				"db_password": "{{ org/repo/dir/password}}",
				"db_port": 5432
			}
			`,
			expected: `
			{
				"db_host": "db.internal",
				"db_user": "{{ op://vault/item/user }}",
				"db_password": "{{ op://vault/item/password }}",
				"db_port": 5432
			}
			`,
			mapping: map[string]string{
				"secrethub://org/repo/dir/user":     "op://vault/item/user",
				"secrethub://org/repo/dir/password": "op://vault/item/password",
			},
		},
		"yaml": {
			in: `
			db_host: db.internal
			db_user: "{{ org/repo/dir/user }}"
			db_password: {{ org/repo/dir/password }}
			db_port: 5432
			`,
			expected: `
			db_host: db.internal
			db_user: "{{ op://vault/item/user }}"
			db_password: {{ op://vault/item/password }}
			db_port: 5432
			`,
			mapping: map[string]string{
				"secrethub://org/repo/dir/user":     "op://vault/item/user",
				"secrethub://org/repo/dir/password": "op://vault/item/password",
			},
		},
		"missing secret": {
			in: `
			db_host: db.internal
			db_user: "{{ org/repo/dir/user }}"
			db_password: {{ org/repo/dir/password }}
			db_port: 5432
			`,
			expectedErr: true,
			mapping: map[string]string{
				"secrethub://org/repo/dir/user": "op://vault/item/user",
			},
		},
		"with vars": {
			in: `
			db_host: db.internal
			db_user: "{{ org/repo/$env/dir/user }}"
			db_password: {{ org/repo/$env/dir/password }}
			db_port: 5432
			`,
			expected: `
			db_host: db.internal
			db_user: "{{ op://vault-$ENV/item/user }}"
			db_password: {{ op://vault-$ENV/item/password }}
			db_port: 5432
			`,
			mapping: map[string]string{
				"secrethub://org/repo/prod/dir/user":     "op://vault-prod/item/user",
				"secrethub://org/repo/prod/dir/password": "op://vault-prod/item/password",
				"secrethub://org/repo/dev/dir/user":      "op://vault-dev/item/user",
				"secrethub://org/repo/dev/dir/password":  "op://vault-dev/item/password",
			},
			vars: map[string][]string{
				"env": {"dev", "prod"},
			},
		},
		"with vars no whitespaces": {
			in: `
			db_host: db.internal
			db_user: "{{org/repo/$env/dir/user}}"
			db_password: {{org/repo/$env/dir/password}}
			db_port: 5432
			`,
			expected: `
			db_host: db.internal
			db_user: "{{ op://vault-$ENV/item/user }}"
			db_password: {{ op://vault-$ENV/item/password }}
			db_port: 5432
			`,
			mapping: map[string]string{
				"secrethub://org/repo/prod/dir/user":     "op://vault-prod/item/user",
				"secrethub://org/repo/prod/dir/password": "op://vault-prod/item/password",
				"secrethub://org/repo/dev/dir/user":      "op://vault-dev/item/user",
				"secrethub://org/repo/dev/dir/password":  "op://vault-dev/item/password",
			},
			vars: map[string][]string{
				"env": {"dev", "prod"},
			},
		},
		"with vars one whitespaces": {
			in: `
			db_host: db.internal
			db_user: "{{ org/repo/$env/dir/user}}"
			db_password: {{org/repo/$env/dir/password }}
			db_port: 5432
			`,
			expected: `
			db_host: db.internal
			db_user: "{{ op://vault-$ENV/item/user }}"
			db_password: {{ op://vault-$ENV/item/password }}
			db_port: 5432
			`,
			mapping: map[string]string{
				"secrethub://org/repo/prod/dir/user":     "op://vault-prod/item/user",
				"secrethub://org/repo/prod/dir/password": "op://vault-prod/item/password",
				"secrethub://org/repo/dev/dir/user":      "op://vault-dev/item/user",
				"secrethub://org/repo/dev/dir/password":  "op://vault-dev/item/password",
			},
			vars: map[string][]string{
				"env": {"dev", "prod"},
			},
		},
		"no op": {
			in: `
			db_user: "db-user"
			db_port: 5432
			`,
			expected: `
			db_user: "db-user"
			db_port: 5432
			`,
			mapping: map[string]string{},
		},
	} {
		t.Run(name, func(t *testing.T) {
			var out string
			m := referenceMapping(tc.mapping)
			m.stripSecretHubURIScheme()
			err := m.addVarPossibilities(tc.vars)
			assert.OK(t, err)

			out, _, err = migrateTemplateTags(tc.in, m, "{{ %s }}")
			if tc.expectedErr {
				assert.Equal(t, err != nil, true)
				return
			}

			assert.OK(t, err)
			assert.Equal(t, out, tc.expected)
		})
	}
}

func TestMigrateEnvfile(t *testing.T) {
	for name, tc := range map[string]struct {
		in          string
		expected    string
		expectedErr bool
		mapping     map[string]string
		vars        map[string][]string
	}{
		"envfile": {
			in: `
			DB_HOST=db.internal
			DB_USER={{ org/repo/dir/user }}
			DB_PASSWORD={{ org/repo/dir/password }}
			DB_PORT=5432
			`,
			expected: `
			DB_HOST=db.internal
			DB_USER=op://vault/item/user
			DB_PASSWORD=op://vault/item/password
			DB_PORT=5432
			`,
			mapping: map[string]string{
				"secrethub://org/repo/dir/user":     "op://vault/item/user",
				"secrethub://org/repo/dir/password": "op://vault/item/password",
			},
		},
		"envfile no whitespaces": {
			in: `
			DB_HOST=db.internal
			DB_USER={{org/repo/dir/user}}
			DB_PASSWORD={{org/repo/dir/password}}
			DB_PORT=5432
			`,
			expected: `
			DB_HOST=db.internal
			DB_USER=op://vault/item/user
			DB_PASSWORD=op://vault/item/password
			DB_PORT=5432
			`,
			mapping: map[string]string{
				"secrethub://org/repo/dir/user":     "op://vault/item/user",
				"secrethub://org/repo/dir/password": "op://vault/item/password",
			},
		},
		"envfile one whitespace": {
			in: `
			DB_HOST=db.internal
			DB_USER={{org/repo/dir/user }}
			DB_PASSWORD={{ org/repo/dir/password}}
			DB_PORT=5432
			`,
			expected: `
			DB_HOST=db.internal
			DB_USER=op://vault/item/user
			DB_PASSWORD=op://vault/item/password
			DB_PORT=5432
			`,
			mapping: map[string]string{
				"secrethub://org/repo/dir/user":     "op://vault/item/user",
				"secrethub://org/repo/dir/password": "op://vault/item/password",
			},
		},
		"with comments": {
			in: `
			# Database config
			DB_HOST=db.internal
			DB_USER={{ org/repo/dir/user }}
			DB_PASSWORD={{ org/repo/dir/password }}
			DB_PORT=5432
			`,
			expected: `
			# Database config
			DB_HOST=db.internal
			DB_USER=op://vault/item/user
			DB_PASSWORD=op://vault/item/password
			DB_PORT=5432
			`,
			mapping: map[string]string{
				"secrethub://org/repo/dir/user":     "op://vault/item/user",
				"secrethub://org/repo/dir/password": "op://vault/item/password",
			},
		},
		"missing secret": {
			in: `
			# Database config
			DB_HOST=db.internal
			DB_USER={{ org/repo/dir/user }}
			DB_PASSWORD={{ org/repo/dir/password }}
			DB_PORT=5432
			`,
			expectedErr: true,
			mapping: map[string]string{
				"secrethub://org/repo/dir/user": "op://vault/item/user",
			},
		},
		"with vars": {
			in: `
			DB_HOST=db.internal
			DB_USER={{ org/repo/$env/dir/user }}
			DB_PASSWORD={{ org/repo/$env/dir/password }}
			DB_PORT=5432
			`,
			expected: `
			DB_HOST=db.internal
			DB_USER=op://vault-$ENV/item/user
			DB_PASSWORD=op://vault-$ENV/item/password
			DB_PORT=5432
			`,
			mapping: map[string]string{
				"secrethub://org/repo/prod/dir/user":     "op://vault-prod/item/user",
				"secrethub://org/repo/prod/dir/password": "op://vault-prod/item/password",
				"secrethub://org/repo/dev/dir/user":      "op://vault-dev/item/user",
				"secrethub://org/repo/dev/dir/password":  "op://vault-dev/item/password",
			},
			vars: map[string][]string{
				"env": {"dev", "prod"},
			},
		},
		"with vars no whitespaces": {
			in: `
			DB_HOST=db.internal
			DB_USER={{org/repo/$env/dir/user}}
			DB_PASSWORD={{org/repo/$env/dir/password}}
			DB_PORT=5432
			`,
			expected: `
			DB_HOST=db.internal
			DB_USER=op://vault-$ENV/item/user
			DB_PASSWORD=op://vault-$ENV/item/password
			DB_PORT=5432
			`,
			mapping: map[string]string{
				"secrethub://org/repo/prod/dir/user":     "op://vault-prod/item/user",
				"secrethub://org/repo/prod/dir/password": "op://vault-prod/item/password",
				"secrethub://org/repo/dev/dir/user":      "op://vault-dev/item/user",
				"secrethub://org/repo/dev/dir/password":  "op://vault-dev/item/password",
			},
			vars: map[string][]string{
				"env": {"dev", "prod"},
			},
		},
		"with vars one whitespace": {
			in: `
			DB_HOST=db.internal
			DB_USER={{org/repo/$env/dir/user }}
			DB_PASSWORD={{ org/repo/$env/dir/password}}
			DB_PORT=5432
			`,
			expected: `
			DB_HOST=db.internal
			DB_USER=op://vault-$ENV/item/user
			DB_PASSWORD=op://vault-$ENV/item/password
			DB_PORT=5432
			`,
			mapping: map[string]string{
				"secrethub://org/repo/prod/dir/user":     "op://vault-prod/item/user",
				"secrethub://org/repo/prod/dir/password": "op://vault-prod/item/password",
				"secrethub://org/repo/dev/dir/user":      "op://vault-dev/item/user",
				"secrethub://org/repo/dev/dir/password":  "op://vault-dev/item/password",
			},
			vars: map[string][]string{
				"env": {"dev", "prod"},
			},
		},
		"composite secrets": {
			in: `
			DB_ADDRESS={{ org/repo/dir/host }}:5432
			`,
			expectedErr: true,
			mapping: map[string]string{
				"secrethub://org/repo/dir/host": "op://vault/item/host",
			},
		},
		"no op": {
			in: `
			DB_HOST=db.internal
			DB_PORT=5432
			`,
			expected: `
			DB_HOST=db.internal
			DB_PORT=5432
			`,
			mapping: map[string]string{},
		},
	} {
		t.Run(name, func(t *testing.T) {
			var out string
			m := referenceMapping(tc.mapping)
			m.stripSecretHubURIScheme()
			err := m.addVarPossibilities(tc.vars)
			assert.OK(t, err)

			err = func() error {
				err := checkForCompositeSecrets([]byte(tc.in))
				if err != nil {
					return err
				}

				out, _, err = migrateTemplateTags(tc.in, m, "%s")
				return err
			}()

			if tc.expectedErr {
				assert.Equal(t, err != nil, true)
				return
			}

			assert.OK(t, err)
			assert.Equal(t, out, tc.expected)
		})
	}
}
