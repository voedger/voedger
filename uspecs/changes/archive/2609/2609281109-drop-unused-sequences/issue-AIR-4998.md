# voedger: drop unused sequences feature

- URL: https://untill.atlassian.net/browse/AIR-4998
- ID: AIR-4998
- State: in-progress
- Author: Denis Gribanov
- Labels: none
- Assignees: Denis Gribanov
- Linked issues: [AIR-4959: voedger: partition recovery performance](https://untill.atlassian.net/browse/AIR-4959) (parent)

## Description

* preserve pkey prefixes
* do not touch anything related to Sequences Trust Level
* mark `pKeyPrefix_SeqStorage_Part` and `pKeyPrefix_SeqStorage_WS` as deprecated, put an appropriate comment
* do not touch any documentation, it will be cleaned up in <custom data-type="smartlink" data-id="id-0">https://untill.atlassian.net/browse/AIR-4980</custom> 

