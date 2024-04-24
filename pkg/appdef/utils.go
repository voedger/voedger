/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Returns is string is valid identifier and error if not
func ValidIdent(ident string) (bool, error) {
	if len(ident) < 1 {
		return false, ErrMissed("ident")
	}

	if l := len(ident); l > MaxIdentLen {
		return false, ErrOutOfBounds("ident «%s» too long (%d runes, max is %d)", ident, l, MaxIdentLen)
	}

	const (
		char_a    rune = 97
		char_A    rune = 65
		char_z    rune = 122
		char_Z    rune = 90
		char_0    rune = 48
		char_9    rune = 57
		char__    rune = 95
		char_Buck rune = 36
	)

	digit := func(r rune) bool { return (char_0 <= r) && (r <= char_9) }

	letter := func(r rune) bool { return ((char_a <= r) && (r <= char_z)) || ((char_A <= r) && (r <= char_Z)) }

	underScore := func(r rune) bool { return r == char__ }

	buck := func(r rune) bool { return r == char_Buck }

	for p, c := range ident {
		if !letter(c) && !underScore(c) && !buck(c) {
			if (p == 0) || !digit(c) {
				return false, ErrInvalid("ident «%s» has invalid char «%c» at pos %d", ident, c, p)
			}
		}
	}

	return true, nil
}

// Returns is string is valid field name and error if not
func ValidFieldName(ident FieldName) (bool, error) {
	return ValidIdent(ident)
}

// TODO: implement
// Parsing a URI Reference with a Regular Expression [RFC3986, app B](https://datatracker.ietf.org/doc/html/rfc3986#appendix-B)
func ValidPackagePath(path string) (bool, error) {
	if len(path) < 1 {
		return false, ErrMissed("package path")
	}
	return true, nil
}
