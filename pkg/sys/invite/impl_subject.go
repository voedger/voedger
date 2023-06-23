/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 */

package invite

import (
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem"
	"github.com/voedger/voedger/pkg/projectors"
	"github.com/voedger/voedger/pkg/state"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

// buildSubjectsIdx need to build view.sys.SubjectIdx on an existing storage: true -> async projector will be registered, sync otherwise
func provideCDocSubject(cfg *istructsmem.AppConfigType, appDefBuilder appdef.IAppDefBuilder, buildSubjectsIdx bool) {
	doc := appDefBuilder.AddCDoc(QNameCDocSubject)
	doc.
		AddField(Field_Login, appdef.DataKind_string, true).
		AddField(Field_SubjectKind, appdef.DataKind_int32, true).
		AddField(Field_Roles, appdef.DataKind_string, true).
		AddField(Field_ProfileWSID, appdef.DataKind_int64, true)
	doc.SetUniqueField(Field_Login)

	projectors.ProvideViewDef(appDefBuilder, QNameViewSubjectsIdx, func(b appdef.IViewBuilder) {
		b.
			AddPartField(Field_LoginHash, appdef.DataKind_int64).
			AddClustColumn(Field_Login, appdef.DataKind_string).
			AddValueField(Field_SubjectID, appdef.DataKind_RecordID, true)
	})

	if buildSubjectsIdx {
		cfg.AddAsyncProjectors(subjectIdxProjectorFactory)
	} else {
		cfg.AddSyncProjectors(subjectIdxProjectorFactory)
	}
}

func subjectIdxProjectorFactory(partition istructs.PartitionID) istructs.Projector {
	return istructs.Projector{
		Name: QNameViewSubjectsIdx,
		Func: viewSubjectsIdxProjector,
	}
}

func viewSubjectsIdxProjector(event istructs.IPLogEvent, st istructs.IState, intents istructs.IIntents) (err error) {
	return event.CUDs(func(cdocSubject istructs.ICUDRow) error {
		if cdocSubject.QName() != QNameCDocSubject || !cdocSubject.IsNew() {
			return nil
		}

		skbViewSubjectsIdx, err := st.KeyBuilder(state.ViewRecordsStorage, QNameViewSubjectsIdx)
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
