/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package ihttpctl

import (
	"fmt"

	"github.com/voedger/voedger/pkg/ihttp"
)

func NewHTTPProcessorController(api ihttp.IHTTPProcessorAPI, staticResources []StaticResourcesType, redirections RedirectRoutes, defaultRedirection DefaultRedirectRoute, acmeDomains AcmeDomains) (IHTTPProcessorController, error) {
	srs := StaticResourcesType{}
	for _, sr := range staticResources {
		for url, fs := range sr {
			if _, exists := srs[url]; exists {
				panic(fmt.Sprintf("static resource with duplicate url %s", url))
			}
			srs[url] = fs
		}
	}
	if len(defaultRedirection) > 1 {
		panic("default redirection should be single record")
	}
	httpController := &httpProcessorController{
		api:                api,
		staticResources:    srs,
		redirections:       redirections,
		defaultRedirection: defaultRedirection,
	}
	for _, acmeDomain := range acmeDomains {
		httpController.api.AddAcmeDomain(acmeDomain)
	}
	return httpController, nil
}
