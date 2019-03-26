package configuration

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"

	"github.com/keylockerbv/secrethub-cli/internals/cli/posix"
	"github.com/mitchellh/mapstructure"

	"reflect"

	"time"

	"github.com/secrethub/secrethub-go/internals/api/uuid"
	"github.com/secrethub/secrethub-go/internals/errio"
	yaml "gopkg.in/yaml.v2"
)

var (
	errConfig = errio.Namespace("configuration")

	// ErrDecodeFailed is given when the config cannot be decoded.
	ErrDecodeFailed = errConfig.Code("decode_fail").Error("failed to decode config")
	// ErrEncodeFailed is given when the config cannot be encoded.
	ErrEncodeFailed = errConfig.Code("encode_fail").Error("failed to encode config")
	// ErrFileNotFound is given when the config file cannot be found.
	ErrFileNotFound = errConfig.Code("not_found").Error("config file not found")

	// ErrTypeNotSet is given when a config has no type specified
	ErrTypeNotSet = errConfig.Code("type_not_set").Error("field `type` of config is not set")
)

// ReadFromFile attempts to read a config as a struct from a file.
// It will unmarshal yaml and json into the destination.
func ReadFromFile(path string, destination interface{}) error {
	data, err := readFile(path)
	if err != nil {
		return errio.Error(err)
	}

	return Read(data, destination)
}

// Read attempts to read a config in a destination interface.
func Read(data []byte, destination interface{}) error {
	err := yaml.Unmarshal(data, destination)
	return errio.Error(err)
}

// ReadConfigurationDataFromFile retrieves the data and attempts to read the config as a ConfigMap
// from a file. This simplifies the process of determining to parse the config as a ConfigMap or to
// directly unmarshal into a struct.
func ReadConfigurationDataFromFile(path string) (ConfigMap, []byte, error) {
	data, err := readFile(path)
	if err != nil {
		return nil, nil, errio.Error(err)
	}

	configMap, err := ReadMap(data)
	return configMap, data, err
}

// ReadMap attempts to unmarshal a []byte it into the dest map.
// Both json and yaml are supported
func ReadMap(data []byte) (ConfigMap, error) {
	var dest ConfigMap

	// Supports both json and yaml
	if err := yaml.Unmarshal(data, &dest); err != nil {
		return nil, ErrDecodeFailed
	}

	return dest, nil
}

// ParseMap uses mapstructure to convert a ConfigMap into a struct
//
// For example, the following JSON:
//     {
//         "ssh_key": "/home/joris/.ssh/secrethub"
//     }
//
// Can be loaded in a struct of type:
//     type UserConfig struct {
// 	    SSHKeyPath string `json:"ssh_key,omitempty"`
//     }
//
// decodeHook is used to convert non-standard types into the correct format
func ParseMap(src *ConfigMap, dst interface{}) error {

	c := mapstructure.DecoderConfig{
		TagName:          "json",
		Result:           dst,
		WeaklyTypedInput: true,
		DecodeHook:       decodeHook}
	decoder, err := mapstructure.NewDecoder(&c)

	if err != nil {
		return errio.Error(err)
	}

	return decoder.Decode(src)
}

// decodeHook adds extra decoding functionality to parsing the map.
func decodeHook(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if t == reflect.TypeOf(uuid.UUID{}) && f == reflect.TypeOf(string("")) {
		return uuid.FromString(data.(string))
	}

	if t == reflect.TypeOf(time.Duration(0)) && f == reflect.TypeOf(string("")) {
		return time.ParseDuration(data.(string))
	}

	return data, nil
}

// WriteToFile attempts to marshal the src arg and write to a file at the given path.
// If the file is of an unsupported extension we cannot determine what type
// of encoding to use, so it defaults to writing encoded json to that file.
// JSON is written in indented 'pretty' format to allow for easy user editing.
func WriteToFile(src interface{}, path string, fileMode os.FileMode) error {
	path = strings.ToLower(path)

	var data []byte
	var err error
	switch {
	case strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".yaml"):
		data, err = yaml.Marshal(src)
		if err != nil {
			return err
		}
	case strings.HasSuffix(path, ".json"):
		data, err = json.MarshalIndent(src, "", "  ")
		if err != nil {
			return err
		}
	default:
		data, err = json.MarshalIndent(src, "", "  ")
		if err != nil {
			return ErrEncodeFailed
		}
	}

	return ioutil.WriteFile(path, posix.AddNewLine(data), fileMode)
}

func readFile(path string) ([]byte, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, ErrFileNotFound
	}

	return ioutil.ReadFile(path)
}

// ConfigMap is the type used for configurations that are still in a map format
// Map format is to make changes in structure with migrations possible
type ConfigMap map[string]interface{}

// GetVersion returns the version of the configuration file.
// If it is not set, it is assumed that is configuration version 1.
func (c ConfigMap) GetVersion() (int, error) {
	version, ok := c["version"]
	if !ok {
		// Version not set
		return 1, nil
	}

	ret, ok := version.(int)

	if !ok {
		return 0, errConfig.Code("version_wrong_type").Errorf("config value `version` has wrong type %T (actual) != int (expected)", version)
	}

	return ret, nil
}

// GetType returns the type of the configuration file
// If it is not set, it is assumed to have no type.
func (c ConfigMap) GetType() (string, error) {
	t, ok := c["type"]
	if !ok {
		// Config type not set
		return "", ErrTypeNotSet
	}

	ret, ok := t.(string)
	if !ok {
		return "", errConfig.Code("type_wrong_type").Errorf("config value `type` has wrong type %T (actual) != string (expected)", t)
	}
	return ret, nil
}
