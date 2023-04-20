/*
 * Copyright (c) 2021-present Sigma-Soft, Ltd.
 * @author: Nikolay Nikitin
 */

package consts

// system views enumeration
const (
	SysView_Versions     uint16 = 16 + iota // system view versions
	SysView_QNames                          // application QNames system view
	SysView_Containers                      // application container names view
	SysView_Records                         // application Records view
	SysView_PLog                            // application PLog view
	SysView_WLog                            // application WLog view
	SysView_SingletonIDs                    // application singletons IDs view
)
