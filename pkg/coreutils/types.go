/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/voedger/voedger/pkg/istructs"
)

type EmbedFS interface {
	Open(name string) (fs.File, error)
	ReadDir(name string) ([]fs.DirEntry, error)
	ReadFile(name string) ([]byte, error)
}

type HTTPResponse struct {
	Body              string
	HTTPResp          *http.Response
	expectedHTTPCodes []int
}

type ReqOptFunc func(opts *reqOpts)

// implements json.Unmarshaler
type CommandResponse struct {
	NewIDs            map[string]istructs.RecordID
	CurrentWLogOffset istructs.Offset
	CmdResult         map[string]interface{}
}

type QueryResponse struct {
	QPv2Response // TODO: eliminate this after https://github.com/voedger/voedger/issues/1313
	Sections     []struct {
		Elements [][][][]interface{} `json:"elements"`
	} `json:"sections"`
}

type QPv2Response []map[string]interface{}

func (r QPv2Response) Result() map[string]interface{} {
	return r.ResultRow(0)
}

func (r QPv2Response) ResultRow(rowNum int) map[string]interface{} {
	return r[rowNum]
}

// implements json.Unmarshaler
type FuncResponse struct {
	*HTTPResponse
	CommandResponse
	QueryResponse
	SysError error
}

type IHTTPClient interface {
	Req(ctx context.Context, urlStr string, body string, optFuncs ...ReqOptFunc) (*HTTPResponse, error)
	ReqReader(ctx context.Context, urlStr string, bodyReader io.Reader, optFuncs ...ReqOptFunc) (*HTTPResponse, error)
	CloseIdleConnections()
}

type PathReader struct {
	rootPath string
}

func NewPathReader(rootPath string) *PathReader {
	return &PathReader{
		rootPath: rootPath,
	}
}

func (r *PathReader) Open(name string) (fs.File, error) {
	return os.Open(filepath.Join(r.rootPath, name))
}

func (r *PathReader) ReadDir(name string) ([]os.DirEntry, error) {
	return os.ReadDir(filepath.Join(r.rootPath, name))
}

func (r *PathReader) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(r.rootPath, name))
}

type IErrUnwrapper interface {
	Unwrap() []error
}

type CUDs struct {
	Values []CUD `json:"cuds"`
}

func (c CUDs) MustToJSON() string {
	v, err := c.ToJSON()
	if err != nil {
		panic(err)
	}
	return v
}
func (c CUDs) ToJSON() (v string, err error) {
	bb, err := json.Marshal(c)
	if err != nil {
		return
	}
	return string(bb), nil
}

type CUD struct {
	ID     istructs.RecordID      `json:"sys.ID,omitempty"`
	Fields map[string]interface{} `json:"fields"`
}

type IReadFS interface {
	fs.ReadDirFS
	fs.ReadFileFS
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

	if raw, ok := m["HTTPResponse"]; ok && len(raw) > 0 {
		if err := json.Unmarshal(raw, &resp.HTTPResponse); err != nil {
			return err
		}
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
		var sysError SysError
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
			var sysError SysError
			if commonErrorStatusIntf, ok := apiV2Err["status"]; ok {
				sysError.HTTPStatus = int(commonErrorStatusIntf.(float64))
			}
			if commonErrorMessageIntf, ok := apiV2Err["message"]; ok {
				sysError.Message = commonErrorMessageIntf.(string)
			}
			resp.SysError = sysError
		} else {
			var sysError SysError
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
