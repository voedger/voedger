/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package oldacl

import (
	"slices"

	"github.com/voedger/voedger/pkg/appdef"
	"github.com/voedger/voedger/pkg/iauthnz"
)

func matchOrNotSpecified_Principals(pattern [][]iauthnz.Principal, actualPrns []iauthnz.Principal) (ok bool) {
	if len(pattern) == 0 {
		return true
	}
	for _, prnsAND := range pattern {
		matchedCount := 0
		for _, prnAND := range prnsAND {
			// all prnsAND must be among actualPrns
			isMatched := false
			for _, actualPrn := range actualPrns {
				// let's find prnAND
				if actualPrn.Kind != prnAND.Kind {
					continue
				}
				isMatched = func() bool {
					if prnAND.ID > 0 && prnAND.ID != actualPrn.ID {
						return false
					}
					if len(prnAND.Name) > 0 && prnAND.Name != actualPrn.Name {
						return false
					}
					if prnAND.QName != appdef.NullQName && prnAND.QName != actualPrn.QName {
						ancestorMatched := false
						ancestorQName := iauthnz.QNameAncestor(actualPrn.QName)
						for ancestorQName != appdef.NullQName {
							if ancestorQName == prnAND.QName {
								ancestorMatched = true
								break
							}
							ancestorQName = iauthnz.QNameAncestor(ancestorQName)
						}
						if !ancestorMatched {
							return false
						}
					}
					if prnAND.WSID > 0 && prnAND.WSID != actualPrn.WSID {
						return false
					}
					return true
				}()
				if isMatched {
					break
				}
			}
			if isMatched {
				matchedCount++
			}
		}
		if len(prnsAND) == matchedCount {
			return true
		}
	}
	return false
}

func matchOrNotSpecified_OpKinds(arr []appdef.OperationKind, toFind appdef.OperationKind) bool {
	return len(arr) == 0 || slices.Contains(arr, toFind)
}

func matchOrNotSpecified_QNames(arr []appdef.QName, toFind appdef.QName) bool {
	return len(arr) == 0 || slices.Contains(arr, toFind)
}

func matchOrNotSpecified_Fields(patternFields [][]string, requestedFields []string) bool {
	if len(patternFields) == 0 {
		return true
	}
	if len(requestedFields) == 0 {
		return false
	}
or:
	for _, patternORFields := range patternFields {
		for _, patternANDField := range patternORFields {
			if !slices.Contains(requestedFields, patternANDField) {
				continue or
			}
		}
		return true
	}
	return false
}
