package fakes

import "errors"

type FakeVariableReader struct {
	Variables map[string]string
}

func (r FakeVariableReader) ReadVariable(name string) (string, error) {
	variable, ok := r.Variables[name]
	if !ok {
		return "", errors.New("variable not found: " + name)
	}
	return variable, nil
}
