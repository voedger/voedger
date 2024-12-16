/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/istructs"
)

func applyViewSubjectsIdx() istructs.Projector {
	return istructs.Projector{
		Name: QNameApplyViewSubjectsIdx,
		Func: applyViewSubjectsIdxProjector,
	}
}

func applyViewSubjectsIdxProjector(event istructs.IPLogEvent, st istructs.IState, intents istructs.IIntents) (err error) {
	for cdocSubject := range event.CUDs {
		if cdocSubject.QName() != QNameCDocSubject || !cdocSubject.IsNew() {
			continue
		}

		actualLogin := cdocSubject.AsString(Field_Login) // cdoc.sys.Subject.Login <- cdoc.sys.Invite.ActualLogin by ap.sys.ApplyJoinWorkspace
		skbViewSubjectsIdx, err := GetSubjectIdxViewKeyBuilder(actualLogin, st)
		if err != nil {
			// notest
			return err
		}

		// ap.sys.ApplyJoinWorkspace will not insert cdoc.sys.Subject if view.sys.SubjectsIdx record exists already by the login
		// according to https://github.com/voedger/voedger/issues/1107
		// so no overwrite here
		subjectsIdxBuilder, err := intents.NewValue(skbViewSubjectsIdx)
		if err != nil {
			// notest
			return err
		}
		subjectsIdxBuilder.PutRecordID(Field_SubjectID, cdocSubject.ID())
	}
	return nil
}
