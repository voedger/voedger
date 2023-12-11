/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package dm

func NewGoBasedDependencyManager(workingDir string) (IDependencyManager, error) {
	if err := checkGoInstalled(); err != nil {
		return nil, err
	}
	modFile, goModFilePath, err := getGoModFile(workingDir)
	if err != nil {
		return nil, err
	}
	return &goImpl{
		cachePath:     getCachePath(),
		goModFilePath: goModFilePath,
		modFile:       modFile,
	}, nil
}
