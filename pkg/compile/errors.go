package compile

import "errors"

var (
	ErrAppSchemaNotFound = errors.New("package does not have an APPLICATION statement")
)
