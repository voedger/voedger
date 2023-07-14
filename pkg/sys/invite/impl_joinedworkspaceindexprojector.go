/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
)

func provideSyncProjectorJoinedWorkspaceIndexFactory() istructs.ProjectorFactory {
	return func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name:         QNameProjectorJoinedWorkspaceIndex,
			EventsFilter: []appdef.QName{qNameCmdCreateJoinedWorkspace},
			Func:         joinedWorkspaceIndexProjector,
		}
	}
}

var joinedWorkspaceIndexProjector = func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
	return event.CUDs(func(rec istructs.ICUDRow) (err error) {
		if rec.QName() != QNameCDocJoinedWorkspace {
			return
		}

		skbViewJoinedWorkspaceIndex, err := s.KeyBuilder(state.ViewRecordsStorage, QNameViewJoinedWorkspaceIndex)
		if err != nil {
			return err
		}
		skbViewJoinedWorkspaceIndex.PutInt32(field_Dummy, value_Dummy_Two)
		skbViewJoinedWorkspaceIndex.PutInt64(Field_InvitingWorkspaceWSID, rec.AsInt64(Field_InvitingWorkspaceWSID))

		svbViewJoinedWorkspaceIndex, err := intents.NewValue(skbViewJoinedWorkspaceIndex)
		if err != nil {
			return err
		}

		svbViewJoinedWorkspaceIndex.PutRecordID(field_JoinedWorkspaceID, rec.ID())

		return
	})
}
