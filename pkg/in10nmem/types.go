/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * Aleksei Ponomarev
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package in10nmem

import (
	"github.com/voedger/voedger/pkg/in10n"
	istructs "github.com/voedger/voedger/pkg/istructs"
)

type UpdateUnit struct {
	Projection in10n.ProjectionKey
	Offset     istructs.Offset
}

type CreateChannelParamsType struct {
	SubjectLogin  istructs.SubjectLogin
	ProjectionKey []in10n.ProjectionKey
}
