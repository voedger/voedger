/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package ihttpctl

import (
	"github.com/untillpro/voedger/pkg/ihttp"
	"github.com/untillpro/voedger/pkg/iservices"
)

// Proposed factory signature
type NewType func(api ihttp.IHTTPProcessorAPI) (intf IHTTPProcessorController, cleanup func(), err error)

type IHTTPProcessorController interface {
	iservices.IService
}
