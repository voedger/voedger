/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package dm

func NewGoBasedDependencyManager(wd string) (IDependencyManager, error) {
	if err := checkGoInstalled(); err != nil {
		return nil, err
	}
	packageCachePath, err := locatePackageCache()
	if err != nil {
		return nil, err
	}
	dependencyFilePath, err := locateDependencyFile(wd)
	if err != nil {
		return nil, err
	}
	return &goBasedDependencyManager{
		dependencyCachePath: packageCachePath,
		dependencyFilePath:  dependencyFilePath,
	}, nil
}
