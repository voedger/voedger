/*
* Copyright (c) 2024-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */

package compile

// Compile loads a synthetic AppDef from a folder
func Compile(dir string, checkAppSchema bool) (*Result, error) {
	return compile(dir, checkAppSchema)
}
