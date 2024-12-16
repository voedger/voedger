/*
 * Copyright (c) 2024-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package descr

type Role struct {
	Type
	ACL *ACL `json:",omitempty"`
}
