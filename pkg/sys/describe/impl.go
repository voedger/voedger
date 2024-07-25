/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package describe

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/voedger/voedger/pkg/istructs"
)

func qryDescribePackageNames(ctx context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
	as := args.State.AppStructs()
	names := as.DescribePackageNames()
	namesStr := strings.Join(names, ",")
	return callback(&result{res: namesStr})
}

func qryDescribePackage(ctx context.Context, args istructs.ExecQueryArgs, callback istructs.ExecQueryCallback) (err error) {
	as := args.State.AppStructs()

	packageName := args.ArgumentObject.AsString(field_PackageName)
	packageDescription := as.DescribePackage(packageName)

	b, err := json.Marshal(packageDescription)
	if err != nil {
		return err
	}

	return callback(&result{res: string(b)})
}

func (r *result) AsString(string) string {
	return r.res
}
