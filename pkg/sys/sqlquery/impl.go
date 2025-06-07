/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sqlquery

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strconv"

	"github.com/blastrain/vitess-sqlparser/sqlparser"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/coreutils/federation"
	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/dml"
	"github.com/voedger/voedger/pkg/goutils/logger"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/itokens"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"github.com/voedger/voedger/pkg/sys"
	"github.com/voedger/voedger/pkg/sys/authnz"
)

func provideExecQrySQLQuery(federation federation.IFederation, itokens itokens.ITokens) func(ctx context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
	return func(ctx context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {

		query := args.ArgumentObject.AsString(field_Query)

		op, err := dml.ParseQuery(query)
		if err != nil {
			return coreutils.NewHTTPError(http.StatusBadRequest, err)
		}

		if op.Kind != dml.OpKind_Select {
			return coreutils.NewHTTPErrorf(http.StatusBadRequest, "'select' operation is expected")
		}
		appStructs := args.State.AppStructs()
		var wsID istructs.WSID
		switch op.Workspace.Kind {
		case dml.WorkspaceKind_AppWSNum:
			wsID = istructs.NewWSID(istructs.CurrentClusterID(), istructs.FirstBaseAppWSID+istructs.WSID(op.Workspace.ID))
		case dml.WorkspaceKind_WSID:
			wsID = istructs.WSID(op.Workspace.ID)
		case dml.WorkspaceKind_PseudoWSID:
			wsID = coreutils.GetAppWSID(istructs.WSID(op.Workspace.ID), appStructs.NumAppWorkspaces())
		default:
			wsID = args.WSID
		}
		if (op.AppQName != appdef.NullAppQName && op.AppQName != args.State.App()) || (wsID != args.WSID) {
			targetWSID := wsID
			if op.Workspace.Kind == dml.WorkspaceKind_PseudoWSID {
				targetWSID = istructs.WSID(op.Workspace.ID)
			}
			targetAppQName := op.AppQName
			if targetAppQName == appdef.NullAppQName {
				targetAppQName = args.State.App()
			}
			subjKB, err := args.State.KeyBuilder(sys.Storage_RequestSubject, appdef.NullQName)
			if err != nil {
				//notest
				return err
			}
			subj, err := args.State.MustExist(subjKB)
			if err != nil {
				// notest
				return err
			}

			tokenForTargetApp := subj.AsString(sys.Storage_RequestSubject_Field_Token)
			if targetAppQName != args.State.App() {
				// query is for a foreign app -> re-issue token for the target app
				var pp payloads.PrincipalPayload
				if _, err = itokens.ValidateToken(tokenForTargetApp, &pp); err != nil {
					// notest: validated already by the processor
					return err
				}
				if tokenForTargetApp, err = itokens.IssueToken(targetAppQName, authnz.DefaultPrincipalTokenExpiration, &pp); err != nil {
					return err
				}
			}
			logger.Info(fmt.Sprintf("forwarding query to %s/%d", targetAppQName, targetWSID))
			body := fmt.Sprintf(`{"args":{"Query":%q},"elements":[{"fields":["Result"]}]}`, op.VSQLWithoutAppAndWSID)
			resp, err := federation.Func(fmt.Sprintf("api/%s/%d/q.sys.SqlQuery", targetAppQName, targetWSID),
				body, coreutils.WithAuthorizeBy(tokenForTargetApp))
			if err != nil {
				return err
			}
			for i := 0; i < resp.NumRows(); i++ {
				if err := callback(&result{value: resp.SectionRow(i)[0].(string)}); err != nil {
					// notest
					return err
				}
			}
			return nil
		}
		stmt, err := sqlparser.Parse(op.CleanSQL)
		if err != nil {
			return err
		}
		s := stmt.(*sqlparser.Select)

		f := &filter{fields: make(map[string]bool)}
		for _, intf := range s.SelectExprs {
			switch expr := intf.(type) {
			case *sqlparser.StarExpr:
				f.acceptAll = true
			case *sqlparser.AliasedExpr:
				column := expr.Expr.(*sqlparser.ColName)
				if !column.Qualifier.Name.IsEmpty() {
					f.fields[fmt.Sprintf("%s.%s", column.Qualifier.Name, column.Name)] = true
				} else {
					f.fields[column.Name.String()] = true
				}
			}
		}

		var whereExpr sqlparser.Expr
		if s.Where == nil {
			whereExpr = nil
		} else {
			whereExpr = s.Where.Expr
		}

		table := s.From[0].(*sqlparser.AliasedTableExpr).Expr.(sqlparser.TableName)
		source := appdef.NewQName(table.Qualifier.String(), table.Name.String())
		if source.Entity() == "blob" {
			// FIXME: eliminate this hack
			// sys.BLOB translates to sys.blob by vitess-sqlparser
			// https://github.com/voedger/voedger/issues/3708
			source = appdef.NewQName(appdef.SysPackage, "BLOB")
		}

		kind := appStructs.AppDef().Type(source).Kind()
		if _, ok := appStructs.AppDef().Type(source).(appdef.IStructure); ok {
			// is a structure -> check ACL
			switch kind {
			case appdef.TypeKind_ViewRecord, appdef.TypeKind_CDoc, appdef.TypeKind_CRecord, appdef.TypeKind_WDoc:
				fields := make([]string, 0, len(f.fields))
				for f := range f.fields {
					fields = append(fields, f)
				}
				apppart := args.Workpiece.(interface{ AppPartition() appparts.IAppPartition }).AppPartition()
				roles := args.Workpiece.(interface{ Roles() []appdef.QName }).Roles()
				ok, err := apppart.IsOperationAllowed(args.Workspace, appdef.OperationKind_Select, source, fields, roles)
				if err != nil {
					// notest
					if errors.Is(err, appdef.ErrNotFoundError) {
						return coreutils.WrapSysError(err, http.StatusBadRequest)
					}
					return err
				}
				if !ok {
					return coreutils.NewHTTPErrorf(http.StatusForbidden)
				}
			}
		}
		switch kind {
		case appdef.TypeKind_ViewRecord:
			if op.EntityID > 0 {
				return errors.New("ID must not be specified on select from view")
			}
			return readViewRecords(ctx, wsID, appdef.NewQName(table.Qualifier.String(), table.Name.String()), whereExpr, appStructs, f, callback)
		case appdef.TypeKind_CDoc, appdef.TypeKind_CRecord, appdef.TypeKind_WDoc, appdef.TypeKind_ODoc, appdef.TypeKind_ORecord:
			return coreutils.WrapSysError(readRecords(wsID, source, whereExpr, appStructs, f, callback, istructs.RecordID(op.EntityID)),
				http.StatusBadRequest)
		default:
			if source != plog && source != wlog {
				break
			}
			limit, offset, e := params(whereExpr, s.Limit, istructs.Offset(op.EntityID))
			if e != nil {
				return e
			}
			appParts := args.Workpiece.(interface {
				AppPartitions() appparts.IAppPartitions
			}).AppPartitions()
			if source == plog {
				return readPlog(ctx, wsID, offset, limit, appStructs, f, callback, appStructs.AppDef(), appParts)
			}
			return readWlog(ctx, wsID, offset, limit, appStructs, f, callback, appStructs.AppDef())
		}

		return fmt.Errorf("do not know how to read from the requested %s, %s", source, kind)
	}
}

func params(expr sqlparser.Expr, limit *sqlparser.Limit, simpleOffset istructs.Offset) (int, istructs.Offset, error) {
	l, err := lim(limit)
	if err != nil {
		return 0, 0, err
	}
	o, eq, err := offs(expr, simpleOffset)
	if err != nil {
		return 0, 0, err
	}
	if eq && l != 0 {
		l = 1
	}
	return l, o, nil
}

func lim(limit *sqlparser.Limit) (int, error) {
	if limit == nil {
		return DefaultLimit, nil
	}
	v, err := parseInt64(limit.Rowcount.(*sqlparser.SQLVal).Val)
	if err != nil {
		return 0, err
	}
	if v < -1 {
		return 0, errors.New("limit must be greater than -2")
	}
	if v == -1 {
		return istructs.ReadToTheEnd, nil
	}
	return int(v), err
}

func offs(expr sqlparser.Expr, simpleOffset istructs.Offset) (istructs.Offset, bool, error) {
	o := istructs.FirstOffset
	eq := false
	switch r := expr.(type) {
	case *sqlparser.ComparisonExpr:
		if r.Left.(*sqlparser.ColName).Name.String() != "offset" {
			return 0, false, fmt.Errorf("unsupported column name: %s", r.Left.(*sqlparser.ColName).Name.String())
		}
		if simpleOffset > 0 {
			return 0, false, errors.New("both .Offset and 'where offset ...' clause can not be provided in one query")
		}
		v, e := parseUint64(r.Right.(*sqlparser.SQLVal).Val)
		if e != nil {
			return 0, false, e
		}
		switch r.Operator {
		case sqlparser.EqualStr:
			eq = true
			fallthrough
		case sqlparser.GreaterEqualStr:
			o = istructs.Offset(v)
		case sqlparser.GreaterThanStr:
			o = istructs.Offset(v + 1)
		default:
			return 0, false, fmt.Errorf("unsupported operation: %s", r.Operator)
		}
		if o <= 0 {
			return 0, false, errors.New("offset must be greater than zero")
		}
	case nil:
		if simpleOffset != istructs.NullOffset {
			o = simpleOffset
		}
	default:
		return 0, false, fmt.Errorf("unsupported expression: %T", r)
	}
	return o, eq, nil
}

func parseInt64(bb []byte) (int64, error) {
	return strconv.ParseInt(string(bb), utils.DecimalBase, utils.BitSize64)
}

func parseUint64(bb []byte) (uint64, error) {
	return strconv.ParseUint(string(bb), utils.DecimalBase, utils.BitSize64)
}

func getFilter(f func(string) bool) coreutils.MapperOpt {
	return coreutils.Filter(func(name string, kind appdef.DataKind) bool {
		return f(name)
	})
}

func renderDBEvent(data map[string]interface{}, f *filter, event istructs.IDbEvent, appDef appdef.IAppDef, offset istructs.Offset) {
	defer func() {
		if r := recover(); r != nil {
			eventKind := "plog"
			if _, ok := event.(istructs.IWLogEvent); ok {
				eventKind = "wlog"
			}
			stackTrace := string(debug.Stack())
			errMes := fmt.Sprintf("failed to render %s event %s offset %d registered at %s: %v\n%s", eventKind, event.QName(), offset, event.RegisteredAt().String(), r, stackTrace)
			logger.Error(errMes)
			data["!!!Panic"] = errMes
		}
	}()
	if f.filter("QName") {
		data["QName"] = event.QName().String()
	}
	if f.filter("ArgumentObject") {
		data["ArgumentObject"] = coreutils.ObjectToMap(event.ArgumentObject(), appDef)
	}
	if f.filter("CUDs") {
		data["CUDs"] = coreutils.CUDsToMap(event, appDef)
	}
	if f.filter("RegisteredAt") {
		data["RegisteredAt"] = event.RegisteredAt()
	}
	if f.filter("Synced") {
		data["Synced"] = event.Synced()
	}
	if f.filter("DeviceID") {
		data["DeviceID"] = event.DeviceID()
	}
	if f.filter("SyncedAt") {
		data["SyncedAt"] = event.SyncedAt()
	}
	if f.filter("Error") && event.Error() != nil {
		errorData := make(map[string]interface{})
		errorData["ErrStr"] = event.Error().ErrStr()
		errorData["QNameFromParams"] = event.Error().QNameFromParams().String()
		errorData["ValidEvent"] = event.Error().ValidEvent()
		errorData["OriginalEventBytes"] = event.Error().OriginalEventBytes()
		data["Error"] = errorData
	}
}
