/*
 * Copyright (c) 2024-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package cluster

import (
	"log"
	"testing"

	"github.com/voedger/voedger/pkg/istructs"
)

func TestMain(t *testing.T) {
	appWSID := istructs.NewWSID(istructs.MainClusterID, istructs.WSID(1+int(istructs.FirstBaseAppWSID)))
	log.Println(appWSID.BaseWSID() - istructs.FirstBaseAppWSID)
}
