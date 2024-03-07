/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Function is a type of extension that can take params and return value.
//
// Function may be query or command.
type IFunction interface {
	IExtension

	// Parameter. Returns nil if not assigned
	Param() IType

	// Result. Returns nil if not assigned
	Result() IType
}

type IFunctionBuilder interface {
	IExtensionBuilder

	// Sets function parameter. Must be known type from next kinds:
	//	 - Data
	//	 - ODoc
	//	 - Object
	//
	// If NullQName passed then it means that function has no parameter.
	// If QNameANY passed then it means that parameter may be any.
	SetParam(QName) IFunctionBuilder

	// Sets function result. Must be known type from next kinds:
	//	 - Data
	//	 - GDoc,  CDoc, WDoc, ODoc
	//	 - Object
	//
	// If NullQName passed then it means that function has no result.
	// If QNameANY passed then it means that result may be any.
	SetResult(QName) IFunctionBuilder
}
