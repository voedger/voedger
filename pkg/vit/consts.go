/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vit

import (
	"embed"
	"time"

	"github.com/voedger/voedger/pkg/appparts"
	"github.com/voedger/voedger/pkg/istructs"
)

const (
	debugTimeout                 = time.Hour
	day                          = 24 * time.Hour
	defaultWorkspaceAwaitTimeout = 3 * time.Minute // so long for Test_Race_RestaurantIntenseUsage with -race
	testTimeout                  = 10 * time.Second
	workspaceQueryDelay          = 30 * time.Millisecond
	allowedGoroutinesNumDiff     = 200
	field_Input                  = "Input"
	testEmailsAwaitingTimeout    = 5 * time.Second
	testTimeMillis               = 1649667286774
	testRegistryPartsNum         = 2
	testAppPartsNum              = 5
)

var (
	vits = map[*VITConfig]*VIT{}
	//go:embed schemaTestApp1.vsql
	SchemaTestApp1FS embed.FS
	//go:embed schemaTestApp2.vsql
	SchemaTestApp2FS embed.FS
	//go:embed schemaTestApp2WithJob.vsql
	SchemaTestApp2WithJobFS embed.FS

	DefaultTestAppEnginesPool = appparts.PoolSize(10, 10, 20, 10)
	maxRateLimit2PerMinute    = istructs.RateLimit{
		Period:                time.Minute,
		MaxAllowedPerDuration: 2,
	}
	maxRateLimit4PerHour = istructs.RateLimit{
		Period:                time.Hour,
		MaxAllowedPerDuration: 4,
	}
	TestAppDeploymentDescriptor = appparts.AppDeploymentDescriptor{
		NumParts:         testAppPartsNum,
		EnginePoolSize:   DefaultTestAppEnginesPool,
		NumAppWorkspaces: istructs.DefaultNumAppWorkspaces,
	}
)
