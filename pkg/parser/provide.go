/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package sqlschema

// TODO: NewFSParser()
// Can be one per app
func ProvideEmbedParser() EmbedParser {
	return embedParserImpl
}

// TODO: NewStringParser
// Can be one per app
func ProvideStringParser() StringParser {
	return stringParserImpl
}
