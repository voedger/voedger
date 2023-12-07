/*
 * Copyright (c) 2023-present unTill Pro, Ltd.
 * @author Alisher Nurmanov
 */

package dm

type IDependencyManager interface {
	// ModulePath returns module path, which is the module’s unique identifier (when combined with the module version number).
	// The module path becomes the import prefix for all packages the module contains.
	ModulePath() string
	// LocalPath returns path to local dependency path.
	// Implementation should download dependency if it is not downloaded yet.
	// E.g. github.com/voedger/voedger/pkg/sys => /home/user/go/pkg/mod/github.com/voedger/voedger@v0.0.0-20231103100658-8d2fb878c2f9/pkg/sys
	LocalPath(depURL string) (localDepPath string, err error)
	// CachePath returns path to dependency cache
	CachePath() string
}
