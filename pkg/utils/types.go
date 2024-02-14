/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/voedger/voedger/pkg/istructs"
)

type EmbedFS interface {
	Open(name string) (fs.File, error)
	ReadDir(name string) ([]fs.DirEntry, error)
	ReadFile(name string) ([]byte, error)
}

type HTTPResponse struct {
	Body                 string
	HTTPResp             *http.Response
	expectedSysErrorCode int
	expectedHTTPCodes    []int
}

type ReqOptFunc func(opts *reqOpts)

type FuncResponse struct {
	*HTTPResponse
	Sections []struct {
		Elements [][][][]interface{} `json:"elements"`
	} `json:"sections"`
	NewIDs            map[string]int64
	CurrentWLogOffset int64
	SysError          SysError               `json:"sys.Error"`
	CmdResult         map[string]interface{} `json:"Result"`
}

type FuncError struct {
	SysError
	ExpectedHTTPCodes []int
}

type IFederation interface {
	POST(appQName istructs.AppQName, wsid istructs.WSID, fn string, body string, opts ...ReqOptFunc) (*HTTPResponse, error)
	URL() *url.URL
}

type TimeFunc func() time.Time

type CommandProcessorsCount int

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
