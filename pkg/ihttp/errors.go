/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package ihttp

import "errors"

var ErrUnknownApplication = errors.New("unknown application")
var ErrUnknownAppPartition = errors.New("unknown app partition")
var ErrUnknownDynamicSubresource = errors.New("unknown dynamic subresource")
var ErrUnknownDynamicSubresourceAlias = errors.New("unknown dynamic subresource alias")
