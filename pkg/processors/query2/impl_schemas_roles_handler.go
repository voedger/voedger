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
	"github.com/voedger/voedger/pkg/istructsmem"
)

type schemasRolesHandler struct {
}

var _ IApiPathHandler = (*schemasRolesHandler)(nil) // ensure that queryHandler implements IApiPathHandler

func (h *schemasRolesHandler) IsArrayResult() bool {
	return false
}

func (h *schemasRolesHandler) CheckRateLimit(ctx context.Context, qw *queryWork) error {
	return nil
}

func (h *schemasRolesHandler) SetRequestType(ctx context.Context, qw *queryWork) error {
	return nil
}

func (h *schemasRolesHandler) SetResultType(ctx context.Context, qw *queryWork, statelessResources istructsmem.IStatelessResources) error {
	return nil
}

func (h *schemasRolesHandler) RequestOpKind() appdef.OperationKind {
	return appdef.OperationKind_Select
}

func (h *schemasRolesHandler) AuthorizeResult(ctx context.Context, qw *queryWork) (err error) {
	return nil
}

func (h *schemasRolesHandler) RowsProcessor(ctx context.Context, qw *queryWork) (err error) {
	return nil
}

func (h *schemasRolesHandler) Exec(ctx context.Context, qw *queryWork) (err error) {
	wsQname := qw.msg.WorkspaceQName()
	if wsQname == appdef.NullQName {
		return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Errorf("workspace is not specified"))
	}
	workspace := qw.appStructs.AppDef().Workspace(wsQname)
	if workspace == nil {
		return coreutils.NewHTTPErrorf(http.StatusNotFound, fmt.Errorf("workspace %s not found", wsQname.String()))
	}

	generatedHTML := fmt.Sprintf("<html><head><title>App %s: workspace %s published roles</title></head><body>", qw.msg.AppQName().String(), wsQname.String())
	generatedHTML += fmt.Sprintf("<h1>App %s</h1><h2>Workspace %s published roles</h2>", qw.msg.AppQName().String(), wsQname.String())
	roles := make([]appdef.IRole, 0)
	for _, typ := range workspace.Types() {
		if typ.Kind() == appdef.TypeKind_Role {
			role := typ.(appdef.IRole)
			if role.Published() {
				roles = append(roles, role)
				break
			}
		}
	}

	if len(roles) == 0 {
		generatedHTML += "<p>No published roles</p>"
	} else {
		generatedHTML += "<ul>"
		for _, role := range roles {
			ref := fmt.Sprintf("/api/v2/users/%s/apps/%s/schemas/%s/roles/%s", qw.msg.AppQName().Owner(), qw.msg.AppQName().Name(), workspace.QName().String(), role.QName().String())
			generatedHTML += fmt.Sprintf(`<li><a href="%s">%s</a></li>`, ref, role.QName().String())
		}
		generatedHTML += "</ul>"
	}
	generatedHTML += "</body></html>"

	return qw.msg.Responder().Respond(bus.ResponseMeta{ContentType: coreutils.ContentType_TextHTML, StatusCode: http.StatusOK}, generatedHTML)
}
