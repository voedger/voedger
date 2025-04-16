/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */
package query2

import (
	"context"
	"fmt"
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
)

func schemasHandler() apiPathHandler {
	return apiPathHandler{
		exec: schemasExec,
	}
}

func schemasExec(ctx context.Context, qw *queryWork) (err error) {
	generatedHTML := fmt.Sprintf("<html><head><title>App %s schema</title></head><body>", qw.msg.AppQName().String())
	generatedHTML += fmt.Sprintf("<h1>App %s schema</h1>", qw.msg.AppQName().String())

	workspaces := make([]appdef.IWorkspace, 0)

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
		if hasPublishedRoles {
			workspaces = append(workspaces, ws)
		}
	}

	if len(workspaces) == 0 {
		generatedHTML += "<p>No workspaces with published roles found</p>"
	} else {
		generatedHTML += "<ul>"
		for _, ws := range workspaces {
			ref := fmt.Sprintf("/api/v2/apps/%s/%s/schemas/%s/roles", qw.msg.AppQName().Owner(), qw.msg.AppQName().Name(), ws.QName().String())
			generatedHTML += fmt.Sprintf(`<li><a href="%s">%s</a></li>`, ref, ws.QName().String())
		}
		generatedHTML += "</ul>"
	}
	generatedHTML += "</body></html>"

	return qw.msg.Responder().Respond(bus.ResponseMeta{ContentType: coreutils.ContentType_TextHTML, StatusCode: http.StatusOK}, generatedHTML)
}
