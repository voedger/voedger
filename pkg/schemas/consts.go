/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package schemas

import "github.com/voedger/voedger/pkg/istructs"

// MaxIdentLen is maximum identificator length
const MaxIdentLen = 255

// NullSchema is used for return then schema	is not founded
var NullSchema = newSchema(nil, istructs.NullQName, istructs.SchemaKind_null)
