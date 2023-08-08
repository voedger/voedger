/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vit

import (
	"embed"
	"math"
	"time"

	coreutils "github.com/voedger/voedger/pkg/utils"
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
)

var (
	ts                        = &timeService{currentInstant: DefaultTestTime}
	vits                      = map[*VITConfig]*VIT{}
	DefaultTestTime           = time.UnixMilli(1649667286774) // 2022-04-11 11:54:46 +0300 MSK
	workspaceInitAwaitTimeout = defaultWorkspaceAwaitTimeout
	//go:embed schemasSimpleApp.sql
	schemasSimpleApp embed.FS
)

func init() {
	if coreutils.IsDebug() {
		workspaceInitAwaitTimeout = math.MaxInt
	}
}
