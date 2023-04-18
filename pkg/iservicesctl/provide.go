/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package iservicesctl

import (
	"github.com/voedger/voedger/pkg/iservices"
)

func New() (impl iservices.IServicesController) {
	return &servicesController{}
}
