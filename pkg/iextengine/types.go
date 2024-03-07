/*
  - Copyright (c) 2023-present unTill Software Development Group B.V.
    @author Michael Saigachenko
*/
package iextengine

func NewExtQName(packagePath, extName string) ExtQName {
	return ExtQName{packagePath, extName}
}
