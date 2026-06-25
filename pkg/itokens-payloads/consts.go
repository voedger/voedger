/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package payloads

import (
	"time"

	"github.com/voedger/voedger/pkg/istructs"
)

var (
	systemPrincipalPayload = PrincipalPayload{
		Login:       "system",
		SubjectKind: istructs.SubjectKind_User,
		ProfileWSID: istructs.NullWSID,
	}
)

const DefaultSystemPrincipalDuration = 10 * time.Minute
