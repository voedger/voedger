/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package states

import (
	"fmt"
	"strconv"
)

func (a AttributeKind) String() string {
	if (a < 0) || (a >= AttributeKindCount) {
		return fmt.Sprintf("AttributeKind(%d)", a)
	}
	return AttributeKindNames[a]
}

func (a AttributeKind) MarshalJson() ([]byte, error) {
	var data string
	if (a < 0) || (a >= AttributeKindCount) {
		const base = 10
		data = strconv.FormatInt(int64(a), base)
	} else {
		data = strconv.Quote(a.String())
	}
	return []byte(data), nil
}

var encodeAttributeKind map[string]AttributeKind = func() map[string]AttributeKind {
	m := make(map[string]AttributeKind)
	for value, name := range AttributeKindNames {
		m[name] = AttributeKind(value)
	}
	return m
}()

func (a *AttributeKind) UnmarshalJSON(data []byte) (err error) {
	if text, err := strconv.Unquote(string(data)); err == nil {
		if value, ok := encodeAttributeKind[text]; ok {
			*a = value
			return nil
		}
		return fmt.Errorf(fmtAttributeKindSyntaxError, text, strconv.ErrSyntax)
	}

	var value int64
	const base, byteBits = 10, 8
	value, err = strconv.ParseInt(string(data), base, byteBits)
	if err != nil {
		return err
	}

	*a = AttributeKind(value)
	return nil
}

func (s ActualStatus) String() string {
	if (s < 0) || (s >= ActualStatusCount) {
		return fmt.Sprintf("ActualStatus(%d)", s)
	}
	return ActualStatusNames[s]
}

func (s ActualStatus) MarshalJson() ([]byte, error) {
	var data string
	if (s < 0) || (s >= ActualStatusCount) {
		const base = 10
		data = strconv.FormatInt(int64(s), base)
	} else {
		data = strconv.Quote(s.String())
	}
	return []byte(data), nil
}

var encodeActualStatus map[string]ActualStatus = func() map[string]ActualStatus {
	m := make(map[string]ActualStatus)
	for value, name := range ActualStatusNames {
		m[name] = ActualStatus(value)
	}
	return m
}()

func (s *ActualStatus) UnmarshalJSON(data []byte) (err error) {
	if text, err := strconv.Unquote(string(data)); err == nil {
		if value, ok := encodeActualStatus[text]; ok {
			*s = value
			return nil
		}
		return fmt.Errorf(fmtActualStatusSyntaxError, text, strconv.ErrSyntax)
	}

	var value int64
	const base, byteBits = 10, 8
	value, err = strconv.ParseInt(string(data), base, byteBits)
	if err != nil {
		return err
	}

	*s = ActualStatus(value)
	return nil
}
