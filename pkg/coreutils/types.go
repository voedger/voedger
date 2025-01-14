/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"io"
	"io/fs"
	"net/http"
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

type CommandResponse struct {
	NewIDs            map[string]istructs.RecordID
	CurrentWLogOffset istructs.Offset
	SysError          SysError               `json:"sys.Error"`
	CmdResult         map[string]interface{} `json:"Result"`
}

type FuncResponse struct {
	*HTTPResponse
	CommandResponse
	Sections []struct {
		Elements [][][][]interface{} `json:"elements"`
	} `json:"sections"`
}

type FuncError struct {
	SysError
	ExpectedHTTPCodes []int
}

type IHTTPClient interface {
	Req(urlStr string, body string, optFuncs ...ReqOptFunc) (*HTTPResponse, error)
	ReqReader(urlStr string, bodyReader io.Reader, optFuncs ...ReqOptFunc) (*HTTPResponse, error)
	CloseIdleConnections()
}

type retrier struct {
	macther func(err error) bool
	timeout time.Duration
	delay   time.Duration
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

type CUD struct {
	Fields map[string]interface{} `json:"fields"`
}

type CUDs struct {
	Cuds []CUD `json:"cuds"`
}

type IReadFS interface {
	fs.ReadDirFS
	fs.ReadFileFS
}
