/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/untillpro/goutils/iterate"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func applyViewSubjectsIdx(partition istructs.PartitionID) istructs.Projector {
	return istructs.Projector{
		Name: QNameApplyViewSubjectsIdx,
		Func: viewSubjectsIdxProjector,
	}
}

func viewSubjectsIdxProjector(event istructs.IPLogEvent, st istructs.IState, intents istructs.IIntents) (err error) {
	return iterate.ForEachError(event.CUDs, func(cdocSubject istructs.ICUDRow) error {
		if cdocSubject.QName() != QNameCDocSubject || !cdocSubject.IsNew() {
			return nil
		}

		skbViewSubjectsIdx, err := st.KeyBuilder(state.View, QNameViewSubjectsIdx)
		if err != nil {
			// notest
			return err
		}
		login := cdocSubject.AsString(Field_Login)
		skbViewSubjectsIdx.PutInt64(Field_LoginHash, coreutils.HashBytes([]byte(login)))
		skbViewSubjectsIdx.PutString(Field_Login, login)
		_, ok, err := st.CanExist(skbViewSubjectsIdx)
		if err != nil {
			// notest
			return err
		}
		if ok {
			// already handled by the projector in async mode
			return nil
		}
		subjectsIdxBuilder, err := intents.NewValue(skbViewSubjectsIdx)
		if err != nil {
			// notest
			return err
		}
		subjectsIdxBuilder.PutRecordID(Field_SubjectID, cdocSubject.ID())
		return nil
	})
}
