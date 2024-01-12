/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package ibus

import "context"

// HTTPMethod s.e.
// see const.go/HTTPMethodGET...
type HTTPMethod int

// Request s.e.
type Request struct {
	Method HTTPMethod

	QueueID string

	// Always 0 for non-party queues
	WSID int64
	// Calculated by bus using WSID and queue parameters
	PartitionNumber int

	Header map[string][]string `json:",omitempty"`

	// Part of URL which follows: queue alias in non-party queues, part dividend in partitioned queues
	Resource string

	// Part of URL which follows ? (URL.Query())
	Query map[string][]string `json:",omitempty"`

	// Content of http.Request JSON-parsed Body
	Body []byte `json:",omitempty"`

	// attachment-name => attachment-id
	// Must be non-null
	Attachments map[string]string `json:",omitempty"`

	// airs-bp3
	// AppQName need to determine where to send c.sys.Init requests on creating a new workspace
	AppQName string

	Host string
}

// Response s.e.
type Response struct {
	ContentType string
	StatusCode  int
	Data        []byte
}

// SectionKind int
type SectionKind int

const (
	// SectionKindUnspecified s.e.
	SectionKindUnspecified SectionKind = iota
	// SectionKindMap s.e.
	SectionKindMap
	// SectionKindArray s.e.
	SectionKindArray
	// SectionKindObject s.e.
	SectionKindObject
)

// ISection s.e.
type ISection interface {
	Type() string
}

// IDataSection s.e.
type IDataSection interface {
	ISection
	Path() []string
}

// IObjectSection s.e.
type IObjectSection interface {
	IDataSection
	// Caller MUST call Value() even if it does not need the value
	// note: second and further Value() calls will return nil
	Value(ctx context.Context) []byte
}

// IArraySection s.e.
type IArraySection interface {
	IDataSection
	// Caller MUST call Next() until !ok
	Next(ctx context.Context) (value []byte, ok bool)
}

// IMapSection s.e.
type IMapSection interface {
	IDataSection
	// Caller MUST call Next() until !ok
	Next(ctx context.Context) (name string, value []byte, ok bool)
}

// IResultSender used by ParallelFunction
// If error happens in any Send* method all subsequent calls also return error
type IResultSender interface {
	// Must be called before first Send*
	// Can be called multiple times - each time new section started
	// Section path may NOT include data from database, only constants should be used
	StartArraySection(sectionType string, path []string)
	StartMapSection(sectionType string, path []string)
	ObjectSection(sectionType string, path []string, element interface{}) (err error)

	// For reading journal
	// StartBinarySection(sectionType string, path []string)

	// element should be "marshallable" by json package
	// Send* can be called multiple times per array
	// name is ignored for Array section
	// For reading journal
	// if element is []byte then it will be sent sent as is. Note: JSON malformation is possible for airs-router's http client. Sender must take care of this.
	SendElement(name string, element interface{}) (err error)
}

// IResultSenderClosable s.e.
type IResultSenderClosable interface {
	IResultSender
	// Close() must be the last call to the interface
	Close(err error)
}
