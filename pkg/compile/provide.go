/*
* Copyright (c) 2024-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */

package compile

// Compile loads a synthetic AppDef from a folder
func Compile(dir string) (*Result, error) {
	return compile(dir, false)
}

// CompileNoDummyApp loads a synthetic AppDef from a folder without adding a dummy app schema
func CompileNoDummyApp(dir string) (*Result, error) {
	return compile(dir, true)
}
