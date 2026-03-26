# AIR-3414: Eliminate vit.MockBuckets

**Source:** [AIR-3414](https://untill.atlassian.net/browse/AIR-3414)

Eliminate `vit.MockBuckets` and replace the corresponding integration test with additional HTTP requests combined with time advancement between them to force real bucket depletion, rather than relying on mock behavior.

