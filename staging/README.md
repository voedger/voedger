# External Repository Staging Area

This directory is the staging area for packages that have been split to their own repository. The content here will be periodically published to respective top-level github.com/voedger repositories.

Repositories currently staged here:
- [`github.com/voedger/exttinygo`](https://github.com/voedger/exttinygo)

# Unstage assuming there were number of commits with modifications of a staged repo
- install [git filter-repo](https://github.com/newren/git-filter-repo/)
  - install python3
  - place [the file](https://github.com/newren/git-filter-repo/blob/main/git-filter-repo) at `C:\Program Files\Git\mingw64\libexec\git-core`
- backup voedger working copy
- clone voedger in a new dir
- in new voedger working copy root:
  - WARNING!!! the history of local voedger repo will be destroyed forever ever after the next command!!!
  - `git filter-repo --subdirectory-filter staging/src/github.com/untillpro/ibusmem`
    - the repo will contain only `staging/src/github.com/untillpro/ibusmem`
	- history of the local repo will consists of modifications of `...ibusmem` only
  - `git remote add lib ../ibusmem`
    - target repo working copy here
  - `git push lib main:merge-staging`
- rebase\merge from branch `merge-staging` in target repo working copy
- delete voedger working copy
