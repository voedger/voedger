/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */
package query2

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appdef/acl"
	"github.com/voedger/voedger/pkg/bus"
	"github.com/voedger/voedger/pkg/coreutils"
)

func swaggerUI(url string) string {
	return fmt.Sprintf(swaggerUI_HTML, url)
}

// [~server.apiv2.role/cmp.schemasRoleHandler~impl]
func schemasRoleHandler() apiPathHandler {
	return apiPathHandler{
		requestOpKind: appdef.OperationKind_Select,
		exec:          schemasRoleExec,
	}
}

func schemasRoleExec(ctx context.Context, qw *queryWork) (err error) {

	wsQname := qw.msg.WorkspaceQName()
	if wsQname == appdef.NullQName {
		return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Errorf("workspace is not specified"))
	}
	workspace := qw.appStructs.AppDef().Workspace(wsQname)
	if workspace == nil {
		return coreutils.NewHTTPErrorf(http.StatusNotFound, fmt.Errorf("workspace %s not found", wsQname.String()))
	}

	role := workspace.Type(qw.msg.QName())
	if role == nil || role.Kind() != appdef.TypeKind_Role {
		return coreutils.NewHTTPErrorf(http.StatusNotFound, fmt.Errorf("role %s not found in workspace %s", qw.msg.QName().String(), wsQname.String()))
	}

	schemaMeta := SchemaMeta{
		SchemaTitle:   fmt.Sprintf("%s: %s OpenAPI 3.0", qw.msg.AppQName().Name(), role.QName().Entity()),
		SchemaVersion: "1.0.0", // TODO: get app name and version from appdef
		Description:   role.Comment(),
		AppName:       qw.msg.AppQName(),
	}

	writer := new(bytes.Buffer)

	err = CreateOpenAPISchema(writer, workspace, qw.msg.QName(), acl.PublishedTypes, schemaMeta)
	if err != nil {
		return coreutils.NewHTTPErrorf(http.StatusInternalServerError, err)
	}

	if strings.Contains(qw.msg.Accept(), coreutils.ContentType_TextHTML) {
		url := fmt.Sprintf(`/api/v2/users/%s/apps/%s/schemas/%s/roles/%s`,
			qw.msg.AppQName().Owner(), qw.msg.AppQName().Name(), wsQname.String(), qw.msg.QName().String())
		return qw.msg.Responder().Respond(bus.ResponseMeta{ContentType: coreutils.ContentType_TextHTML, StatusCode: http.StatusOK}, swaggerUI(url))
	}
	return qw.msg.Responder().Respond(bus.ResponseMeta{ContentType: coreutils.ContentType_ApplicationJSON, StatusCode: http.StatusOK}, writer.Bytes())
}
