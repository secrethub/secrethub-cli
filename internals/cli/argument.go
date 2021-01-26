package cli

import (
	"net/url"
)

type Argument struct {
	Value       ArgValue
	Name        string
	Required    bool
	Placeholder string
	Description string
	Hidden      bool
}

type ArgValue interface {
	Set(string) error
}

func ArgumentRegister(params []Argument, args []string) error {
	for i, arg := range args {
		err := params[i].Value.Set(arg)
		if err != nil {
			return err
		}
	}
	return nil
}

func ArgumentArrRegister(params Argument, args []string) error {
	for _, arg := range args {
		err := params.Value.Set(arg)
		if err != nil {
			return err
		}
	}
	return nil
}

type StringValue struct {
	Value string
}

func (s *StringValue) Set(replacer string) error {
	s.Value = replacer
	return nil
}

type StringListValue []string

func (s *StringListValue) Set(replacer string) error {
	*s = append(*s, replacer)
	return nil
}

type URLValue struct {
	*url.URL
}

func (s *URLValue) Set(replacer string) error {
	var err error
	s.URL, err = url.Parse(replacer)
	return err
}

func (s *URLValue) String() string {
	if s.URL == nil {
		return ""
	}
	return s.URL.String()
}

type ByteValue []byte

func (s *ByteValue) Set(replacer string) error {
	*s = []byte(replacer)
	return nil
}

func getRequired(params []Argument) int {
	required := 0
	for _, arg := range params {
		if arg.Required {
			required++
		}
	}
	return required
}
