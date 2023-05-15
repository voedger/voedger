/*
 * Copyright (c) 2022-present unTill Pro, Ltd.
 */

package iauthnzimpl

import (
	"context"
	"fmt"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iauthnz"
	"github.com/voedger/voedger/pkg/istructs"
	payloads "github.com/voedger/voedger/pkg/itokens-payloads"
	"golang.org/x/exp/slices"
)

func (i *implIAuthenticator) Authenticate(requestContext context.Context, as istructs.IAppStructs, appTokens istructs.IAppTokens, req iauthnz.AuthnRequest) (principals []iauthnz.Principal, principalPayload payloads.PrincipalPayload, err error) {
	defer func() {
		if !principalPayload.IsAPIToken {
			principals = append(principals, iauthnz.Principal{
				Kind: iauthnz.PrincipalKind_Host,
				Name: req.Host,
			})
		}
	}()
	if len(req.Token) == 0 {
		return
	}

	if _, err = appTokens.ValidateToken(req.Token, &principalPayload); err != nil {
		return nil, principalPayload, err
	}

	if principalPayload.IsAPIToken {
		for _, role := range principalPayload.Roles {
			if role.WSID != req.RequestWSID {
				continue
			}
			principals = append(principals, iauthnz.Principal{
				Kind:  iauthnz.PrincipalKind_Role,
				WSID:  req.RequestWSID,
				QName: role.QName,
			})
		}
		return
	}

	// read roles from cdoc.sys.Subjects from the current workspace
	subjectRoles, err := i.subjectRolesGetter(requestContext, principalPayload.Login, as, req.RequestWSID)
	if err != nil {
		return nil, principalPayload, err
	}
	for _, sr := range subjectRoles {
		principals = append(principals, iauthnz.Principal{
			Kind:  iauthnz.PrincipalKind_Role,
			WSID:  req.RequestWSID,
			QName: sr,
		})
	}

	profileWSID := principalPayload.ProfileWSID // for user and device subject kinds
	pkt := iauthnz.PrincipalKind_NULL
	name := ""
	switch principalPayload.SubjectKind {
	case istructs.SubjectKind_User:
		pkt = iauthnz.PrincipalKind_User
		name = principalPayload.Login
	case istructs.SubjectKind_Device:
		pkt = iauthnz.PrincipalKind_Device
	default:
		return nil, principalPayload, fmt.Errorf("unsupported subject kind: %v", principalPayload)
	}

	// system role
	sysPrn := iauthnz.Principal{
		Kind:  iauthnz.PrincipalKind_Role,
		WSID:  req.RequestWSID,
		QName: iauthnz.QNameRoleSystem,
	}
	if slices.Contains(principals, sysPrn) {
		return // nothing else matters
	} else if profileWSID == istructs.NullWSID {
		principals = append(principals, sysPrn)
		return // nothing else matters
	}

	// user or device principal
	prn := iauthnz.Principal{
		Kind: pkt,
		WSID: profileWSID,
		Name: name,
	}
	if !slices.Contains(principals, prn) {
		principals = append(principals, prn)
	}

	workspaceSubject := false
	switch pkt {
	case iauthnz.PrincipalKind_User:
		if req.RequestWSID == profileWSID {
			prnProfileOwner := iauthnz.Principal{
				Kind:  iauthnz.PrincipalKind_Role,
				WSID:  req.RequestWSID,
				QName: iauthnz.QNameRoleProfileOwner,
			}
			if !slices.Contains(principals, prnProfileOwner) {
				principals = append(principals, prnProfileOwner)
			}
			workspaceSubject = true
		} else {
			wsDesc, err := as.Records().GetSingleton(req.RequestWSID, qNameCDocWorkspaceDescriptor)
			if err != nil {
				return nil, principalPayload, err
			}
			if wsDesc.QName() != appdef.NullQName {
				ownerWSID := wsDesc.AsInt64(field_OwnerWSID)
				prnWSOwner := iauthnz.Principal{
					Kind:  iauthnz.PrincipalKind_Role,
					WSID:  req.RequestWSID,
					QName: iauthnz.QNameRoleWorkspaceOwner,
				}
				if ownerWSID == int64(profileWSID) && !slices.Contains(principals, prnWSOwner) {
					principals = append(principals, prnWSOwner)
					workspaceSubject = true
				}
				// check roles came from token
				for _, role := range principalPayload.Roles {
					if role.WSID != istructs.WSID(ownerWSID) {
						continue
					}
					prn := iauthnz.Principal{
						Kind:  iauthnz.PrincipalKind_Role,
						WSID:  req.RequestWSID,
						QName: role.QName,
					}
					if !slices.Contains(principals, prn) {
						principals = append(principals, prn)
						if role.QName == iauthnz.QNameRoleProfileOwner || role.QName == iauthnz.QNameRoleWorkspaceOwner {
							workspaceSubject = true
						}
					}
				}
			}
		}
	case iauthnz.PrincipalKind_Device:
		deviceProfileWSID := principalPayload.ProfileWSID
		prnWSDevice := iauthnz.Principal{
			Kind:  iauthnz.PrincipalKind_Role,
			WSID:  deviceProfileWSID,
			QName: iauthnz.QNameRoleWorkspaceDevice,
		}
		if !slices.Contains(principals, prnWSDevice) {
			compRec, _, err := GetComputersRecByDeviceProfileWSID(as, req.RequestWSID, deviceProfileWSID)
			if err != nil {
				return nil, principalPayload, err
			}
			if compRec.QName() == appdef.NullQName {
				break
			}
			if compRec.AsBool(appdef.SystemField_IsActive) {
				principals = append(principals, iauthnz.Principal{
					Kind:  iauthnz.PrincipalKind_Role,
					WSID:  deviceProfileWSID,
					QName: iauthnz.QNameRoleWorkspaceDevice,
				})
				workspaceSubject = true
			}
		}
	}
	if workspaceSubject {
		prnWSSubject := iauthnz.Principal{
			Kind:  iauthnz.PrincipalKind_Role,
			WSID:  req.RequestWSID,
			QName: iauthnz.QNameRoleWorkspaceSubject,
		}
		if !slices.Contains(principals, prnWSSubject) {
			principals = append(principals, prnWSSubject)
		}
	}
	

	// ResellersAdmin || UntillPaymentsReseller -> WorkspaceAdmin
	for _, prn := range principals {
		if prn.Kind == iauthnz.PrincipalKind_Role && (prn.QName == qNameRoleResellersAdmin || prn.QName == qNameRoleUntillPaymentsReseller) {
			prnWSAdmin := iauthnz.Principal{
				Kind:  iauthnz.PrincipalKind_Role,
				WSID:  req.RequestWSID,
				QName: iauthnz.QNameRoleWorkspaceAdmin,
			}
			if !slices.Contains(principals, prnWSAdmin) {
				principals = append(principals, prnWSAdmin)
				break
			}
		}
	}

	return principals, principalPayload, nil
}

// principals obtained from IAuhtenticator
func (i *implIAuthorizer) Authorize(as istructs.IAppStructs, principals []iauthnz.Principal, req iauthnz.AuthzRequest) (ok bool, err error) {
	for _, prn := range principals {
		if prn.Kind == iauthnz.PrincipalKind_Role && prn.QName == iauthnz.QNameRoleSystem {
			return true, nil
		}
	}
	return i.acl.IsAllowed(principals, req), nil
}
