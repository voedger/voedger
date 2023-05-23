/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package main

import (
	"github.com/voedger/voedger/pkg/ibus"
	"github.com/voedger/voedger/pkg/ihttp"
	"github.com/voedger/voedger/pkg/ihttpctl"
)

type WiredServer struct {
	ibus.IBus
	ihttp.IHTTPProcessor
	ihttp.IHTTPProcessorAPI
	ihttpctl.IHTTPProcessorController
}

