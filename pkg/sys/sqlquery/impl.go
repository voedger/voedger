/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package sqlquery

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/blastrain/vitess-sqlparser/sqlparser"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func execQrySqlQuery(asp istructs.IAppStructsProvider, appQName istructs.AppQName, numCommandProcessors coreutils.CommandProcessorsCount) func(ctx context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
	return func(ctx context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
		wsid := args.WSID
		if index := strings.Index(args.ArgumentObject.AsString(field_Query), flag_WSID); index != -1 {
			v, err := strconv.Atoi(args.ArgumentObject.AsString(field_Query)[index+len(flag_WSID):])
			if err != nil {
				return err
			}
			wsid = istructs.WSID(v)
		}

		stmt, err := sqlparser.Parse(args.ArgumentObject.AsString(field_Query))
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

		appStructs, err := asp.AppStructs(appQName)
		if err != nil {
			return err
		}

		var whereExpr sqlparser.Expr
		if s.Where == nil {
			whereExpr = nil
		} else {
			whereExpr = s.Where.Expr
		}

		table := s.From[0].(*sqlparser.AliasedTableExpr).Expr.(sqlparser.TableName)
		source := appdef.NewQName(table.Qualifier.String(), table.Name.String())

		switch appStructs.AppDef().Type(source).Kind() {
		case appdef.TypeKind_ViewRecord:
			return readViewRecords(ctx, wsid, appdef.NewQName(table.Qualifier.String(), table.Name.String()), whereExpr, appStructs, f, callback)
		case appdef.TypeKind_CDoc:
			fallthrough
		case appdef.TypeKind_CRecord:
			fallthrough
		case appdef.TypeKind_WDoc:
			return readRecords(wsid, source, whereExpr, appStructs, f, callback)
		default:
			if source != plog && source != wlog {
				break
			}
			limit, offset, e := params(whereExpr, s.Limit)
			if e != nil {
				return e
			}
			if source == plog {
				return readPlog(ctx, wsid, numCommandProcessors, offset, limit, appStructs, f, callback)
			}
			return readWlog(ctx, wsid, offset, limit, appStructs, f, callback)
		}

		return fmt.Errorf("unsupported source: %s", source)
	}
}

func params(expr sqlparser.Expr, limit *sqlparser.Limit) (int, istructs.Offset, error) {
	l, err := lim(limit)
	if err != nil {
		return 0, 0, err
	}
	o, eq, err := offs(expr)
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
		return 0, fmt.Errorf("limit must be greater than -2")
	}
	if v == -1 {
		return istructs.ReadToTheEnd, nil
	}
	return int(v), err
}

func offs(expr sqlparser.Expr) (istructs.Offset, bool, error) {
	o := DefaultOffset
	eq := false
	switch r := expr.(type) {
	case *sqlparser.ComparisonExpr:
		if r.Left.(*sqlparser.ColName).Name.String() != "offset" {
			return 0, false, fmt.Errorf("unsupported column name: %s", r.Left.(*sqlparser.ColName).Name.String())
		}
		v, e := parseInt64(r.Right.(*sqlparser.SQLVal).Val)
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
			return 0, false, fmt.Errorf("offset must be greater than zero")
		}
	case nil:
	default:
		return 0, false, fmt.Errorf("unsupported expression: %T", r)
	}
	return o, eq, nil
}

func parseInt64(bb []byte) (int64, error) {
	return strconv.ParseInt(string(bb), base, bitSize64)
}

func getFilter(f func(string) bool) coreutils.MapperOpt {
	return coreutils.Filter(func(name string, kind appdef.DataKind) bool {
		return f(name)
	})
}
