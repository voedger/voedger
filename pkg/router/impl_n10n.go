/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/voedger/voedger/pkg/coreutils/utils"
	"github.com/voedger/voedger/pkg/goutils/logger"

	"github.com/voedger/voedger/pkg/in10n"
	"github.com/voedger/voedger/pkg/in10nmem"
	in10nmemv1 "github.com/voedger/voedger/pkg/in10nmem/v1"
	"github.com/voedger/voedger/pkg/istructs"
)

/*
curl -G --data-urlencode "payload={\"SubjectLogin\": \"paa\", \"ProjectionKey\":[{\"App\":\"Application\",\"Projection\":\"paa.price\",\"WS\":1}, {\"App\":\"Application\",\"Projection\":\"paa.wine_price\",\"WS\":1}]}" https://alpha2.dev.untill.ru/n10n/channel -H "Content-Type: application/json"
*/
func (s *httpService) subscribeAndWatchHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		var (
			urlParams in10nmem.CreateChannelParamsType
			channel   in10n.ChannelID
			flusher   http.Flusher
			err       error
		)
		rw.Header().Set("Content-Type", "text/event-stream")
		rw.Header().Set("Cache-Control", "no-cache")
		rw.Header().Set("Connection", "keep-alive")
		jsonParam, ok := req.URL.Query()["payload"]
		if !ok || len(jsonParam[0]) < 1 {
			errMsg := "query parameter with payload (SubjectLogin id and ProjectionKey) is missing"
			logger.Error(errMsg)
			WriteTextResponse(rw, errMsg, http.StatusBadRequest)
			return
		}
		if err = json.Unmarshal([]byte(jsonParam[0]), &urlParams); err != nil {
			err = fmt.Errorf("cannot unmarshal input payload %w", err)
			logger.Error(err)
			WriteTextResponse(rw, err.Error(), http.StatusBadRequest)
			return
		}
		logger.Info("n10n subscribeAndWatch: ", urlParams)
		flusher, ok = rw.(http.Flusher)
		if !ok {
			// notest
			WriteTextResponse(rw, "streaming unsupported!", http.StatusInternalServerError)
			return
		}
		channel, err = s.n10n.NewChannel(urlParams.SubjectLogin, hours24)
		if err != nil {
			logger.Error(err)
			WriteTextResponse(rw, "create new channel failed: "+err.Error(), n10nErrorToStatusCode(err))
			return
		}
		if _, err = fmt.Fprintf(rw, "event: channelId\ndata: %s\n\n", channel); err != nil {
			logger.Error("failed to write created channel id to client:", err)
			return
		}
		for _, projection := range urlParams.ProjectionKey {
			if err = s.n10n.Subscribe(channel, projection); err != nil {
				logger.Error(err)
				WriteTextResponse(rw, "subscribe failed: "+err.Error(), n10nErrorToStatusCode(err))
				return
			}
		}
		flusher.Flush()
		ch := make(chan in10nmem.UpdateUnit)
		watchChannelContext, cancel := context.WithCancel(req.Context())
		go func() {
			defer close(ch)
			s.n10n.WatchChannel(watchChannelContext, channel, func(projection in10n.ProjectionKey, offset istructs.Offset) {
				var unit = in10nmem.UpdateUnit{
					Projection: projection,
					Offset:     offset,
				}
				ch <- unit
			})
		}()
		defer logger.Info("watch done")
		for req.Context().Err() == nil {
			result, ok := <-ch
			if !ok {
				break
			}
			sseMessage := fmt.Sprintf("event: %s\ndata: %s\n\n", result.Projection.ToJSON(), utils.UintToString(result.Offset))
			if _, err = fmt.Fprint(rw, sseMessage); err != nil {
				logger.Error("failed to write sse message for subjectLogin", urlParams.SubjectLogin, "to client:", sseMessage, ":", err.Error())
				break // WatchChannel will be finished on cancel()
			}
			if logger.IsVerbose() {
				logger.Verbose(fmt.Sprintf("sse message sent for subjectLogin %s:", urlParams.SubjectLogin), strings.ReplaceAll(sseMessage, "\n", " "))
			}
			flusher.Flush()
		}
		// graceful client disconnect -> req.Context() closed
		// failed to write sse message -> need to close watchChannelContext
		cancel()
		for range ch {
		}
	}
}

func n10nErrorToStatusCode(err error) int {
	switch {
	case errors.Is(err, in10n.ErrChannelDoesNotExist), errors.Is(err, in10nmemv1.ErrMetricDoesNotExists):
		return http.StatusBadRequest
	case errors.Is(err, in10n.ErrQuotaExceeded_Subsciptions), errors.Is(err, in10n.ErrQuotaExceeded_SubsciptionsPerSubject),
		errors.Is(err, in10n.ErrQuotaExceeded_Channels), errors.Is(err, in10n.ErrQuotaExceeded_ChannelsPerSubject):
		return http.StatusTooManyRequests
	}
	return http.StatusInternalServerError
}

/*
curl -G --data-urlencode "payload={\"Channel\": \"a23b2050-b90c-4ed1-adb7-1ecc4f346f2b\", \"ProjectionKey\":[{\"App\":\"Application\",\"Projection\":\"paa.wine_price\",\"WS\":1}]}" https://alpha2.dev.untill.ru/n10n/subscribe -H "Content-Type: application/json"
*/
func (s *httpService) subscribeHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		var parameters subscriberParamsType
		err := getJsonPayload(req, &parameters)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
		}
		logger.Info("n10n subscribe: ", parameters)
		for _, projection := range parameters.ProjectionKey {
			err = s.n10n.Subscribe(parameters.Channel, projection)
			if err != nil {
				logger.Error(err)
				http.Error(rw, "subscribe failed: "+err.Error(), n10nErrorToStatusCode(err))
				return
			}
		}
	}
}

/*
curl -G --data-urlencode "payload={\"Channel\": \"a23b2050-b90c-4ed1-adb7-1ecc4f346f2b\", \"ProjectionKey\":[{\"App\":\"Application\",\"Projection\":\"paa.wine_price\",\"WS\":1}]}" https://alpha2.dev.untill.ru/n10n/unsubscribe -H "Content-Type: application/json"
*/
func (s *httpService) unSubscribeHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		var parameters subscriberParamsType
		err := getJsonPayload(req, &parameters)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
		}
		logger.Info("n10n unsubscribe: ", parameters)
		for _, projection := range parameters.ProjectionKey {
			err = s.n10n.Unsubscribe(parameters.Channel, projection)
			if err != nil {
				logger.Error(err)
				http.Error(rw, err.Error(), n10nErrorToStatusCode(err))
				return
			}
		}
	}
}

// curl -X POST "http://localhost:3001/n10n/update" -H "Content-Type: application/json" -d "{\"App\":\"Application\",\"Projection\":\"paa.price\",\"WS\":1}"
// TODO: eliminate after airs-bp3 integration tests implementation
func (s *httpService) updateHandler() http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		var p in10n.ProjectionKey
		body, err := io.ReadAll(req.Body)
		if err != nil {
			logger.Error(err)
			http.Error(resp, "Error when read request body", http.StatusInternalServerError)
			return
		}
		err = json.Unmarshal(body, &p)
		if err != nil {
			logger.Error(err)
			http.Error(resp, "Error when parse request body", http.StatusBadRequest)
			return
		}

		params := mux.Vars(req)
		offset := params["offset"]
		if off, err := strconv.ParseUint(offset, utils.DecimalBase, utils.BitSize64); err == nil {
			s.n10n.Update(p, istructs.Offset(off))
		}
	}
}

func getJsonPayload(req *http.Request, payload *subscriberParamsType) (err error) {
	jsonParam, ok := req.URL.Query()["payload"]
	if !ok || len(jsonParam[0]) < 1 {
		err = errors.New("url parameter with payload (channel id and projection key) is missing")
		logger.Error(err)
		return err
	}
	err = json.Unmarshal([]byte(jsonParam[0]), payload)
	if err != nil {
		err = fmt.Errorf("cannot unmarshal input payload %w", err)
		logger.Error(err)
	}
	return err
}
