/*
 * Copyright (c) 2023-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import (
	"strconv"

	"github.com/voedger/voedger/pkg/coreutils/utils"
)

func (o Occurs) String() string {
	switch o {
	case Occurs_Unbounded:
		return Occurs_UnboundedStr
	default:
		return utils.UintToString(o)
	}
}

func (o Occurs) MarshalJSON() ([]byte, error) {
	s := o.String()
	if o == Occurs_Unbounded {
		s = strconv.Quote(s)
	}
	return []byte(s), nil
}

func (o *Occurs) UnmarshalJSON(data []byte) (err error) {
	switch string(data) {
	case strconv.Quote(Occurs_UnboundedStr):
		*o = Occurs_Unbounded
		return nil
	default:
		var i uint64
		const base, wordBits = 10, 16
		i, err = strconv.ParseUint(string(data), base, wordBits)
		if err == nil {
			*o = Occurs(i)
		}
		return err
	}
}
