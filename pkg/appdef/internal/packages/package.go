/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package packages

import (
	"slices"

	"github.com/voedger/voedger/pkg/appdef"
)

// # Supports:
//   - appdef.IWithPackages
type WithPackages struct {
	local       []string
	localByPath map[string]string
	pathByLocal map[string]string
}

func MakeWithPackages() WithPackages {
	return WithPackages{
		local:       make([]string, 0),
		localByPath: make(map[string]string),
		pathByLocal: make(map[string]string),
	}
}

func (p WithPackages) FullQName(n appdef.QName) appdef.FullQName {
	if path, ok := p.pathByLocal[n.Pkg()]; ok {
		return appdef.NewFullQName(path, n.Entity())
	}
	return appdef.NullFullQName
}

func (p WithPackages) LocalQName(n appdef.FullQName) appdef.QName {
	if pkg, ok := p.localByPath[n.PkgPath()]; ok {
		return appdef.NewQName(pkg, n.Entity())
	}
	return appdef.NullQName
}

func (p WithPackages) PackageFullPath(localName string) string { return p.pathByLocal[localName] }

func (p WithPackages) PackageLocalName(path string) string { return p.localByPath[path] }

func (p WithPackages) PackageLocalNames() []string { return p.local }

func (p WithPackages) Packages() map[string]string { return p.pathByLocal }

func (p *WithPackages) add(local, path string) {
	if ok, err := appdef.ValidIdent(local); !ok {
		panic(err)
	}
	if p, ok := p.pathByLocal[local]; ok {
		panic(appdef.ErrAlreadyExists("package local name «%s» already used for «%s»", local, p))
	}

	if path == "" {
		panic(appdef.ErrMissed("package «%s» path", local))
	}
	if l, ok := p.localByPath[path]; ok {
		panic(appdef.ErrAlreadyExists("package path «%s» already used for «%s»", path, l))
	}

	p.local = append(p.local, local)
	slices.Sort(p.local)

	p.localByPath[path] = local
	p.pathByLocal[local] = path
}

type PackagesBuilder struct {
	p *WithPackages
}

func MakePackagesBuilder(p *WithPackages) PackagesBuilder {
	return PackagesBuilder{p}
}

func (pb *PackagesBuilder) AddPackage(localName, path string) appdef.IPackagesBuilder {
	pb.p.add(localName, path)
	return pb
}

func AddPackage(p *WithPackages, localName, path string) {
	p.add(localName, path)
}
