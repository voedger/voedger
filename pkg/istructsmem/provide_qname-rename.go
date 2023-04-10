/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package istructsmem

import (
	"encoding/binary"
	"fmt"

	"github.com/untillpro/voedger/pkg/istorage"
	"github.com/untillpro/voedger/pkg/istructs"
)

func RenameQName(storage istorage.IAppStorage, old, new istructs.QName) error {

	const errFmt = "can not rename QName from «%v» to «%v»: "

	if old == new {
		return fmt.Errorf(errFmt+"names are equals", old, new)
	}

	data := make([]byte, 0)
	if ok, err := storage.Get(toBytes(uint16(QNameIDSysVesions)), toBytes(uint16(verSysQNames)), &data); !ok {
		return fmt.Errorf("error read version of QNames system view: %w", err)
	}

	ver := versionValueType(binary.BigEndian.Uint16(data))

	switch ver {
	case verSysQNames01:
		pKey := toBytes(uint16(QNameIDSysQNames), uint16(verSysQNames01))

		ok, err := storage.Get(pKey, []byte(old.String()), &data)
		if err != nil {
			return fmt.Errorf(errFmt+"error read old QName ID: %w", old, new, err)
		}
		if !ok {
			return fmt.Errorf(errFmt+"old QName ID not found", old, new)
		}
		if id := QNameID(binary.BigEndian.Uint16(data)); id == NullQNameID {
			return fmt.Errorf(errFmt+"old QName already deleted, ID %v", old, new, id)
		}

		newData := make([]byte, 0)
		ok, err = storage.Get(pKey, []byte(new.String()), &newData)
		if err != nil {
			return fmt.Errorf(errFmt+"error checking existence of new QName ID: %w", old, new, err)
		}
		if ok {
			if id := QNameID(binary.BigEndian.Uint16(newData)); id != NullQNameID {
				return fmt.Errorf(errFmt+"new QName already exists, ID %v", old, new, id)
			}
		}

		if err := storage.Put(pKey, []byte(new.String()), data); err != nil {
			return fmt.Errorf(errFmt+"error write new QName ID: %w", old, new, err)
		}
		if err := storage.Put(pKey, []byte(old.String()), toBytes(uint16(NullQNameID))); err != nil {
			return fmt.Errorf(errFmt+"error write old QName ID: %w", old, new, err)
		}
	default:
		return fmt.Errorf("unsupported version of QNames system view: %v", ver)
	}

	return nil
}
