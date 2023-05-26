/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"io/fs"
	"net/http"
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
	SysError          SysError `json:"sys.Error"`
}

type FuncError struct {
	SysError
	ExpectedHTTPCodes []int
}
