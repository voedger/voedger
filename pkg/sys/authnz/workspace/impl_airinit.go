/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 */

package workspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"strconv"
	"strings"

	airc_ticket_layouts "github.com/untillpro/airc-ticket-layouts"
	workspacemgmt "github.com/untillpro/airs-bp3/packages/air/workspace"
	"github.com/voedger/voedger/pkg/istructs"
	coreutils "github.com/voedger/voedger/pkg/utils"
)

func airWSPostInit(targetAppQName istructs.AppQName, newWSID istructs.WSID, federationURL *url.URL, authToken string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("air location post-init failed: %w", err)
		}
	}()

	//Get sys.WorkspaceDescriptor
	collectionURL := fmt.Sprintf("api/%s/%d/q.sys.Collection", targetAppQName.String(), newWSID)
	body := `{"args":{"Schema":"sys.WorkspaceDescriptor"},"elements":[{"fields":["OwnerWSID"]}]}`
	resp, err := coreutils.FederationFunc(federationURL, collectionURL, body, coreutils.WithAuthorizeBy(authToken))
	if err != nil {
		return
	}
	workspaceDescriptor := resp.SectionRow(0)

	//Get air.UserProfile
	collectionURL = fmt.Sprintf("api/%s/%d/q.sys.Collection", targetAppQName.String(), int64(workspaceDescriptor[0].(float64)))
	body = `{"args":{"Schema":"air.UserProfile"},"elements":[{"fields":["sys.ID","Country"]}]}`
	resp, err = coreutils.FederationFunc(federationURL, collectionURL, body, coreutils.WithAuthorizeBy(authToken))
	if err != nil {
		return
	}
	if resp.IsEmpty() {
		return errUserProfileNotInitialized
	}
	userProfile := resp.SectionRow(0)

	initialize := func(fileName string) (NewIDs map[string]int64, err error) {
		path := make([]string, 0, pathCap)
		path = append(path, "postinit")
		if fileName != "common" {
			path = append(path, strings.ToLower(userProfile[1].(string)))
		}
		path = append(path, fileName+".json")
		bb, err := Postinit.ReadFile(strings.Join(path, "/"))
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		if err != nil {
			return
		}

		entities := make([]map[string]interface{}, 0)
		err = json.Unmarshal(bb, &entities)
		if err != nil {
			return
		}

		cc := cuds{}
		for _, entity := range entities {
			cc.Cuds = append(cc.Cuds, cud{Fields: entity})
		}
		bb, err = json.Marshal(cc)
		if err != nil {
			return
		}

		initURL := fmt.Sprintf("api/%s/%d/c.sys.Init", targetAppQName.String(), newWSID)
		resp, err = coreutils.FederationFunc(federationURL, initURL, string(bb), coreutils.WithAuthorizeBy(authToken))
		if err != nil {
			return
		}

		return resp.NewIDs, nil
	}

	//Store ticket layout BLOBs
	newTicketBLOBIDs := make(map[interface{}]interface{})
	for code, blob := range airc_ticket_layouts.ProvideTicketLayouts() {
		blobURL := fmt.Sprintf("blob/%s/%d?name=ticketData&mimeType=application/x-binary", targetAppQName, newWSID)
		resp, err := coreutils.FederationPOST(federationURL, blobURL, blob, coreutils.WithAuthorizeBy(authToken), coreutils.WithHeaders(coreutils.ContentType, "application/x-www-form-urlencoded"))
		if err != nil {
			return err
		}
		blobID, err := strconv.Atoi(resp.Body)
		if err != nil {
			return err
		}
		newTicketBLOBIDs[code] = blobID
	}

	newCurrencyAndPaymentsIDs, err := initialize("currency_and_payments")
	if err != nil {
		return
	}
	_, err = initialize("vat")
	if err != nil {
		return
	}
	newCommonIDs, err := initialize("common")
	if err != nil {
		return
	}

	//Store untill.tickets
	bb, err := Postinit.ReadFile(strings.Join([]string{"postinit", "ticket.json"}, "/"))
	if err != nil {
		return
	}
	tickets := make([]map[string]interface{}, 0, ticketsCap)
	err = json.Unmarshal(bb, &tickets)
	if err != nil {
		return
	}
	for i := range tickets {
		tickets[i]["ticket_data_id"] = newTicketBLOBIDs[tickets[i]["airc_ticket_layout_code"]]
		delete(tickets[i], "airc_ticket_layout_code")
	}
	cc := cuds{}
	for _, entity := range tickets {
		cc.Cuds = append(cc.Cuds, cud{Fields: entity})
	}
	bb, err = json.Marshal(cc)
	if err != nil {
		return
	}

	initURL := fmt.Sprintf("api/%s/%d/c.sys.Init", targetAppQName.String(), newWSID)
	resp, err = coreutils.FederationFunc(federationURL, initURL, string(bb), coreutils.WithAuthorizeBy(authToken))
	if err != nil {
		return
	}
	newTicketIDs := resp.NewIDs

	//Get air.Restaurant
	collectionURL = fmt.Sprintf("api/%s/%d/q.sys.Collection", targetAppQName.String(), newWSID)
	body = `{"args":{"Schema":"air.Restaurant"},"elements":[{"fields":["sys.ID"]}]}`
	resp, err = coreutils.FederationFunc(federationURL, collectionURL, body, coreutils.WithAuthorizeBy(authToken))
	if err != nil {
		return
	}
	restaurant := resp.SectionRow(0)

	//Set restaurant default entities
	data := make(map[string]interface{})
	data[workspacemgmt.Field_DefaultCurrency] = newCurrencyAndPaymentsIDs["1"]
	data[workspacemgmt.Field_DefaultPriceID] = newCommonIDs["1"]
	data[workspacemgmt.Field_DefaultSpaceID] = newCommonIDs["2"]
	data[workspacemgmt.Field_XZReportTicketLayout] = newTicketIDs["107"]
	data[workspacemgmt.Field_NextCourseTicketLayout] = newTicketIDs["101"]
	data[workspacemgmt.Field_TransferTicketLayout] = newTicketIDs["104"]
	data[workspacemgmt.Field_BillTicketLayout] = newTicketIDs["105"]
	data[workspacemgmt.Field_OrderTicketLayout] = newTicketIDs["106"]
	data[workspacemgmt.Field_LangOfService] = workspacemgmt.LangOfService.Get(userProfile[1].(string))

	bb, err = json.Marshal(data)
	if err != nil {
		return
	}

	cudURL := fmt.Sprintf("api/%s/%d/c.sys.CUD", targetAppQName.String(), newWSID)
	body = fmt.Sprintf(`{"cuds":[{"sys.ID":%d,"fields":%s}]}`, int64(restaurant[0].(float64)), string(bb))
	_, err = coreutils.FederationFunc(federationURL, cudURL, body, coreutils.WithAuthorizeBy(authToken), coreutils.WithDiscardResponse())

	return err
}
