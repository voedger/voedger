/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
)

func ProvideSyncProjectorInviteIndexFactory() istructs.ProjectorFactory {
	return func(partition istructs.PartitionID) istructs.Projector {
		return istructs.Projector{
			Name:         qNameViewInviteIndex,
			EventsFilter: []appdef.QName{qNameCmdInitiateInvitationByEMail},
			Func:         inviteIndexProjector,
		}
	}
}

var inviteIndexProjector = func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
	return event.CUDs(func(rec istructs.ICUDRow) (err error) {
		if rec.QName() != qNameCDocInvite {
			return
		}

		skbViewInviteIndex, err := s.KeyBuilder(state.ViewRecordsStorage, qNameViewInviteIndex)
		if err != nil {
			return err
		}
		skbViewInviteIndex.PutInt32(field_Dummy, value_Dummy_One)
		skbViewInviteIndex.PutString(Field_Login, event.ArgumentObject().AsString(field_Email))

		svViewInviteIndex, err := intents.NewValue(skbViewInviteIndex)
		if err != nil {
			return err
		}

		svViewInviteIndex.PutRecordID(field_InviteID, rec.ID())

		return
	})
}
