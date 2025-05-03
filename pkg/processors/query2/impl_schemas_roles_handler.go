/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */
package query2

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
)

func schemasRolesHandler() apiPathHandler {
	return apiPathHandler{
		exec: schemasRolesExec,
	}
}

func schemasRolesExec(ctx context.Context, qw *queryWork) (err error) {
	wsQname := qw.msg.WorkspaceQName()
	if wsQname == appdef.NullQName {
		return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Errorf("workspace is not specified"))
	}
	workspace := qw.appStructs.AppDef().Workspace(wsQname)
	if workspace == nil {
		return coreutils.NewHTTPErrorf(http.StatusNotFound, fmt.Errorf("workspace %s not found", wsQname.String()))
	}

	developer := qw.isDeveloper()
	rolesTitle := "roles"
	if !developer {
		rolesTitle = "published roles"
	}

	generatedHTML := fmt.Sprintf("<html><head><title>App %s: workspace %s %s</title></head><body>", qw.msg.AppQName().String(), wsQname.String(), rolesTitle)
	generatedHTML += fmt.Sprintf("<h1>App %s</h1><h2>Workspace %s %s</h2>", qw.msg.AppQName().String(), wsQname.String(), rolesTitle)
	packages := make(map[string][]appdef.IRole)
	for _, typ := range workspace.Types() {
		if typ.Kind() == appdef.TypeKind_Role {
			role := typ.(appdef.IRole)
			if role.Published() || developer {
				pkgName := role.QName().Pkg()
				roles := packages[pkgName]
				if roles == nil {
					roles = make([]appdef.IRole, 0)
				}
				roles = append(roles, role)
				packages[pkgName] = roles
			}
		}
	}

	if len(packages) == 0 {
		generatedHTML += fmt.Sprintf("<p>No %s</p>", rolesTitle)
	} else {
		// Sort packages alphabetically
		pkgNames := make([]string, 0, len(packages))
		for pkg := range packages {
			pkgNames = append(pkgNames, pkg)
		}
		sort.Strings(pkgNames)

		for _, pkg := range pkgNames {
			roles := packages[pkg]
			// Sort roles alphabetically
			sort.Slice(roles, func(i, j int) bool {
				return roles[i].QName().String() < roles[j].QName().String()
			})

			generatedHTML += fmt.Sprintf("<h2>Package %s</h2>", pkg)
			generatedHTML += "<ul>"
			for _, role := range roles {
				ref := fmt.Sprintf("/api/v2/apps/%s/%s/schemas/%s/roles/%s", qw.msg.AppQName().Owner(), qw.msg.AppQName().Name(), workspace.QName().String(), role.QName().String())
				generatedHTML += fmt.Sprintf(`<li><a href="%s">%s</a></li>`, ref, role.QName().String())
			}
			generatedHTML += "</ul>"
		}
	}
	generatedHTML += "</body></html>"

	return qw.msg.Responder().Respond(bus.ResponseMeta{ContentType: coreutils.ContentType_TextHTML, StatusCode: http.StatusOK}, generatedHTML)
}
