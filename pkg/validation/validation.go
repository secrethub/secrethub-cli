package validation

import (
	"regexp"
	"unicode"

	"github.com/secrethub/secrethub-go/internals/errio"
)

// Errors
var (
	errPackage          = errio.Namespace("validation")
	ErrInvalidEnvarName = errPackage.Code("invalid_envar_name").ErrorPref("environment variable names may not contain NUL or = characters and may only contain characters of the portable character set defined in IEEE Std 1003.1: %s")
)

const (
	// posixEnvarPattern defines the regex for valid environment variable
	// names to be used in POSIX environments. Got the definition from
	// this StackOverflow post: https://stackoverflow.com/a/2821183
	posixEnvarPattern = "^[a-zA-Z_]{1,}[a-zA-Z0-9_]{0,}$"
)

var (
	// envarCharset is defines a unicode range table for characters that are allowed in
	// environment variable names on systems that conform to IEEE Std 1003.1-2017. Valid
	// characters are all characters in the portable character set, excluding the NUL
	// and = characters. See the specification for more information:
	//
	//		http://pubs.opengroup.org/onlinepubs/9699919799/basedefs/V1_chap06.html#tagtcjh_3
	//
	// See also this StackOverflow post:
	//
	// 		https://stackoverflow.com/a/2821183
	//
	// For a discussion on what to allow as envar names, see this Kubernetes pull request
	// and issue:
	//
	// 		https://github.com/kubernetes/kubernetes/pull/48986
	// 		https://github.com/kubernetes/kubernetes/issues/2707#issuecomment-285309156
	ieeeEnvarCharset = &unicode.RangeTable{
		R16: []unicode.Range16{
			{0x0007, 0x000d, 1},
			{0x0020, 0x003c, 1},
			{0x003e, 0x007e, 1},
		},
	}

	posixEnvarWhiteList = regexp.MustCompile(posixEnvarPattern)
)

// IsEnvarName returns true when the given string is a valid environment
// variable name according to IEEE Std 1003.1-2017, i.e. containing only
// characters of the Portable character set, excluding the NUL and = characters.
func IsEnvarName(name string) bool {
	return len(name) > 0 && hasOnlyCharset(ieeeEnvarCharset, name)
}

// hasOnlyCharset is a helper function to determine whether all characters of
// a string are within the given character set.
func hasOnlyCharset(rt *unicode.RangeTable, str string) bool {
	for _, r := range str {
		if !unicode.Is(rt, r) {
			return false
		}
	}

	return true
}

// IsEnvarNamePosix returns true when the given string is a POSIX compliant
// environment variable name, i.e. contains only letters, numbers and
// underscores, and does not start with a number.
func IsEnvarNamePosix(name string) bool {
	return posixEnvarWhiteList.MatchString(name)
}

// ValidateEnvarName validates an environment variable name using the
// default validation function.
func ValidateEnvarName(name string) error {
	if !IsEnvarName(name) {
		return ErrInvalidEnvarName(name)
	}

	return nil
}
