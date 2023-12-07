/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package dm

func NewGoBasedDependencyManager() (IDependencyManager, error) {
	if err := checkGoInstalled(); err != nil {
		return nil, err
	}
	modFile, goModFilePath, err := getGoModFile()
	if err != nil {
		return nil, err
	}
	return &goImpl{
		cachePath:     getCachePath(),
		goModFilePath: goModFilePath,
		modFile:       modFile,
	}, nil
}
