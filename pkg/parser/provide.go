/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package sqlschema

// TODO: NewFSParser()
func ProvideEmbedParser() EmbedParser {
	return embedParserImpl
}

// TODO: NewStringParser()
func ProvideStringParser() StringParser {
	return stringParserImpl
}
