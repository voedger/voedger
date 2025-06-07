/*
* Copyright (c) 2024-present unTill Pro, Ltd.
* @author Alisher Nurmanov
 */

package compile

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/voedger/voedger/pkg/appdef/builder"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"golang.org/x/exp/maps"
	"golang.org/x/tools/go/packages"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/parser"
	"github.com/voedger/voedger/pkg/sys"
)

func compile(dir string, checkAppSchema bool) (*Result, error) {
	var errs []error
	notFoundDeps := make(map[string]struct{})

	loadedPkgs, err := loadPackages(dir, notFoundDeps)
	if err != nil {
		return nil, err
	}

	var pkgs []*parser.PackageSchemaAST
	importedStmts := make(map[string]parser.ImportStmt)
	pkgFiles := make(map[string][]string)

	// compile sys package first
	sysPackageAst, compileSysErrs, isSysDir := compileSysPackage(dir, loadedPkgs, importedStmts, pkgFiles, notFoundDeps)
	pkgs = append(pkgs, sysPackageAst...)
	errs = append(errs, compileSysErrs...)

	if !isSysDir {
		// compile working dir after sys package
		compileDirPackageAst, compileDirErrs := compileDir(loadedPkgs, dir, loadedPkgs.packagePath, nil, importedStmts, pkgFiles, notFoundDeps)
		pkgs = append(pkgs, compileDirPackageAst...)
		errs = append(errs, compileDirErrs...)
	}
	// add dummy app schema if no app schema found
	appSchemaExists := hasAppSchema(pkgs)
	if checkAppSchema && !appSchemaExists {
		errs = append(errs, ErrAppSchemaNotFound)
		return nil, errors.Join(errs...)
	}
	if !appSchemaExists {
		appPackageAst, err := getDummyAppPackageAst(maps.Values(importedStmts))
		if err != nil {
			errs = append(errs, err)
		}
		pkgs = append(pkgs, appPackageAst)
		addMissingUses(appPackageAst, getUseStmts(maps.Values(importedStmts)))
	}

	// remove nil packages
	nonNilPackages := make([]*parser.PackageSchemaAST, 0, len(pkgs))
	for _, p := range pkgs {
		if p != nil {
			nonNilPackages = append(nonNilPackages, p)
		}
	}
	// build app schema
	appAst, err := parser.BuildAppSchema(nonNilPackages)
	if err != nil {
		errs = append(errs, coreutils.SplitErrors(err)...)
	}

	result := &Result{
		ModulePath:   loadedPkgs.packagePath,
		PkgFiles:     pkgFiles,
		NotFoundDeps: maps.Keys(notFoundDeps),
	}
	// build app defs from app schema
	if appAst != nil {
		builder := builder.New()
		if err := parser.BuildAppDefs(appAst, builder); err != nil {
			errs = append(errs, err)
		}
		result.AppDefBuilder = builder
		result.AppDef, err = builder.Build()
		if err != nil {
			errs = append(errs, err)
		}
		if len(errs) == 0 {
			if logger.IsVerbose() {
				logger.Verbose("compiling succeeded")
			}
		}
	}
	return result, errors.Join(errs...)
}

func compileSysPackage(dir string, loadedPkgs *loadedPackages, importedStmts map[string]parser.ImportStmt, pkgFiles map[string][]string, notFoundDeps map[string]struct{}) (pkgAsts []*parser.PackageSchemaAST, errs []error, isSysDir bool) {
	if loadedPkgs.modulePath == VoedgerPath {
		rootPkgPath := loadedPkgs.rootPkgs[0].PkgPath
		relPath := rootPkgPath[len(VoedgerPath):]
		baseDir := dir
		if len(relPath) > 0 {
			baseDir = dir[:len(dir)-len(relPath)]
		}
		sysPkgDir := filepath.Join(baseDir, "pkg/sys")
		if dir == sysPkgDir {
			isSysDir = true
		}
		pkgAsts, errs = compileDir(loadedPkgs, sysPkgDir, appdef.SysPackage, nil, importedStmts, pkgFiles, notFoundDeps)
		return
	}
	pkgAsts, errs = compileDependency(loadedPkgs, appdef.SysPackage, nil, importedStmts, pkgFiles, notFoundDeps)
	return
}

func hasAppSchema(packages []*parser.PackageSchemaAST) bool {
	for _, p := range packages {
		if p != nil {
			for _, f := range p.Ast.Statements {
				if f.Application != nil {
					return true
				}
			}
		}
	}
	return false
}

func getDummyAppPackageAst(imports []parser.ImportStmt) (*parser.PackageSchemaAST, error) {
	fileAst := &parser.FileSchemaAST{
		FileName: sysSchemaFileName,
		Ast: &parser.SchemaAST{
			Imports: imports,
			Statements: []parser.RootStatement{
				{
					Application: &parser.ApplicationStmt{
						Name: DummyAppName,
					},
				},
			},
		},
	}
	return parser.BuildPackageSchema(DummyAppName, []*parser.FileSchemaAST{fileAst})
}

func getUseStmts(imports []parser.ImportStmt) []parser.UseStmt {
	uses := make([]parser.UseStmt, len(imports))
	for i, imp := range imports {
		use := parser.Ident(filepath.Base(imp.Name))
		if imp.Alias != nil {
			use = *imp.Alias
		}
		uses[i] = parser.UseStmt{
			Name: use,
		}
	}
	return uses
}

func addMissingUses(appPkg *parser.PackageSchemaAST, uses []parser.UseStmt) {
	for _, f := range appPkg.Ast.Statements {
		if f.Application != nil {
			for _, use := range uses {
				found := false
				for _, useInApp := range f.Application.Uses {
					if useInApp.Name == use.Name {
						found = true
						break
					}
				}
				if !found {
					f.Application.Uses = append(f.Application.Uses, use)
				}
			}
		}
	}
}

// checkImportedStmts checks if qpn is already imported. If not, it adds it to importedStmts
func checkImportedStmts(qpn string, alias *parser.Ident, importedStmts map[string]parser.ImportStmt) bool {
	aliasPtr := alias
	// workaround for sys package
	if qpn == appdef.SysPackage || qpn == sys.PackagePath {
		qpn = appdef.SysPackage
		alias := parser.Ident(qpn)
		aliasPtr = &alias
	}
	if _, exists := importedStmts[qpn]; exists {
		return false
	}
	importedStmts[qpn] = parser.ImportStmt{
		Name:  qpn,
		Alias: aliasPtr,
	}
	return true
}

func compileDir(loadedPkgs *loadedPackages, dir, packagePath string, alias *parser.Ident, importedStmts map[string]parser.ImportStmt, pkgFiles map[string][]string, notFoundDeps map[string]struct{}) (packages []*parser.PackageSchemaAST, errs []error) {
	if ok := checkImportedStmts(packagePath, alias, importedStmts); !ok {
		return
	}
	if logger.IsVerbose() {
		logger.Verbose("compiling " + dir)
	}

	packageAst, fileNames, err := parser.ParsePackageDirCollectingFiles(packagePath, coreutils.NewPathReader(dir), "")
	if err != nil {
		errs = append(errs, coreutils.SplitErrors(err)...)
	}
	// collect all the files that belong to the package
	for _, f := range fileNames {
		pkgFiles[packagePath] = append(pkgFiles[packagePath], filepath.Join(dir, f))
	}
	// iterate over all imports and compile them as well
	var compileDepErrs []error
	var importedPackages []*parser.PackageSchemaAST
	if packageAst != nil {
		importedPackages, compileDepErrs = compileDependencies(loadedPkgs, packageAst.Ast.Imports, importedStmts, pkgFiles, notFoundDeps)
		errs = append(errs, compileDepErrs...)
	}
	packages = append([]*parser.PackageSchemaAST{packageAst}, importedPackages...)
	return
}

func compileDependencies(loadedPkgs *loadedPackages, imports []parser.ImportStmt, importedStmts map[string]parser.ImportStmt, pkgFiles map[string][]string, notFoundDeps map[string]struct{}) (packages []*parser.PackageSchemaAST, errs []error) {
	for _, imp := range imports {
		dependentPackages, compileDepErrs := compileDependency(loadedPkgs, imp.Name, imp.Alias, importedStmts, pkgFiles, notFoundDeps)
		errs = append(errs, compileDepErrs...)
		packages = append(packages, dependentPackages...)
	}
	return
}

func compileDependency(loadedPkgs *loadedPackages, depURL string, alias *parser.Ident, importedStmts map[string]parser.ImportStmt, pkgFiles map[string][]string, notFoundDeps map[string]struct{}) (packages []*parser.PackageSchemaAST, errs []error) {
	// workaround for sys package
	depURLToFind := depURL
	if depURL == appdef.SysPackage {
		depURLToFind = sys.PackagePath
	}
	path, err := localPath(loadedPkgs, depURLToFind, notFoundDeps)
	if err != nil {
		errs = append(errs, err)
	}
	if logger.IsVerbose() {
		logger.Verbose(fmt.Sprintf("dependency: %s\nlocation: %s\n", depURL, path))
	}
	var compileDirErrs []error
	packages, compileDirErrs = compileDir(loadedPkgs, path, depURL, alias, importedStmts, pkgFiles, notFoundDeps)
	errs = append(errs, compileDirErrs...)
	return
}

func loadPackages(dir string, notFoundDeps map[string]struct{}) (*loadedPackages, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedDeps | packages.NeedModule,
		Dir:  dir,
	}

	rootPkgs, err := packages.Load(cfg)
	if err != nil {
		return nil, err
	}

	importedPkgs := allImportedPackages(rootPkgs)
	if len(rootPkgs) > 0 && rootPkgs[0].Module != nil {
		return &loadedPackages{
			importedPkgs: importedPkgs,
			rootPkgs:     rootPkgs,
			modulePath:   rootPkgs[0].Module.Path,
			packagePath:  rootPkgs[0].PkgPath,
			name:         rootPkgs[0].Name,
		}, nil
	}
	notFoundDeps[dir] = struct{}{}
	return nil, fmt.Errorf("%s: cannot find module path. Hint: if there is a go.work in upstream dirs then try to add the path to the root go.work", dir)
}

func allImportedPackages(initialPkgs []*packages.Package) (importedPkgs map[string]*packages.Package) {
	importedPkgs = make(map[string]*packages.Package)
	initialMap := make(map[string]*packages.Package)

	for _, p := range initialPkgs {
		initialMap[p.ID] = p
	}

	var appendImportedPackages func(res map[string]*packages.Package, pkgs map[string]*packages.Package)
	appendImportedPackages = func(res map[string]*packages.Package, pkgs map[string]*packages.Package) {
		for _, pkg := range pkgs {
			// if res already contains pkg.ID, skip it
			if _, ok := res[pkg.ID]; ok {
				continue
			}
			res[pkg.ID] = pkg
			appendImportedPackages(res, pkg.Imports)
		}
	}
	appendImportedPackages(importedPkgs, initialMap)

	// Remove initial packages
	for k := range initialMap {
		delete(importedPkgs, k)
	}

	return importedPkgs
}

// localPath returns local path of the dependency
// E.g. github.com/voedger/voedger/pkg/sys => /home/user/go/pkg/mod/github.com/voedger/voedger@v0.0.0-20231103100658-8d2fb878c2f9/pkg/sys
func localPath(loadedPkgs *loadedPackages, depURL string, notFoundDeps map[string]struct{}) (localDepPath string, err error) {
	if logger.IsVerbose() {
		logger.Verbose(fmt.Sprintf("resolving dependency %s ...", depURL))
	}
	// find in root packages
	path := getLocalPathOfTheDep(loadedPkgs.rootPkgs, depURL)
	if path != "" {
		return path, nil
	}
	// find in imported packages
	path = getLocalPathOfTheDep(maps.Values(loadedPkgs.importedPkgs), depURL)
	if path != "" {
		return path, nil
	}
	notFoundDeps[depURL] = struct{}{}
	return "", fmt.Errorf("%[1]s: cannot find dependency. Ensure you have an import \"_ %[1]s\" in golang sources", depURL)
}

func getLocalPathOfTheDep(pkgs []*packages.Package, depURL string) string {
	for _, pkg := range pkgs {
		if pkg.PkgPath == depURL {
			if len(pkg.GoFiles) > 0 {
				return filepath.Dir(pkg.GoFiles[0])
			}
		}
	}
	return ""
}
