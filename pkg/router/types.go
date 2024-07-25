/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Denis Gribanov
 */

package router

import (
	"net"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/acme/autocert"

	ibus "github.com/voedger/voedger/staging/src/github.com/untillpro/airs-ibus"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iblobstorage"
	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/iprocbus"
	"github.com/voedger/voedger/pkg/iprocbusmem"
	"github.com/voedger/voedger/pkg/istructs"
)

type RouterParams struct {
	Port                 int
	WriteTimeout         int
	ReadTimeout          int
	ConnectionsLimit     int
	HTTP01ChallengeHosts []string
	CertDir              string
	RouteDefault         string            // http://10.0.0.3:3000/not-found : https://alpha.dev.untill.ru/unknown/foo -> http://10.0.0.3:3000/not-found/unknown/foo
	Routes               map[string]string // /grafana=http://10.0.0.3:3000 : https://alpha.dev.untill.ru/grafana/foo -> http://10.0.0.3:3000/grafana/foo
	RoutesRewrite        map[string]string // /grafana-rewrite=http://10.0.0.3:3000/rewritten : https://alpha.dev.untill.ru/grafana-rewrite/foo -> http://10.0.0.3:3000/rewritten/foo
	RouteDomains         map[string]string // resellerportal.dev.untill.ru=http://resellerportal : https://resellerportal.dev.untill.ru/foo -> http://resellerportal/foo
}

type httpService struct {
	RouterParams
	*BlobberParams
	listenAddress      string
	router             *mux.Router
	server             *http.Server
	listener           net.Listener
	n10n               in10n.IN10nBroker
	blobWG             sync.WaitGroup
	bus                ibus.IBus
	busTimeout         time.Duration
	numsAppsWorkspaces map[appdef.AppQName]istructs.NumAppWorkspaces
	name               string
	listeningPort      atomic.Int32
}

type httpsService struct {
	*httpService
	crtMgr *autocert.Manager
}

type acmeService struct {
	http.Server
}

type BLOBMaxSizeType int64

type BlobberServiceChannels []iprocbusmem.ChannelGroup

type BlobberParams struct {
	ServiceChannels        []iprocbusmem.ChannelGroup
	BLOBStorage            iblobstorage.IBLOBStorage
	BLOBWorkersNum         int
	procBus                iprocbus.IProcBus
	RetryAfterSecondsOn503 int
	BLOBMaxSize            BLOBMaxSizeType
}

type route struct {
	targetURL  *url.URL
	isRewrite  bool
	fromDomain string
}

type subscriberParamsType struct {
	Channel       in10n.ChannelID
	ProjectionKey []in10n.ProjectionKey
}
