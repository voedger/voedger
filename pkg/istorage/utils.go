/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package istorage

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/voedger/voedger/pkg/appdef"
)

func NewSafeAppName(appQName appdef.AppQName, uniqueFunc func(name string) (bool, error)) (san SafeAppName, err error) {
	appName := appQName.String()
	appName = strings.ToLower(appName)

	removeWrongCharsRegexp := regexp.MustCompile("[^a-z0-9]+")
	appName = removeWrongCharsRegexp.ReplaceAllString(appName, "")

	if len(appName) == 0 {
		appName = getUUID()
	} else if len(appName) > MaxSafeNameLength {
		appName = appName[:MaxSafeNameLength]
	}

	if unicode.IsDigit([]rune(appName)[0]) {
		appName = "a" + string([]rune(appName)[1:]) // replace the first letter for the case if the entire name is uuid to make tests work expecting the string length is 32 bytes
	}

	for i := 0; i < maxMatchedOccurances; i++ {
		ok, err := uniqueFunc(appName)
		if err != nil {
			return san, err
		}
		if ok {
			return SafeAppName{appName}, nil
		}
		appName = mutateAppName(appName)
	}
	return san, ErrNoSafeAppName
}

func mutateAppName(name string) string {
	uuid := getUUID()
	idxToInsertUUIDAt := len(name)
	if idxToInsertUUIDAt+len(uuid) > MaxSafeNameLength {
		idxToInsertUUIDAt = MaxSafeNameLength - len(uuid)
	}
	return name[:idxToInsertUUIDAt] + uuid
}

func getUUID() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}

func (san SafeAppName) String() string {
	return san.name
}

// nolint
func (san SafeAppName) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(san.name)), nil
}

// need to marshal map[SafeAppName]any
// nolint
func (san SafeAppName) MarshalText() (text []byte, err error) {
	return []byte(san.name), nil
}

func (san *SafeAppName) UnmarshalJSON(text []byte) (err error) {
	str, err := strconv.Unquote(string(text))
	if err != nil {
		return err
	}
	*san = SafeAppName{name: str}
	return nil
}

// need to unmarshal map[SafeAppName]any
// golang json looks on UnmarshalText presence only on unmarshal map[SafeAppName]any. UnmarshalJSON() will be used anyway
// but no UnmarshalText -> fail to unmarshal map[SafeAppName]any
// see https://github.com/golang/go/issues/29732
func (san *SafeAppName) UnmarshalText(text []byte) error {
	// notest
	return nil
}

// used in tests only
// need to make different keyspaces when few integration tests run on the same non-memory storage. E.g. on `go test ./...` in github
// otherwise tests from different packages will conflict
// see https://dev.untill.com/projects/#!638565
func (san *SafeAppName) ApplyKeyspaceIsolationSuffix(suffix string) {
	if len(suffix) == 0 {
		return
	}
	if !SafeAppNameRegexp.MatchString(suffix) {
		panic("keyspace isolation suffix must match regexp " + SafeAppNameRegexp.String())
	}
	if len(san.name+suffix) > MaxSafeNameLength {
		panic("too long safe name including suffix")
	}
	san.name += suffix
}
