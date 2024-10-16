/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

func syncProjectorJoinedWorkspaceIndex() istructs.Projector {
	return istructs.Projector{
		Name: QNameProjectorJoinedWorkspaceIndex,
		Func: joinedWorkspaceIndexProjector,
	}
}

var joinedWorkspaceIndexProjector = func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
	for rec := range event.CUDs {
		if rec.QName() != QNameCDocJoinedWorkspace {
			continue
		}

		skbViewJoinedWorkspaceIndex, err := s.KeyBuilder(sys.Storage_View, QNameViewJoinedWorkspaceIndex)
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
	}
	return nil
}
