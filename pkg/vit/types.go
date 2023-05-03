/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vit

import (
	"sync"
	"testing"
	"time"

	"github.com/untillpro/airs-bp3/utils"
	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state/smtptest"
	"github.com/voedger/voedger/pkg/vvm"
)

// Heeus Integration Test
type HIT struct {
	lock sync.Mutex
	T    *testing.T
	*vvm.HeeusVM
	HVMConfig            *vvm.HVMConfig
	cleanups             []func(hit *HIT)
	isFinalized          bool
	nextNumber           int
	appWorkspaces        map[istructs.AppQName]map[string]*AppWorkspace
	principals           map[istructs.AppQName]map[string]*Principal // потому что принципалы обновляются
	isOnSharedConfig     bool
	initialGoroutinesNum int
	configCleanupsAmount int
	emailMessagesChan    chan smtptest.Message
}

type timeService struct {
	m              sync.Mutex
	currentInstant time.Time
}

type HITConfig struct {
	opts     []hitConfigOptFunc
	isShared bool
}

type hitApps map[istructs.AppQName]*app // указатель потому, что к app потом будут опции применяться ([]logins, например)

type hitPreConfig struct {
	hvmCfg   *vvm.HVMConfig
	hitApps  hitApps
	cleanups []func(hit *HIT)
}

type hitConfigOptFunc func(*hitPreConfig)
type AppOptFunc func(app *app, cfg *vvm.HVMConfig)
type hitOptFunc func(hit *HIT)
type signInOptFunc func(opts *signInOpts)
type signUpOptFunc func(opts *signUpOpts)
type PostConstructFunc func(intf interface{})

type Login struct {
	Name, Pwd         string
	PseudoProfileWSID istructs.WSID
	AppQName          istructs.AppQName
	subjectKind       istructs.SubjectKindType
	clusterID         istructs.ClusterID
	singletons        map[appdef.QName]map[string]interface{}
}

type WSParams struct {
	Name           string
	TemplateName   string
	TemplateParams string
	Kind           appdef.QName
	InitDataJSON   string
	ownerLoginName string
	ClusterID      istructs.ClusterID
	singletons     map[appdef.QName]map[string]interface{}
}

type WorkspaceDescriptor struct {
	WSParams
	WSID    istructs.WSID
	WSError string
}

type AppWorkspace struct {
	WorkspaceDescriptor
	Owner *Principal // потому что токены принципала обновляются, когда меняется время
}

type Principal struct {
	Login
	Token       string
	ProfileWSID istructs.WSID
}

type app struct {
	name            istructs.AppQName
	logins          []Login
	ws              map[string]WSParams
	wsTemplateFuncs []func(vvm.IStandardExtensionPoints)
}

type BLOB struct {
	Content  []byte
	Name     string
	MimeType string
}

type signInOpts struct {
	failOnTimeout bool
}

type signUpOpts struct {
	profileClusterID istructs.ClusterID
	reqOpts          []utils.ReqOptFunc
}

type EmailCaptor struct {
	hit *HIT
	ch  chan smtptest.Message
}
