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
	"github.com/voedger/voedger/pkg/istructsmem"
)

type schemasHandler struct {
}

var _ IApiPathHandler = (*schemasHandler)(nil) // ensure that queryHandler implements IApiPathHandler

func (h *schemasHandler) IsArrayResult() bool {
	return false
}

func (h *schemasHandler) CheckRateLimit(ctx context.Context, qw *queryWork) error {
	return nil
}

func (h *schemasHandler) SetRequestType(ctx context.Context, qw *queryWork) error {
	return nil
}

func (h *schemasHandler) SetResultType(ctx context.Context, qw *queryWork, statelessResources istructsmem.IStatelessResources) error {
	return nil
}

func (h *schemasHandler) RequestOpKind() appdef.OperationKind {
	return appdef.OperationKind_Select
}

func (h *schemasHandler) AuthorizeResult(ctx context.Context, qw *queryWork) (err error) {
	return nil
}

func (h *schemasHandler) RowsProcessor(ctx context.Context, qw *queryWork) (err error) {
	// sender := &sender{responder: qw.msg.Responder(), isArrayResponse: false, contentType: contentTypeHtml}
	// qw.rowsProcessor = pipeline.NewAsyncPipeline(ctx, "View rows processor", pipeline.WireAsyncOperator("Sender", sender))
	// qw.responseWriterGetter = func() bus.IResponseWriter {
	// 	return sender.respWriter
	// }
	return nil
}

func (h *schemasHandler) Exec(ctx context.Context, qw *queryWork) (err error) {
	// wsQname := qw.msg.WorkspaceQName()
	// if wsQname == appdef.NullQName {
	// 	return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Errorf("workspace is not specified"))
	// }
	generatedHtml := "<html><head><title>Schema</title></head><body>"
	generatedHtml += "<h1>Schema</h1>"

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
		generatedHtml += "<p>No workspaces with published roles found</p>"
	} else {
		generatedHtml += "<ul>"
		for _, ws := range workspaces {
			ref := fmt.Sprintf("/api/v2/users/%s/apps/%s/schemas/%s/roles", qw.msg.AppQName().Owner(), qw.msg.AppQName().Name(), ws.QName().String())
			generatedHtml += fmt.Sprintf(`<li><a href="%s">"%s"</a></li>`, ref, ws.QName().String())
		}
		generatedHtml += "</ul>"
	}

	return qw.msg.Responder().Respond(bus.ResponseMeta{ContentType: contentTypeHtml, StatusCode: http.StatusOK}, generatedHtml)
}
