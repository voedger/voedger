/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package coreutils

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/voedger/voedger/pkg/istructs"
)

type EmbedFS interface {
	Open(name string) (fs.File, error)
	ReadDir(name string) ([]fs.DirEntry, error)
	ReadFile(name string) ([]byte, error)
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
