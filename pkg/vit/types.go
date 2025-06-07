/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package vit

import (
	"context"
	"sync"
	"testing"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/coreutils"
	"github.com/voedger/voedger/pkg/extensionpoints"
	"github.com/voedger/voedger/pkg/goutils/testingu"
	"github.com/voedger/voedger/pkg/isecrets"
	"github.com/voedger/voedger/pkg/istructs"
	"github.com/voedger/voedger/pkg/state/smtptest"
	"github.com/voedger/voedger/pkg/vvm"
)

// Voedger Integration Test
type VIT struct {
	lock sync.Mutex
	T    testing.TB
	*vvm.VoedgerVM
	*vvm.VVMConfig
	cleanups             []func(vit *VIT)
	isFinalized          bool
	nextNumber           int
	appWorkspaces        map[appdef.AppQName]map[string]*AppWorkspace
	principals           map[appdef.AppQName]map[string]*Principal // because principals could be updated
	isOnSharedConfig     bool
	initialGoroutinesNum int
	configCleanupsAmount int
	emailCaptor          emailCaptor
	httpClient           coreutils.IHTTPClient
	mockTime             testingu.IMockTime
	vvmProblemCtx        context.Context
}

type VITConfig struct {
	opts     []VITConfigOptFunc
	isShared bool
}

type vitApps map[appdef.AppQName]*app // pointer here because options could be applied to app later, e.g. []logins

type vitPreConfig struct {
	vvmCfg       *vvm.VVMConfig
	vitApps      vitApps
	cleanups     []func(vit *VIT)
	initFuncs    []func()
	postInitFunc func(vit *VIT)
	secrets      map[string][]byte
}

type VITConfigOptFunc func(*vitPreConfig)
type AppOptFunc func(app *app, cfg *vvm.VVMConfig)
type vitOptFunc func(vit *VIT)
type signInOptFunc func(opts *signInOpts)
type signUpOptFunc func(opts *signUpOpts)
type PostConstructFunc func(intf interface{})

type Login struct {
	Name, Pwd         string
	PseudoProfileWSID istructs.WSID
	AppQName          appdef.AppQName
	subjectKind       istructs.SubjectKindType
	clusterID         istructs.ClusterID
	docs              map[appdef.QName]func(verifiedValues map[string]string) map[string]interface{}
	subjects          []subject
}

type WSParams struct {
	Name           string
	TemplateName   string
	TemplateParams string
	Kind           appdef.QName
	InitDataJSON   string
	ownerLoginName string
	ClusterID      istructs.ClusterID
	docs           map[appdef.QName]func(verifiedValues map[string]string) map[string]interface{}
	childs         []WSParams
	subjects       []subject
}

type subject struct {
	login       string
	subjectKind istructs.SubjectKindType
	roles       []appdef.QName
}

type WorkspaceDescriptor struct {
	WSParams
	WSID    istructs.WSID
	WSError string
}

type AppWorkspace struct {
	WorkspaceDescriptor
	Owner *Principal // because tokens of the principal will be updated when the time will be changed
}

func (a *AppWorkspace) AppQName() appdef.AppQName { return a.Owner.AppQName }

type Principal struct {
	Login
	Token       string
	ProfileWSID istructs.WSID
}

type verifiedValueIntent struct {
	docQName     appdef.QName
	fieldName    string
	desiredValue string
}

func (p *Principal) GetWSID() istructs.WSID       { return p.ProfileWSID }
func (p *Principal) GetAppQName() appdef.AppQName { return p.AppQName }

type app struct {
	name                  appdef.AppQName
	logins                []Login
	ws                    map[string]WSParams
	wsTemplateFuncs       []func(extensionpoints.IExtensionPoint)
	verifiedValuesIntents map[string]verifiedValueIntent
}

type BLOB struct {
	Content     []byte
	Name        string
	ContentType string
}

type signInOpts struct {
	failOnTimeout bool
}

type signUpOpts struct {
	profileClusterID istructs.ClusterID
	reqOpts          []coreutils.ReqOptFunc
}

type emailCaptor chan smtptest.Message

type implVITISecretsReader struct {
	secrets          map[string][]byte
	underlyingReader isecrets.ISecretReader
}
