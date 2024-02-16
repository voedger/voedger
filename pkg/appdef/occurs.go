/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

import "strconv"

const (
	Occurs_Unbounded    = Occurs(0xffff)
	Occurs_UnboundedStr = "unbounded"
)

func (o Occurs) String() string {
	switch o {
	case Occurs_Unbounded:
		return Occurs_UnboundedStr
	default:
		const base = 10
		return strconv.FormatUint(uint64(o), base)
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
