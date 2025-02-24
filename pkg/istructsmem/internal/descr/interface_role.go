/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

type Role struct {
	Type

	// #3335: is role published
	Published bool `json:",omitempty,omitzero"`
}
