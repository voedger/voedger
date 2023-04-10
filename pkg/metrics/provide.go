/*
 * Copyright (c) 2021-present unTill Pro, Ltd.
*
* @author Michael Saigachenko
*/

package imetrics

// Provide s.e.
func Provide() IMetrics {
	return newMetrics()
}
