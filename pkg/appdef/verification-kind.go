/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"strconv"
	"strings"
)

//go:generate stringer -type=VerificationKind -output=verification-kind_string.go

const (
	VerificationKind_EMail VerificationKind = iota
	VerificationKind_Phone
	VerificationKind_FakeLast
)

var VerificationKind_Any = []VerificationKind{VerificationKind_EMail, VerificationKind_Phone}

func (k VerificationKind) MarshalJSON() ([]byte, error) {
	var s string
	if k < VerificationKind_FakeLast {
		s = strconv.Quote(k.String())
	} else {
		const base = 10
		s = strconv.FormatUint(uint64(k), base)
	}
	return []byte(s), nil
}

// Renders an VerificationKind in human-readable form, without "VerificationKind_" prefix,
// suitable for debugging or error messages
func (k VerificationKind) TrimString() string {
	const pref = "VerificationKind_"
	return strings.TrimPrefix(k.String(), pref)
}

func (k *VerificationKind) UnmarshalJSON(data []byte) (err error) {
	text := string(data)
	if t, err := strconv.Unquote(text); err == nil {
		text = t
		for v := VerificationKind(0); v < VerificationKind_FakeLast; v++ {
			if v.String() == text {
				*k = v
				return nil
			}
		}
	}

	var i uint64
	const base, wordBits = 10, 16
	i, err = strconv.ParseUint(text, base, wordBits)
	if err == nil {
		*k = VerificationKind(i)
	}
	return err
}
