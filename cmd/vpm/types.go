/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package main

type vpmParams struct {
	WorkingDir string
	TargetDir  string
}

// packageFiles is a map of package name to a list of files that belong to the package
type packageFiles map[string][]string
