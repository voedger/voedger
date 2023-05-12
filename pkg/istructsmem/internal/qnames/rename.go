/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package qnames

import (
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
)

func renameQName(storage istorage.IAppStorage, old, new appdef.QName) error {
	const (
		errFmt     = "can not rename QName from «%v» to «%v»: %s"
		errWrapFmt = errFmt + ": %w"
	)

	if old == new {
		return fmt.Errorf(errFmt, old, new, "names are equals")
	}

	vers := vers.New()
	if err := vers.Prepare(storage); err != nil {
		return fmt.Errorf(errWrapFmt, old, new, "unable to read versions", err)
	}

	qnames := New()
	if err := qnames.Prepare(storage, vers, nil, nil); err != nil {
		return fmt.Errorf(errWrapFmt, old, new, "unable to read qnames", err)
	}

	id, err := qnames.ID(old)
	if err != nil {
		return fmt.Errorf(errWrapFmt, old, new, "old not found", err)
	}

	if exists, err := qnames.ID(new); err == nil {
		return fmt.Errorf(errWrapFmt, old, new, fmt.Sprintf("new already exists (id=%v)", exists), err)
	}

	set := func(n appdef.QName, id QNameID) {
		qnames.qNames[n] = id
		qnames.ids[id] = n
		qnames.changes++
	}

	set(old, NullQNameID)
	set(new, id)

	if err := qnames.store(storage, vers); err != nil {
		return fmt.Errorf(errWrapFmt, old, new, "unable to write storage", err)
	}

	return nil
}
