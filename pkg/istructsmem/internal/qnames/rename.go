/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package qnames

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
)

func renameQName(storage istorage.IAppStorage, oldQName, newQName appdef.QName) error {
	const (
		errFmt     = "can not rename QName from «%v» to «%v»: %s"
		errWrapFmt = errFmt + ": %w"
	)

	if oldQName == newQName {
		return fmt.Errorf(errFmt, oldQName, newQName, "names are equals")
	}

	vers := vers.New()
	if err := vers.Prepare(storage); err != nil {
		return fmt.Errorf(errWrapFmt, oldQName, newQName, "unable to read versions", err)
	}

	qnames := New()
	if err := qnames.Prepare(storage, vers, nil); err != nil {
		return fmt.Errorf(errWrapFmt, oldQName, newQName, "unable to read qnames", err)
	}

	id, err := qnames.ID(oldQName)
	if err != nil {
		return fmt.Errorf(errWrapFmt, oldQName, newQName, "old not found", err)
	}

	if exists, err := qnames.ID(newQName); err == nil {
		return fmt.Errorf(errWrapFmt, oldQName, newQName, fmt.Sprintf("new already exists (id=%v)", exists), err)
	}

	set := func(n appdef.QName, id istructs.QNameID) {
		qnames.qNames[n] = id
		qnames.ids[id] = n
		qnames.changes++
	}

	set(oldQName, istructs.NullQNameID)
	set(newQName, id)

	if err := qnames.store(storage, vers); err != nil {
		return fmt.Errorf(errWrapFmt, oldQName, newQName, "unable to write storage", err)
	}

	return nil
}
