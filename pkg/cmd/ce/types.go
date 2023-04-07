/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package ce

import (
	"github.com/untillpro/voedger/pkg/ibus"
	"github.com/untillpro/voedger/pkg/ihttp"
	"github.com/untillpro/voedger/pkg/ihttpctl"
)

type WiredServer struct {
	ibus.IBus
	ihttp.IHTTPProcessor
	ihttp.IHTTPProcessorAPI
	ihttpctl.IHTTPProcessorController
}
