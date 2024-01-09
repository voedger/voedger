/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package ihttp

import (
	"github.com/voedger/voedger/pkg/istorage"
)

type Alias struct {
	Domain string
	Path   string
}

type SectionsHandlerType func(section interface{})
type Status struct {
	// Ref. https://go.dev/src/net/http/status.go
	// StatusBadRequest(400) if server got the request but could not process it
	// StatusGatewayTimeout(504) if timeout expired
	HTTPStatus   int
	ErrorMessage string
	ErrorData    string
}
type IRouterStorage istorage.IAppStorage
type AcmeDomains []string
type CLIParams struct {
	Port        int
	AcmeDomains AcmeDomains
}
