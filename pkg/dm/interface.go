/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package dm

type IDependencyManager interface {
	DependencyCachePath() string
	ValidateDependencySubDir(depURL, version, subDir string) (depPath string, err error)
	ParseDepQPN(qpn string) (depURL, subDir, depVersion string, err error)
}
