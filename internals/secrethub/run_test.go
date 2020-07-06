package secrethub

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/secrethub/secrethub-go/internals/api/uuid"

	"github.com/secrethub/secrethub-cli/internals/cli/ui/fakeui"

	"github.com/secrethub/secrethub-cli/internals/secrethub/tpl"
	"github.com/secrethub/secrethub-cli/internals/secrethub/tpl/fakes"
	generictpl "github.com/secrethub/secrethub-cli/internals/tpl"

	"github.com/secrethub/secrethub-go/internals/api"
	"github.com/secrethub/secrethub-go/internals/assert"
	"github.com/secrethub/secrethub-go/pkg/secrethub"
	"github.com/secrethub/secrethub-go/pkg/secrethub/fakeclient"
)

func elemEqual(t *testing.T, actual []envvar, expected []envvar) {
isExpected:
	for _, a := range actual {
		for _, e := range expected {
			if a == e {
				continue isExpected
			}
		}
		t.Errorf("%+v encountered but not expected", a)
	}

isEncountered:
	for _, e := range expected {
		for _, a := range actual {
			if a == e {
				continue isEncountered
			}
		}
		t.Errorf("%+v expected but not encountered", e)
	}
}

func TestParseDotEnv(t *testing.T) {
	cases := map[string]struct {
		raw      string
		expected []envvar
		err      error
	}{
		"success": {
			raw: "foo=bar\nbaz={{path/to/secret}}",
			expected: []envvar{
				{
					key:               "foo",
					value:             "bar",
					lineNumber:        1,
					columnNumberKey:   1,
					columnNumberValue: 5,
				},
				{
					key:               "baz",
					value:             "{{path/to/secret}}",
					lineNumber:        2,
					columnNumberKey:   1,
					columnNumberValue: 5,
				},
			},
		},
		"success with spaces": {
			raw: "key = value",
			expected: []envvar{
				{
					key:               "key",
					value:             "value",
					lineNumber:        1,
					columnNumberKey:   1,
					columnNumberValue: 7,
				},
			},
		},
		"success with multiple spaces after key": {
			raw: "key    = value",
			expected: []envvar{
				{
					key:               "key",
					value:             "value",
					lineNumber:        1,
					columnNumberKey:   1,
					columnNumberValue: 10,
				},
			},
		},
		"success with multiple spaces before value": {
			raw: "key =  value",
			expected: []envvar{
				{
					key:               "key",
					value:             "value",
					lineNumber:        1,
					columnNumberKey:   1,
					columnNumberValue: 8,
				},
			},
		},
		"success with leading space": {
			raw: " key = value",
			expected: []envvar{
				{
					key:               "key",
					value:             "value",
					lineNumber:        1,
					columnNumberKey:   2,
					columnNumberValue: 8,
				},
			},
		},
		"success with leading tab": {
			raw: "\tkey = value",
			expected: []envvar{
				{
					key:               "key",
					value:             "value",
					lineNumber:        1,
					columnNumberKey:   2,
					columnNumberValue: 8,
				},
			},
		},
		"success with trailing space": {
			raw: "key = value ",
			expected: []envvar{
				{
					key:               "key",
					value:             "value",
					lineNumber:        1,
					columnNumberKey:   1,
					columnNumberValue: 7,
				},
			},
		},
		"success with tabs": {
			raw: "key\t=\tvalue",
			expected: []envvar{
				{
					key:               "key",
					value:             "value",
					lineNumber:        1,
					columnNumberKey:   1,
					columnNumberValue: 7,
				},
			},
		},
		"success with single quotes": {
			raw: "key='value'",
			expected: []envvar{
				{
					key:               "key",
					value:             "value",
					lineNumber:        1,
					columnNumberKey:   1,
					columnNumberValue: 6,
				},
			},
		},
		"success with double quotes": {
			raw: `key="value"`,
			expected: []envvar{
				{
					key:               "key",
					value:             "value",
					lineNumber:        1,
					columnNumberKey:   1,
					columnNumberValue: 6,
				},
			},
		},
		"success with quotes and whitespace": {
			raw: "key = 'value'",
			expected: []envvar{
				{
					key:               "key",
					value:             "value",
					lineNumber:        1,
					columnNumberKey:   1,
					columnNumberValue: 8,
				},
			},
		},
		"success comment": {
			raw: "# database\nDB_USER = user\nDB_PASS = pass",
			expected: []envvar{
				{
					key:               "DB_USER",
					value:             "user",
					lineNumber:        2,
					columnNumberKey:   1,
					columnNumberValue: 11,
				},
				{
					key:               "DB_PASS",
					value:             "pass",
					lineNumber:        3,
					columnNumberKey:   1,
					columnNumberValue: 11,
				},
			},
		},
		"success comment prefixed with spaces": {
			raw: "    # database\nDB_USER = user\nDB_PASS = pass",
			expected: []envvar{
				{
					key:               "DB_USER",
					value:             "user",
					lineNumber:        2,
					columnNumberKey:   1,
					columnNumberValue: 11,
				},
				{
					key:               "DB_PASS",
					value:             "pass",
					lineNumber:        3,
					columnNumberKey:   1,
					columnNumberValue: 11,
				},
			},
		},
		"success comment prefixed with a tab": {
			raw: "\t# database\nDB_USER = user\nDB_PASS = pass",
			expected: []envvar{
				{
					key:               "DB_USER",
					value:             "user",
					lineNumber:        2,
					columnNumberKey:   1,
					columnNumberValue: 11,
				},
				{
					key:               "DB_PASS",
					value:             "pass",
					lineNumber:        3,
					columnNumberKey:   1,
					columnNumberValue: 11,
				},
			},
		},
		"success empty lines": {
			raw: "foo=bar\n\nbar=baz",
			expected: []envvar{
				{
					key:               "foo",
					value:             "bar",
					lineNumber:        1,
					columnNumberKey:   1,
					columnNumberValue: 5,
				},
				{
					key:               "bar",
					value:             "baz",
					lineNumber:        3,
					columnNumberKey:   1,
					columnNumberValue: 5,
				},
			},
		},
		"success line with only spaces": {
			raw: "foo=bar\n    \nbar = baz",
			expected: []envvar{
				{
					key:               "foo",
					value:             "bar",
					lineNumber:        1,
					columnNumberKey:   1,
					columnNumberValue: 5,
				},
				{
					key:               "bar",
					value:             "baz",
					lineNumber:        3,
					columnNumberKey:   1,
					columnNumberValue: 7,
				},
			},
		},
		"= sign in value": {
			raw: "foo=foo=bar",
			expected: []envvar{
				{
					key:               "foo",
					value:             "foo=bar",
					lineNumber:        1,
					columnNumberKey:   1,
					columnNumberValue: 5,
				},
			},
		},
		"invalid": {
			raw: "foobar",
			err: ErrTemplate(1, errors.New("template is not formatted as key=value pairs")),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			actual, err := parseDotEnv(strings.NewReader(tc.raw))

			elemEqual(t, actual, tc.expected)
			assert.Equal(t, err, tc.err)
		})
	}
}

func TestParseYML(t *testing.T) {
	cases := map[string]struct {
		raw      string
		expected []envvar
		err      error
	}{
		"success": {
			raw: "foo: bar\nbaz: ${path/to/secret}",
			expected: []envvar{
				{
					key:        "foo",
					value:      "bar",
					lineNumber: -1,
				},
				{
					key:        "baz",
					value:      "${path/to/secret}",
					lineNumber: -1,
				},
			},
		},
		"= in value": {
			raw: "foo: foo=bar\nbar: baz",
			expected: []envvar{
				{
					key:        "foo",
					value:      "foo=bar",
					lineNumber: -1,
				},
				{
					key:        "bar",
					value:      "baz",
					lineNumber: -1,
				},
			},
		},
		"nested yml": {
			raw: "ROOT:\n\tSUB\n\t\tNAME: val1",
			err: errors.New("yaml: line 2: found character that cannot start any token"),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			actual, err := parseYML(strings.NewReader(tc.raw))

			elemEqual(t, actual, tc.expected)
			assert.Equal(t, err, tc.err)
		})
	}
}

func TestNewEnv(t *testing.T) {
	cases := map[string]struct {
		raw               string
		replacements      map[string]string
		templateVarReader tpl.VariableReader
		expected          map[string]string
		err               error
	}{
		"success": {
			raw: "foo=bar\nbaz={{path/to/secret}}",
			replacements: map[string]string{
				"path/to/secret": "val",
			},
			expected: map[string]string{
				"foo": "bar",
				"baz": "val",
			},
		},
		"success with vars": {
			raw: "foo=bar\nbaz={{${app}/db/pass}}",
			replacements: map[string]string{
				"company/application/db/pass": "secret",
			},
			templateVarReader: fakes.FakeVariableReader{
				Variables: map[string]string{
					"app": "company/application",
				},
			},
			expected: map[string]string{
				"foo": "bar",
				"baz": "secret",
			},
		},
		"success with var in key": {
			raw: "${var}=value",
			templateVarReader: fakes.FakeVariableReader{
				Variables: map[string]string{
					"var": "key",
				},
			},
			expected: map[string]string{
				"key": "value",
			},
		},
		"success yml": {
			raw: "foo: bar\nbaz: ${path/to/secret}",
			replacements: map[string]string{
				"path/to/secret": "val",
			},
			expected: map[string]string{
				"foo": "bar",
				"baz": "val",
			},
		},
		"secret not allowed in key": {
			raw: "{{ path/to/secret }}key=value",
			err: ErrSecretsNotAllowedInKey,
		},
		"yml template error": {
			raw: "foo: bar: baz",
			err: ErrTemplate(1, errors.New("template is not formatted as key=value pairs")),
		},
		"yml secret template error": {
			raw: "foo: ${path/to/secret\nbar: ${ path/to/secret }",
			err: generictpl.ErrTagNotClosed("}"),
		},
		"secret template error": {
			raw: "foo={{path/to/secret",
			err: tpl.ErrSecretTagNotClosed(1, 21),
		},
		"secret template error second line": {
			raw: "foo=bar\nbar={{ error@secretpath }}",
			err: tpl.ErrIllegalSecretCharacter(2, 13, '@'),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			parser, err := getTemplateParser([]byte(tc.raw), "auto")
			assert.OK(t, err)

			env, err := NewEnv("secrethub.env", strings.NewReader(tc.raw), tc.templateVarReader, parser)
			if err != nil {
				assert.Equal(t, err, tc.err)
			} else {
				actualValues, err := env.env()
				assert.Equal(t, err, tc.err)

				// resolve values
				actual := make(map[string]string, len(actualValues))
				for name, value := range actualValues {
					actual[name], err = value.resolve(fakes.FakeSecretReader{Secrets: tc.replacements})
					if err != nil {
						t.Fail()
					}
				}
				assert.Equal(t, actual, tc.expected)
			}
		})
	}
}

func TestRunCommand_Run(t *testing.T) {
	osStatNotExist := func(_ string) (info os.FileInfo, err error) {
		return nil, os.ErrNotExist
	}

	cases := map[string]struct {
		command RunCommand
		err     error
	}{
		"success, no secrets": {
			command: RunCommand{
				io: fakeui.NewIO(t),
				environment: &environment{
					osStat: osStatNotExist,
				},
				command: []string{"echo", "test"},
			},
		},
		"missing secret": {
			command: RunCommand{
				command: []string{"echo", "test"},
				environment: &environment{
					envar: map[string]string{
						"missing": "path/to/unexisting/secret",
					},
					osStat: osStatNotExist,
				},
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						SecretService: &fakeclient.SecretService{
							VersionService: &fakeclient.SecretVersionService{
								GetWithDataFunc: func(path string) (*api.SecretVersion, error) {
									return nil, api.ErrSecretNotFound
								},
							},
						},
					}, nil
				},
				ignoreMissingSecrets: false,
			},
			err: api.ErrSecretNotFound,
		},
		"missing secret ignored": {
			command: RunCommand{
				command: []string{"echo", "test"},
				environment: &environment{
					osStat: osStatNotExist,
					envar: map[string]string{
						"missing": "path/to/unexisting/secret",
					},
				},
				io: fakeui.NewIO(t),
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						SecretService: &fakeclient.SecretService{
							VersionService: &fakeclient.SecretVersionService{
								GetWithDataFunc: func(path string) (*api.SecretVersion, error) {
									return nil, api.ErrSecretNotFound
								},
							},
						},
					}, nil
				},
				ignoreMissingSecrets: true,
			},
			err: nil,
		},
		"repo does not exist ignored": {
			command: RunCommand{
				command: []string{"echo", "test"},
				environment: &environment{
					envar: map[string]string{
						"missing": "unexisting/repo/secret",
					},
					osStat: osStatNotExist,
				},
				io: fakeui.NewIO(t),
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						SecretService: &fakeclient.SecretService{
							VersionService: &fakeclient.SecretVersionService{
								GetWithDataFunc: func(path string) (*api.SecretVersion, error) {
									return nil, api.ErrRepoNotFound("unexisting/repo")
								},
							},
						},
					}, nil
				},
				ignoreMissingSecrets: true,
			},
			err: nil,
		},
		"invalid template var: start with a number": {
			command: RunCommand{
				environment: &environment{
					osStat:  osStatNotExist,
					envFile: "secrethub.env",
					templateVars: map[string]string{
						"0foo": "value",
					},
					envar: map[string]string{},
				},
			},
			err: ErrInvalidTemplateVar("0foo"),
		},
		"invalid template var: illegal character": {
			command: RunCommand{
				environment: &environment{
					osStat:  osStatNotExist,
					envFile: "secrethub.env",
					templateVars: map[string]string{
						"foo@bar": "value",
					},
					envar: map[string]string{},
				},
			},
			err: ErrInvalidTemplateVar("foo@bar"),
		},
		"os env secret not found": {
			command: RunCommand{
				command: []string{"echo", "test"},
				io:      fakeui.NewIO(t),
				environment: &environment{
					osEnv:  []string{"TEST=secrethub://nonexistent/secret/path"},
					osStat: osStatNotExist,
				},
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						SecretService: &fakeclient.SecretService{
							VersionService: &fakeclient.SecretVersionService{
								GetWithDataFunc: func(path string) (*api.SecretVersion, error) {
									return nil, api.ErrSecretNotFound
								},
							},
						},
					}, nil
				},
			},
			err: api.ErrSecretNotFound,
		},
		"os env secret not found ignored": {
			command: RunCommand{
				ignoreMissingSecrets: true,
				command:              []string{"echo", "test"},
				io:                   fakeui.NewIO(t),
				environment: &environment{
					osEnv:  []string{"TEST=secrethub://nonexistent/secret/path"},
					osStat: osStatNotExist,
				},
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						SecretService: &fakeclient.SecretService{
							VersionService: &fakeclient.SecretVersionService{
								GetWithDataFunc: func(path string) (*api.SecretVersion, error) {
									return nil, api.ErrSecretNotFound
								},
							},
						},
					}, nil
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := tc.command.Run()
			assert.Equal(t, err, tc.err)
		})
	}
}

func readFileFunc(name string, content string) func(string) ([]byte, error) {
	return func(filename string) ([]byte, error) {
		if filename == name {
			return []byte(content), nil
		}
		return nil, os.ErrNotExist
	}
}

func osStatFunc(name string, err error) func(string) (os.FileInfo, error) {
	return func(filename string) (os.FileInfo, error) {
		if name == filename {
			return nil, err
		}
		return nil, os.ErrNotExist
	}
}

func TestRunCommand_environment(t *testing.T) {
	testUUID1 := uuid.New()
	testUUID2 := uuid.New()

	cases := map[string]struct {
		command         RunCommand
		expectedEnv     []string
		expectedSecrets []string
		err             error
	}{
		"invalid template syntax": {
			command: RunCommand{
				command: []string{"echo", "test"},
				environment: &environment{
					osStat:          osStatFunc("secrethub.env", nil),
					readFile:        readFileFunc("secrethub.env", "TEST={{path/to/secret}"),
					envFile:         "secrethub.env",
					templateVersion: "2",
				},
			},
			err: ErrParsingTemplate("secrethub.env", "template syntax error at 1:23: expected the closing of a secret tag `}}`, but reached the end of the template. (template.secret_tag_not_closed) "),
		},
		"default env file does not exist": {
			command: RunCommand{
				environment: &environment{
					osStat: osStatFunc("secrethub.env", os.ErrNotExist),
				},
			},
		},
		"default env file exists but cannot be read": {
			command: RunCommand{
				environment: &environment{
					osStat: osStatFunc("secrethub.env", os.ErrPermission),
				},
			},
			err: ErrReadDefaultEnvFile(defaultEnvFile, os.ErrPermission),
		},
		"custom env file does not exist": {
			command: RunCommand{
				environment: &environment{
					envFile: "foo.env",
					readFile: func(filename string) ([]byte, error) {
						if filename == "foo.env" {
							return nil, &os.PathError{Op: "open", Path: "foo.env", Err: os.ErrNotExist}
						}
						return nil, nil
					},
				},
			},
			err: ErrCannotReadFile("foo.env", &os.PathError{Op: "open", Path: "foo.env", Err: os.ErrNotExist}),
		},
		"custom env file success": {
			command: RunCommand{
				environment: &environment{
					osStat:          osStatFunc("foo.env", nil),
					envFile:         "foo.env",
					templateVersion: "2",
					readFile:        readFileFunc("foo.env", "TEST=test"),
				},
			},
			expectedEnv: []string{"TEST=test"},
		},
		"env file secret does not exist": {
			command: RunCommand{
				command: []string{"echo", "test"},
				environment: &environment{
					osStat:          osStatFunc("secrethub.env", nil),
					readFile:        readFileFunc("secrethub.env", "TEST= {{ unexistent/secret/path }}"),
					envFile:         "secrethub.env",
					templateVersion: "2",
				},
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						SecretService: &fakeclient.SecretService{
							VersionService: &fakeclient.SecretVersionService{
								GetWithDataFunc: func(path string) (*api.SecretVersion, error) {
									return nil, api.ErrSecretNotFound
								},
							},
						},
					}, nil
				},
			},
			err: ErrParsingTemplate("secrethub.env", api.ErrSecretNotFound),
		},
		"envar flag has precedence over env file": {
			command: RunCommand{
				environment: &environment{
					osStat:   osStatFunc("secrethub.env", nil),
					readFile: readFileFunc("secrethub.env", "TEST=aaa"),
					envFile:  "secrethub.env",
					envar: map[string]string{
						"TEST": "test/test/test",
					},
					templateVersion: "2",
				},
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						SecretService: &fakeclient.SecretService{
							VersionService: &fakeclient.SecretVersionService{
								GetWithDataFunc: func(path string) (*api.SecretVersion, error) {
									return &api.SecretVersion{Data: []byte("bbb")}, nil
								},
							},
						},
					}, nil
				},
			},
			expectedSecrets: []string{"bbb"},
			expectedEnv:     []string{"TEST=bbb"},
		},
		"env file has precedence over secrets-dir flag": {
			command: RunCommand{
				environment: &environment{
					newClient: func() (secrethub.ClientInterface, error) {
						return fakeclient.Client{
							DirService: &fakeclient.DirService{
								GetTreeFunc: func(path string, depth int, ancestors bool) (*api.Tree, error) {
									return &api.Tree{
										ParentPath: "namespace",
										RootDir: &api.Dir{
											DirID: testUUID1,
											Name:  "repo",
										},
										Secrets: map[uuid.UUID]*api.Secret{
											testUUID2: {
												SecretID: testUUID2,
												DirID:    testUUID1,
												Name:     "foo",
											},
										},
									}, nil
								},
							},
						}, nil
					},
					secretsDir:                   "namespace/repo",
					dontPromptMissingTemplateVar: true,
					templateVersion:              "2",
					osEnv:                        []string{"FOO=bbb"},
					osStat:                       osStatFunc("secrethub.env", nil),
					readFile:                     readFileFunc("secrethub.env", "FOO= {{ other/secret/path }}"),
				},
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						SecretService: &fakeclient.SecretService{
							VersionService: &fakeclient.SecretVersionService{
								GetWithDataFunc: func(path string) (*api.SecretVersion, error) {
									if path == "namespace/repo/foo" {
										return &api.SecretVersion{Data: []byte("aaa")}, nil
									} else if path == "other/secret/path" {
										return &api.SecretVersion{Data: []byte("bbb")}, nil
									}
									return nil, api.ErrSecretNotFound
								},
							},
						},
					}, nil
				},
			},
			expectedSecrets: []string{"bbb"},
			expectedEnv:     []string{"FOO=bbb"},
		},
		"secrets-dir flag has precedence over os environment": {
			command: RunCommand{
				environment: &environment{
					newClient: func() (secrethub.ClientInterface, error) {
						return fakeclient.Client{
							DirService: &fakeclient.DirService{
								GetTreeFunc: func(path string, depth int, ancestors bool) (*api.Tree, error) {
									return &api.Tree{
										ParentPath: "namespace",
										RootDir: &api.Dir{
											DirID: testUUID1,
											Name:  "repo",
										},
										Secrets: map[uuid.UUID]*api.Secret{
											testUUID2: {
												SecretID: testUUID2,
												DirID:    testUUID1,
												Name:     "foo",
											},
										},
									}, nil
								},
							},
						}, nil
					},
					secretsDir:                   "namespace/repo",
					dontPromptMissingTemplateVar: true,
					templateVersion:              "2",
					osEnv:                        []string{"FOO=bbb"},
					osStat:                       osStatFunc("secrethub.env", os.ErrNotExist),
				},
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						SecretService: &fakeclient.SecretService{
							VersionService: &fakeclient.SecretVersionService{
								GetWithDataFunc: func(path string) (*api.SecretVersion, error) {
									if path == "namespace/repo/foo" {
										return &api.SecretVersion{Data: []byte("aaa")}, nil
									}
									return nil, api.ErrSecretNotFound
								},
							},
						},
					}, nil
				},
			},
			expectedSecrets: []string{"aaa"},
			expectedEnv:     []string{"FOO=aaa"},
		},
		// TODO Add test case for: envar flag has precedence over secret reference - requires refactoring of fakeclient
		"secret reference has precedence over secrets-dir flag": {
			command: RunCommand{
				environment: &environment{
					newClient: func() (secrethub.ClientInterface, error) {
						return fakeclient.Client{
							DirService: &fakeclient.DirService{
								GetTreeFunc: func(path string, depth int, ancestors bool) (*api.Tree, error) {
									return &api.Tree{
										ParentPath: "namespace",
										RootDir: &api.Dir{
											DirID: testUUID1,
											Name:  "repo",
										},
										Secrets: map[uuid.UUID]*api.Secret{
											testUUID2: {
												SecretID: testUUID2,
												DirID:    testUUID1,
												Name:     "foo",
											},
										},
									}, nil
								},
							},
						}, nil
					},
					secretsDir:                   "namespace/repo",
					dontPromptMissingTemplateVar: true,
					templateVersion:              "2",
					osEnv:                        []string{"FOO=secrethub://test/test/test"},
					osStat:                       osStatFunc("secrethub.env", os.ErrNotExist),
				},
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						SecretService: &fakeclient.SecretService{
							VersionService: &fakeclient.SecretVersionService{
								GetWithDataFunc: func(path string) (*api.SecretVersion, error) {
									if path == "test/test/test" {
										return &api.SecretVersion{Data: []byte("bbb")}, nil
									} else if path == "namespace/repo/foo" {
										return &api.SecretVersion{Data: []byte("aaa")}, nil
									}
									return nil, api.ErrSecretNotFound
								},
							},
						},
					}, nil
				},
			},
			expectedSecrets: []string{"bbb"},
			expectedEnv:     []string{"FOO=bbb"},
		},
		"secret reference has precedence over .env file": {
			command: RunCommand{
				environment: &environment{
					osStat:                       osStatFunc("secrethub.env", nil),
					readFile:                     readFileFunc("secrethub.env", "TEST=aaa"),
					dontPromptMissingTemplateVar: true,
					templateVersion:              "2",
					osEnv:                        []string{"TEST=secrethub://test/test/test"},
				},
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						SecretService: &fakeclient.SecretService{
							VersionService: &fakeclient.SecretVersionService{
								GetWithDataFunc: func(path string) (*api.SecretVersion, error) {
									return &api.SecretVersion{Data: []byte("bbb")}, nil
								},
							},
						},
					}, nil
				},
			},
			expectedSecrets: []string{"bbb"},
			expectedEnv:     []string{"TEST=bbb"},
		},
		".env file has precedence over other os variables": {
			command: RunCommand{
				environment: &environment{
					osStat:                       osStatFunc("secrethub.env", nil),
					readFile:                     readFileFunc("secrethub.env", "TEST=aaa"),
					dontPromptMissingTemplateVar: true,
					templateVersion:              "2",
				},
				osEnv: []string{"TEST=bbb"},
			},
			expectedSecrets: []string{},
			expectedEnv:     []string{"TEST=aaa"},
		},
		".env file secret has precedence over other os variables": {
			command: RunCommand{
				environment: &environment{
					osStat:                       osStatFunc("secrethub.env", nil),
					readFile:                     readFileFunc("secrethub.env", "TEST={{path/to/secret}}"),
					dontPromptMissingTemplateVar: true,
					templateVersion:              "2",
				},
				osEnv: []string{"TEST=bbb"},
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						SecretService: &fakeclient.SecretService{
							VersionService: &fakeclient.SecretVersionService{
								GetWithDataFunc: func(path string) (*api.SecretVersion, error) {
									return &api.SecretVersion{Data: []byte("aaa")}, nil
								},
							},
						},
					}, nil
				},
			},
			expectedSecrets: []string{"aaa"},
			expectedEnv:     []string{"TEST=aaa"},
		},
		"ignore missing secrets": {
			command: RunCommand{
				ignoreMissingSecrets: true,
				environment: &environment{
					osStat:   osStatFunc("secrethub.env", nil),
					envFile:  "secrethub.env",
					readFile: readFileFunc("secrethub.env", ""),
					envar: map[string]string{
						"TEST": "test/test/test",
					},
					templateVersion: "2",
				},
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						SecretService: &fakeclient.SecretService{
							VersionService: &fakeclient.SecretVersionService{
								GetWithDataFunc: func(path string) (*api.SecretVersion, error) {
									return nil, api.ErrSecretNotFound
								},
							},
						},
					}, nil
				},
			},
			expectedEnv:     []string{"TEST="},
			expectedSecrets: []string{""},
		},
		"--no-prompt": {
			command: RunCommand{
				noMasking: true,
				environment: &environment{
					osStat:                       osStatFunc("secrethub.env", nil),
					readFile:                     readFileFunc("secrethub.env", "TEST = {{ test/$variable/test }}"),
					dontPromptMissingTemplateVar: true,
					envFile:                      "secrethub.env",
					templateVersion:              "2",
				},
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						SecretService: &fakeclient.SecretService{
							VersionService: &fakeclient.SecretVersionService{
								GetWithDataFunc: func(path string) (*api.SecretVersion, error) {
									return nil, api.ErrSecretNotFound
								},
							},
						},
					}, nil
				},
			},
			err: ErrParsingTemplate("secrethub.env", tpl.ErrTemplateVarNotFound("variable")),
		},
		"template var set in os environment": {
			command: RunCommand{
				noMasking: true,
				environment: &environment{
					osStat:                       osStatFunc("secrethub.env", nil),
					readFile:                     readFileFunc("secrethub.env", "TEST = {{ test/$variable/test }}"),
					dontPromptMissingTemplateVar: true,
					templateVersion:              "2",
					osEnv:                        []string{"SECRETHUB_VAR_VARIABLE=test"},
				},
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						SecretService: &fakeclient.SecretService{
							VersionService: &fakeclient.SecretVersionService{
								GetWithDataFunc: func(path string) (*api.SecretVersion, error) {
									return nil, api.ErrSecretNotFound
								},
							},
						},
					}, nil
				},
			},
			err: ErrParsingTemplate("secrethub.env", api.ErrSecretNotFound),
		},
		"template var set by flag": {
			command: RunCommand{
				command: []string{"/bin/sh", "./test.sh"},
				environment: &environment{
					osStat:                       osStatFunc("secrethub.env", nil),
					readFile:                     readFileFunc("secrethub.env", "TEST = {{ test/$variable/test }}"),
					dontPromptMissingTemplateVar: true,
					templateVersion:              "2",
					templateVars:                 map[string]string{"variable": "test"},
				},
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						SecretService: &fakeclient.SecretService{
							VersionService: &fakeclient.SecretVersionService{
								GetWithDataFunc: func(path string) (*api.SecretVersion, error) {
									return nil, api.ErrSecretNotFound
								},
							},
						},
					}, nil
				},
			},
			err: ErrParsingTemplate("secrethub.env", api.ErrSecretNotFound),
		},
		"template var set by flag has precedence over var set by environment": {
			command: RunCommand{
				command: []string{"/bin/sh", "./test.sh"},
				environment: &environment{
					osEnv:                        []string{"SECRETHUB_VAR_VARIABLE=bar"},
					osStat:                       osStatFunc("secrethub.env", nil),
					readFile:                     readFileFunc("secrethub.env", "TEST=$variable"),
					dontPromptMissingTemplateVar: true,
					templateVersion:              "2",
					templateVars:                 map[string]string{"variable": "foo"},
				},
				osEnv: []string{"SECRETHUB_VAR_VARIABLE=bar"},
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						SecretService: &fakeclient.SecretService{
							VersionService: &fakeclient.SecretVersionService{
								GetWithDataFunc: func(path string) (*api.SecretVersion, error) {
									return nil, api.ErrSecretNotFound
								},
							},
						},
					}, nil
				},
			},
			expectedEnv: []string{"TEST=foo", "SECRETHUB_VAR_VARIABLE=bar"},
		},
		"v1 template syntax success": {
			command: RunCommand{
				command: []string{"/bin/sh", "./test.sh"},
				environment: &environment{
					osStat:          osStatFunc("secrethub.env", nil),
					readFile:        readFileFunc("secrethub.env", "TEST= ${path/to/secret}"),
					templateVersion: "1",
				},
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						SecretService: &fakeclient.SecretService{
							VersionService: &fakeclient.SecretVersionService{
								GetWithDataFunc: func(path string) (*api.SecretVersion, error) {
									return &api.SecretVersion{Data: []byte("bbb")}, nil
								},
							},
						},
					}, nil
				},
			},
			expectedSecrets: []string{"bbb"},
			expectedEnv:     []string{"TEST=bbb"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			env, secrets, err := tc.command.sourceEnvironment()

			sort.Strings(env)
			sort.Strings(tc.expectedEnv)
			sort.Strings(secrets)
			sort.Strings(tc.expectedSecrets)

			assert.Equal(t, env, tc.expectedEnv)
			assert.Equal(t, secrets, tc.expectedSecrets)
			assert.Equal(t, err, tc.err)
		})
	}
}

func TestRunCommand_RunWithFile(t *testing.T) {
	readFileWithContent := func(content string) func(string) ([]byte, error) {
		return func(_ string) ([]byte, error) {
			return []byte(content), nil
		}
	}

	osStatOnlySecretHubEnv := func(filename string) (info os.FileInfo, err error) {
		if filename == "secrethub.env" {
			return nil, nil
		}
		return nil, os.ErrNotExist
	}

	cases := map[string]struct {
		script         string
		command        RunCommand
		err            error
		expectedStdOut string
	}{
		"--no-masking flag": {
			script: "echo $TEST",
			command: RunCommand{
				command:   []string{"/bin/sh", "./test.sh"},
				noMasking: true,
				environment: &environment{
					osStat:   osStatOnlySecretHubEnv,
					readFile: readFileWithContent(""),
					envFile:  "secrethub.env",
					envar: map[string]string{
						"TEST": "test/test/test",
					},
					templateVersion: "2",
				},
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						SecretService: &fakeclient.SecretService{
							VersionService: &fakeclient.SecretVersionService{
								GetWithDataFunc: func(path string) (*api.SecretVersion, error) {
									return &api.SecretVersion{Data: []byte("bbb")}, nil
								},
							},
						},
					}, nil
				},
			},
			expectedStdOut: "bbb\n",
		},
		"secret masking": {
			script: "echo $TEST",
			command: RunCommand{
				command: []string{"/bin/sh", "./test.sh"},
				environment: &environment{
					osStat:   osStatOnlySecretHubEnv,
					envFile:  "secrethub.env",
					readFile: readFileWithContent(""),
					envar: map[string]string{
						"TEST": "test/test/test",
					},
					templateVersion: "2",
				},
				newClient: func() (secrethub.ClientInterface, error) {
					return fakeclient.Client{
						SecretService: &fakeclient.SecretService{
							VersionService: &fakeclient.SecretVersionService{
								GetWithDataFunc: func(path string) (*api.SecretVersion, error) {
									return &api.SecretVersion{Data: []byte("bbb")}, nil
								},
							},
						},
					}, nil
				},
			},
			expectedStdOut: maskString + "\n",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if tc.script != "" {
				scriptFile := filepath.Join(os.TempDir(), tc.command.command[1])
				err := ioutil.WriteFile(scriptFile, []byte(tc.script), os.ModePerm)
				if err != nil {
					log.Fatal("Cannot create file for test", err)
				}
				tc.command.command[1] = scriptFile
				defer os.Remove(scriptFile)
			}

			fakeIO := fakeui.NewIO(t)
			tc.command.io = fakeIO

			err := tc.command.Run()
			assert.Equal(t, err, tc.err)

			stdout, err := fakeIO.ReadStdout()
			assert.OK(t, err)
			assert.Equal(t, string(stdout), tc.expectedStdOut)
		})
	}
}

func TestTrimQuotes(t *testing.T) {
	cases := map[string]struct {
		in       string
		expected string
	}{
		"unquoted": {
			in:       `foo`,
			expected: `foo`,
		},
		"single quoted": {
			in:       `'foo'`,
			expected: `foo`,
		},
		"double quoted": {
			in:       `"foo"`,
			expected: `foo`,
		},
		"maintain quotes inside unquoted value": {
			in:       `{"foo":"bar"}`,
			expected: `{"foo":"bar"}`,
		},
		"empty string": {
			in:       "",
			expected: "",
		},
		"single quoted empty string": {
			in:       `''`,
			expected: ``,
		},
		"double qouted empty string": {
			in:       `""`,
			expected: ``,
		},
		"single quote wrapped in single quote": {
			in:       `''foo''`,
			expected: `'foo'`,
		},
		"single quote wrapped in double quote": {
			in:       `"'foo'"`,
			expected: `'foo'`,
		},
		"double quote wrapped in double quote": {
			in:       `""foo""`,
			expected: `"foo"`,
		},
		"double quote wrapped in single quote": {
			in:       `'"foo"'`,
			expected: `"foo"`,
		},
		"single quote opened but not closed": {
			in:       `'foo`,
			expected: `'foo`,
		},
		"double quote opened but not closed": {
			in:       `"foo`,
			expected: `"foo`,
		},
		"single quote closed but not opened": {
			in:       `foo'`,
			expected: `foo'`,
		},
		"double quote closed but not opened": {
			in:       `foo"`,
			expected: `foo"`,
		},
		"single quoted with inner leading whitespace": {
			in:       `' foo'`,
			expected: ` foo`,
		},
		"double quoted with inner leading whitespace": {
			in:       `" foo"`,
			expected: ` foo`,
		},
		"single quoted with inner trailing whitespace": {
			in:       `'foo '`,
			expected: `foo `,
		},
		"double quoted with inner trailing whitespace": {
			in:       `"foo "`,
			expected: `foo `,
		},

		// Trimming OUTER whitespace is explicitly not the responsibility of this function.
		"single quoted with outer leading whitespace": {
			in:       ` 'foo'`,
			expected: ` 'foo'`,
		},
		"double quoted with outer leading whitespace": {
			in:       ` "foo"`,
			expected: ` "foo"`,
		},
		"single quoted with outer trailing whitespace": {
			in:       `'foo' `,
			expected: `'foo' `,
		},
		"double quoted with outer trailing whitespace": {
			in:       `"foo" `,
			expected: `"foo" `,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			actual, _ := trimQuotes(tc.in)

			assert.Equal(t, actual, tc.expected)
		})
	}
}

func Test_parseKeyValueStringsToMap(t *testing.T) {
	input := []string{
		"A=B",
		"B",
		"=::=::\\",
	}

	parsableValues, unparsableValues := parseKeyValueStringsToMap(input)

	assert.Equal(t, parsableValues, map[string]string{
		"A": "B",
		"B": "",
	})
	assert.Equal(t, unparsableValues, []string{
		"=::=::\\",
	})
}
