/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"encoding/binary"
	"fmt"

	"github.com/voedger/voedger/pkg/istorage"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/istructsmem/internal/consts"
	"github.com/voedger/voedger/pkg/istructsmem/internal/qnames"
	"github.com/voedger/voedger/pkg/istructsmem/internal/utils"
	"github.com/voedger/voedger/pkg/istructsmem/internal/vers"
)

func RenameQName(storage istorage.IAppStorage, old, new istructs.QName) error {

	const errFmt = "can not rename QName from «%v» to «%v»: "

	if old == new {
		return fmt.Errorf(errFmt+"names are equals", old, new)
	}

	data := make([]byte, 0)
	if ok, err := storage.Get(utils.ToBytes(consts.SysView_Versions), utils.ToBytes(vers.SysQNamesVersion), &data); !ok {
		return fmt.Errorf("error read version of QNames system view: %w", err)
	}

	ver := vers.VersionValue(binary.BigEndian.Uint16(data))

	switch ver {
	case verSysQNames01:
		pKey := utils.ToBytes(consts.SysView_QNames, verSysQNames01)

		ok, err := storage.Get(pKey, []byte(old.String()), &data)
		if err != nil {
			return fmt.Errorf(errFmt+"error read old QName ID: %w", old, new, err)
		}
		if !ok {
			return fmt.Errorf(errFmt+"old QName ID not found", old, new)
		}
		if id := qnames.QNameID(binary.BigEndian.Uint16(data)); id == qnames.NullQNameID {
			return fmt.Errorf(errFmt+"old QName already deleted, ID %v", old, new, id)
		}

		newData := make([]byte, 0)
		ok, err = storage.Get(pKey, []byte(new.String()), &newData)
		if err != nil {
			return fmt.Errorf(errFmt+"error checking existence of new QName ID: %w", old, new, err)
		}
		if ok {
			if id := qnames.QNameID(binary.BigEndian.Uint16(newData)); id != qnames.NullQNameID {
				return fmt.Errorf(errFmt+"new QName already exists, ID %v", old, new, id)
			}
		}

		if err := storage.Put(pKey, []byte(new.String()), data); err != nil {
			return fmt.Errorf(errFmt+"error write new QName ID: %w", old, new, err)
		}
		if err := storage.Put(pKey, []byte(old.String()), utils.ToBytes(qnames.NullQNameID)); err != nil {
			return fmt.Errorf(errFmt+"error write old QName ID: %w", old, new, err)
		}
	default:
		return fmt.Errorf("unsupported version of QNames system view: %v", ver)
	}

	return nil
}
