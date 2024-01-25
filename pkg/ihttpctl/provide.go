/*
* Copyright (c) 2022-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package ihttpctl

import (
	"fmt"

	"github.com/voedger/voedger/pkg/ihttp"
)

func NewHTTPProcessorController(processor ihttp.IHTTPProcessor, staticResources []StaticResourcesType, redirections RedirectRoutes, defaultRedirection DefaultRedirectRoute, acmeDomains ihttp.AcmeDomains, appRequestHandlers AppRequestHandlers) (IHTTPProcessorController, error) {
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
		processor:          processor,
		staticResources:    srs,
		redirections:       redirections,
		defaultRedirection: defaultRedirection,
		apps:               appRequestHandlers,
	}
	for _, acmeDomain := range acmeDomains {
		httpController.processor.AddAcmeDomain(acmeDomain)
	}
	return httpController, nil
}
