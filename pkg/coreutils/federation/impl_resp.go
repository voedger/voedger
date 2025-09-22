/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package federation

import (
	"encoding/json"

	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/istructs"
)

func (resp *FuncResponse) Len() int {
	return resp.NumRows()
}

func (resp *FuncResponse) NumRows() int {
	if resp.IsEmpty() {
		return 0
	}
	return len(resp.Sections[0].Elements)
}

func (resp *FuncResponse) SectionRow(rowIdx ...int) []interface{} {
	if len(rowIdx) > 1 {
		panic("must be 0 or 1 rowIdx'es")
	}
	if len(resp.Sections) == 0 {
		panic("empty response")
	}
	i := 0
	if len(rowIdx) == 1 {
		i = rowIdx[0]
	}
	return resp.Sections[0].Elements[i][0][0]
}

// returns a new ID for raw ID 1
func (resp *FuncResponse) NewID() istructs.RecordID {
	return resp.NewIDs["1"]
}

func (resp *FuncResponse) IsEmpty() bool {
	return len(resp.Sections) == 0 && len(resp.QPv2Response) == 0
}

// TODO: temporary solution. Eliminate after switching to APIv2
func (cr *CommandResponse) UnmarshalJSON(data []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	if raw, ok := m["NewIDs"]; ok {
		if err := json.Unmarshal(raw, &cr.NewIDs); err != nil {
			return err
		}
	} else if raw, ok = m["newIDs"]; ok {
		if err := json.Unmarshal(raw, &cr.NewIDs); err != nil {
			return err
		}
	}

	if raw, ok := m["CurrentWLogOffset"]; ok {
		if err := json.Unmarshal(raw, &cr.CurrentWLogOffset); err != nil {
			return err
		}
	} else if raw, ok = m["currentWLogOffset"]; ok {
		if err := json.Unmarshal(raw, &cr.CurrentWLogOffset); err != nil {
			return err
		}
	}

	if raw, ok := m["Result"]; ok {
		if err := json.Unmarshal(raw, &cr.CmdResult); err != nil {
			return err
		}
	} else if raw, ok = m["result"]; ok {
		if err := json.Unmarshal(raw, &cr.CmdResult); err != nil {
			return err
		}
	}

	return nil
}

// TODO: temporary solution. Eliminate after switching to APIv2
func (resp *FuncResponse) UnmarshalJSON(data []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	var commandResp CommandResponse
	if err := commandResp.UnmarshalJSON(data); err != nil {
		return err
	}
	resp.CommandResponse = commandResp

	if raw, ok := m["sections"]; ok && len(raw) > 0 {
		if err := json.Unmarshal(raw, &resp.Sections); err != nil {
			return err
		}
	}

	if raw, ok := m["results"]; ok && len(raw) > 0 {
		if err := json.Unmarshal(raw, &resp.QPv2Response); err != nil {
			return err
		}
	}

	if raw, ok := m["sys.Error"]; ok {
		var sysError coreutils.SysError
		if err := json.Unmarshal(raw, &sysError); err != nil {
			return err
		}
		resp.SysError = sysError
	} else {
		if raw, ok := m["error"]; ok {
			apiV2Err := map[string]interface{}{}
			if err := json.Unmarshal(raw, &apiV2Err); err != nil {
				return err
			}
			sysError := coreutils.SysError{}
			if commonErrorStatusIntf, ok := apiV2Err["status"]; ok {
				sysError.HTTPStatus = int(commonErrorStatusIntf.(float64))
			}
			if commonErrorMessageIntf, ok := apiV2Err["message"]; ok {
				sysError.Message = commonErrorMessageIntf.(string)
			}
			resp.SysError = sysError
		} else {
			sysError := coreutils.SysError{}
			if raw, ok := m["status"]; ok {
				if err := json.Unmarshal(raw, &sysError.HTTPStatus); err != nil {
					return err
				}
			}
			if raw, ok := m["message"]; ok {
				if err := json.Unmarshal(raw, &sysError.Message); err != nil {
					return err
				}
			}
			if !sysError.IsNil() {
				resp.SysError = sysError
			}
		}
	}

	return nil
}
