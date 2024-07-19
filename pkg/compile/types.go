/*
* Copyright (c) 2024-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */

package compile

import (
	"golang.org/x/tools/go/packages"

	"github.com/voedger/voedger/pkg/appdef"
)

// Result is a result of compilation
type Result struct {
	ModulePath    string              // module path of compiled module
	PkgFiles      map[string][]string // map of package path to list of file paths belonging to the package
	AppDef        appdef.IAppDef
	AppDefBuilder appdef.IAppDefBuilder
	NotFoundDeps  []string // list of not found dependencies faced during compilation
}

type loadedPackages struct {
	name         string
	packagePath  string
	modulePath   string
	rootPkgs     []*packages.Package
	importedPkgs map[string]*packages.Package // map of imported package path to *packages.Package
}
