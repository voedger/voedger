/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package ihttpctl

import (
	"fmt"

	"github.com/voedger/voedger/pkg/ihttp"
)

func NewHTTPProcessorController(api ihttp.IHTTPProcessorAPI, staticResources []StaticResourcesType) (IHTTPProcessorController, error) {
	srs := StaticResourcesType{}
	for _, sr := range staticResources {
		for url, fs := range sr {
			if _, exists := srs[url]; exists {
				return nil, fmt.Errorf("static resource with duplicate url %s", url) // TODO: panic
			}
			srs[url] = fs
		}
	}
	return &httpProcessorController{
		api:             api,
		staticResources: srs,
	}, nil
}
