/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vit

import (
	"sync"
	"testing"
	"time"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state/smtptest"
	coreutils "github.com/voedger/voedger/pkg/utils"
	"github.com/voedger/voedger/pkg/vvm"
)

// Voedger Integration Test
type VIT struct {
	lock sync.Mutex
	T    *testing.T
	*vvm.VoedgerVM
	*vvm.VVMConfig
	cleanups             []func(vit *VIT)
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

type VITConfig struct {
	opts     []vitConfigOptFunc
	isShared bool
}

type vitApps map[istructs.AppQName]*app // указатель потому, что к app потом будут опции применяться ([]logins, например)

type vitPreConfig struct {
	vvmCfg   *vvm.VVMConfig
	vitApps  vitApps
	cleanups []func(vit *VIT)
}

type vitConfigOptFunc func(*vitPreConfig)
type AppOptFunc func(app *app, cfg *vvm.VVMConfig)
type vitOptFunc func(vit *VIT)
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
	wsTemplateFuncs []func(extensionpoints.IExtensionPoint)
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
	reqOpts          []coreutils.ReqOptFunc
}

type EmailCaptor struct {
	vit *VIT
	ch  chan smtptest.Message
}
