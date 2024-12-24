/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package ibus

import "time"

// DefaultTimeout for PostRequest func
const DefaultTimeout = 10 * time.Second

// HTTP methods
const (
	HTTPMethodGET = HTTPMethod(iota)
	HTTPMethodPOST
	HTTPMethodPUT
	HTTPMethodPATCH
	HTTPMethodDELETE
)

// NameToHTTPMethod maps RFC 7231 names to HTTPMethod
var NameToHTTPMethod = map[string]HTTPMethod{"GET": HTTPMethodGET, "POST": HTTPMethodPOST, "PUT": HTTPMethodPUT, "PATCH": HTTPMethodPATCH, "DELETE": HTTPMethodDELETE}

// HTTPMethodToName s.e.
var HTTPMethodToName = map[HTTPMethod]string{HTTPMethodGET: "GET", HTTPMethodPOST: "POST", HTTPMethodPUT: "PUT", HTTPMethodPATCH: "PATCH", HTTPMethodDELETE: "DELETE"}
