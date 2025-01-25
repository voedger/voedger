/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Michael Saigachenko
 */
package query2

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructsmem"
)

type viewHandler struct {
}

var _ IApiPathHandler = (*viewHandler)(nil) // ensure that viewHandler implements IApiPathHandler

func (h *viewHandler) CheckRateLimit(ctx context.Context, qw *queryWork) error {
	// TODO: implement rate limits check
	return nil
}

func (h *viewHandler) CheckType(ctx context.Context, qw *queryWork) error {
	switch qw.iWorkspace {
	case nil:
		// workspace is dummy
		if qw.iView = appdef.View(qw.appStructs.AppDef().Type, qw.msg.QName()); qw.iView == nil {
			return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Sprintf("view %s does not exist", qw.msg.QName()))
		}
	default:
		if qw.iView = appdef.View(qw.iWorkspace.Type, qw.msg.QName()); qw.iView == nil {
			return coreutils.NewHTTPErrorf(http.StatusBadRequest, fmt.Sprintf("view %s does not exist in %v", qw.msg.QName(), qw.iWorkspace))
		}
	}
	return nil
}

func (h *viewHandler) ResultType(ctx context.Context, qw *queryWork, statelessResources istructsmem.IStatelessResources) error {
	qw.resultType = qw.iView
	return nil
}

func (h *viewHandler) AuthorizeRequest(ctx context.Context, qw *queryWork) error {
	ws := qw.iWorkspace
	if ws == nil {
		// workspace is dummy
		ws = qw.iView.Workspace()
	}
	ok, err := qw.appPart.IsOperationAllowed(ws, appdef.OperationKind_Select, qw.msg.QName(), nil, qw.roles)
	if err != nil {
		return err
	}
	if !ok {
		return coreutils.WrapSysError(errors.New(""), http.StatusForbidden)
	}
	return nil
}

func (h *viewHandler) AuthorizeResult(ctx context.Context, qw *queryWork) error {
	// TODO: implement result authorization
	return nil
}

func (h *viewHandler) RowsProcessor(ctx context.Context, qw *queryWork) error {
	// TODO: implement rows processor
	return nil
}

func (h *viewHandler) Exec(ctx context.Context, qw *queryWork) error {
	// TODO: implement exec
	return nil
}
