# voedger: use pull_request triggering event instead of pull_request_target

- URL: https://untill.atlassian.net/browse/AIR-4609
- ID: AIR-4609
- State: In Progress⚒️
- Author: Denis Gribanov
- Labels: none
- Assignees: Denis Gribanov
- Parent: AIR-4608

## Why

`pull_request_target` was needed to auto-merge the PR. Now auto-merge is disable to it is better to use more safe `pull_request`

Now we have error on pull request from fork:

```
Error: Refusing to check out fork pull request code from a 'pull_request_target' workflow. This workflow runs with the base repository's GITHUB_TOKEN, secrets, default-branch cache scope, and runner access. Fetching and executing a fork's code in that trusted context commonly leads to "pwn request" vulnerabilities. To opt in, review the risks at https://gh.io/securely-using-pull_request_target and set 'allow-unsafe-pr-checkout: true' on the actions/checkout step.
```

## What

* use `pull_request` instead of `pull_request_tagret`
* investigate what else should be changed after `pull_request` change

