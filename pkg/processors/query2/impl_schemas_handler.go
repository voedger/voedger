/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */
package query2

import (
	"cmp"
	"context"
	"fmt"
	"html"
	"net/http"
	"slices"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/goutils/httpu"
)

func schemasHandler() apiPathHandler {
	return apiPathHandler{
		exec: schemasExec,
	}
}

func schemasExec(_ context.Context, qw *queryWork) (err error) {
	var sb strings.Builder
	appStr := qw.msg.AppQName().String()
	fmt.Fprintf(&sb, "<html><head><title>App %s schema</title></head><body>", appStr)
	fmt.Fprintf(&sb, "<h1>App %s schema</h1>", appStr)

	packages := make(map[string][]appdef.IWorkspace)
	developer := qw.isDeveloper()

	for _, ws := range qw.appStructs.AppDef().Workspaces() {
		hasPublishedRoles := false
		for _, typ := range ws.Types() {
			if typ.Kind() == appdef.TypeKind_Role {
				role := typ.(appdef.IRole)
				if role.Published() {
					hasPublishedRoles = true
					break
				}
			}
		}
		if hasPublishedRoles || developer {
			workspaces := packages[ws.QName().Pkg()]
			if workspaces == nil {
				workspaces = make([]appdef.IWorkspace, 0)
			}
			workspaces = append(workspaces, ws)
			packages[ws.QName().Pkg()] = workspaces
		}
	}

	if len(packages) == 0 {
		if !developer {
			sb.WriteString("<p>No workspaces with published roles found</p>")
		} else {
			sb.WriteString("<p>No workspaces found</p>")
		}
	} else {
		// Sort packages alphabetically
		pkgNames := make([]string, 0, len(packages))
		for pkg := range packages {
			pkgNames = append(pkgNames, pkg)
		}
		slices.Sort(pkgNames)

		for _, pkg := range pkgNames {
			fmt.Fprintf(&sb, "<h2>Package %s</h2>", pkg)
			sb.WriteString("<ul>")

			// Sort workspaces by QName for consistent ordering
			workspaces := packages[pkg]
			slices.SortFunc(workspaces, func(a, b appdef.IWorkspace) int {
				return cmp.Compare(a.QName().String(), b.QName().String())
			})

			appOwnerStr := qw.msg.AppQName().Owner()
			appNameStr := qw.msg.AppQName().Name()
			for _, ws := range workspaces {
				ref := fmt.Sprintf("/api/v2/apps/%s/%s/schemas/%s/roles", appOwnerStr, appNameStr, ws.QName().String())
				fmt.Fprintf(&sb, `<li><a href=%q>%s</a></li>`, html.EscapeString(ref), html.EscapeString(ws.QName().String()))
			}
			sb.WriteString("</ul>")
		}
	}
	sb.WriteString("</body></html>")

	return qw.msg.Responder().Respond(bus.ResponseMeta{ContentType: httpu.ContentType_TextHTML, StatusCode: http.StatusOK}, sb.String())
}
