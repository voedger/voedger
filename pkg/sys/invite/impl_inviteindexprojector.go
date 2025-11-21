/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/sys"
)

func syncProjectorInviteIndex() istructs.Projector {
	return istructs.Projector{
		Name: qNameProjectorInviteIndex,
		Func: inviteIndexProjector,
	}
}

var inviteIndexProjector = func(event istructs.IPLogEvent, s istructs.IState, intents istructs.IIntents) (err error) {
	for rec := range event.CUDs {
		if rec.QName() != QNameCDocInvite {
			continue
		}

		skbViewInviteIndex, err := s.KeyBuilder(sys.Storage_View, qNameViewInviteIndex)
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
	}
	return nil
}
