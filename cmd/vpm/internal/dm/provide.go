/*
 * Copyright (c) 2020-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package dm

func NewGoBasedDependencyManager() (IDependencyManager, error) {
	if err := checkGoInstalled(); err != nil {
		return nil, err
	}
	cachePath, err := getCachePath()
	if err != nil {
		return nil, err
	}
	modFile, goModFilePath, err := getGoModFile()
	if err != nil {
		return nil, err
	}
	return &goImpl{
		cachePath:     cachePath,
		goModFilePath: goModFilePath,
		modFile:       modFile,
	}, nil
}
