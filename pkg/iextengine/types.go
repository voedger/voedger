/*
  - Copyright (c) 2023-present unTill Software Development Group B. V.
    @author Michael Saigachenko
*/
package iextengine

func NewExtQName(packageName, extName string) ExtQName {
	return ExtQName{packageName, extName}
}
