/*
* Copyright (c) 2023-present unTill Pro, Ltd.
* @author Michael Saigachenko
 */
package sqlschema

func ProvideEmbedParser() EmbedParser {
	return embedParserImpl
}

func ProvideStringParser() StringParser {
	return stringParserImpl
}
