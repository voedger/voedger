/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package appdef

// Finds reference field by name.
//
// Returns nil if field is not found, or field found but it is not a reference field
func RefField(ff IFields, name FieldName) IRefField {
	if fld := ff.Field(name); fld != nil {
		if fld.DataKind() == DataKind_RecordID {
			if fld, ok := fld.(IRefField); ok {
				return fld
			}
		}
	}
	return nil
}
