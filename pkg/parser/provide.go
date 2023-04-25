/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package sqlschema

// Can be one per app
func NewFSParser() FSParser {
	return embedParserImpl
}

// Can be one per app
func NewStringParser() StringParser {
	return stringParserImpl
}
