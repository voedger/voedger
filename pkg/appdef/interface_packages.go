/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

type IWithPackages interface {
	// Returns package path by package local name.
	//
	// Returns empty string if not found
	PackageFullPath(localName string) string

	// Returns package local name by package path.
	//
	// Returns empty string if not found
	PackageLocalName(path string) string

	// Return all local names of packages in alphabetical order
	PackageLocalNames() []string

	// Enumerates all packages.
	//
	// Returned map key is local name, value is path.
	Packages() map[string]string

	// Returns full qualified name by qualified name.
	//
	// Returns NullFullQName if QName.Pkg() is unknown.
	FullQName(QName) FullQName

	// Returns qualified name by full qualified name.
	//
	// Returns NullQName if FullQName.PkgPath() is unknown.
	LocalQName(FullQName) QName
}

type IPackagesBuilder interface {
	// Adds new package with specified local name and path.
	//
	// # Panics:
	//   - if local name is empty,
	//   - if local name is invalid,
	//   - if package with local name already exists,
	//   - if path is empty,
	//   - if package with path already exists.
	AddPackage(localName, path string) IPackagesBuilder
}
