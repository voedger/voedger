/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package sysmonitor

import (
	"embed"
	"io/fs"

	"github.com/voedger/voedger/pkg/ihttpctl"
)

//go:embed site.hello/*
var sysMonitorSiteHelloFS embed.FS

//go:embed site.main/*
var sysMonitorSiteMainFS embed.FS

func New() ihttpctl.StaticResourcesType {
	var fsHello, fsMain fs.FS
	var err error
	fsHello, err = fs.Sub(sysMonitorSiteHelloFS, "site.hello")
	if err != nil {
		// notest
		panic(err)
	}
	fsMain, err = fs.Sub(sysMonitorSiteMainFS, "site.main")
	if err != nil {
		// notest
		panic(err)
	}
	return ihttpctl.StaticResourcesType{
		"sys/monitor/site/hello": fsHello,
		"sys/monitor/site/main":  fsMain,
	}
}
