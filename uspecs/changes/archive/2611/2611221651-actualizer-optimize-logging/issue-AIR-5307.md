# voedger, acutalizer: optimize logging

- URL: https://untill.atlassian.net/browse/AIR-5307
- ID: AIR-5307
- State: In progress
- Author: Denis Gribanov
- Labels: none
- Assignees: Denis Gribanov
- Linked issues: AIR-4959 (parent: voedger: partition recovery performance)

## Why

if the new projector is created then it actualizer runs it through the whole log starting from the FirstOffset. On each event the actualizer logs details about each event that fires the projector like:  

```
time=2026-11-18T10:23:12.838Z level=DEBUG msg="args={\"AirCluster\":\"dev\",\"AirSubscriptionVersion\":0,\"BillingPeriod\":\"m\",\"CancelScheduleCreated\":false,\"CancelledAt\":0,\"CardResourceVersion\":1731572789948,\"Currency\":\"EUR\",\"Fiscalization\":false,\"HospitalityExtra\":false,\"LastFour\":\"1111\",\"NextBillingAt\":1739487600000,\"NumberOfDevices\":2,\"PaymentMethod\":\"visa\",\"Price\":5929,\"Printing\":false,\"Status\":\"active\",\"SubscriptionID\":\"dev_140737488487951\",\"SubscriptionResourceVersion\":1736809334936,\"TrialEnd\":1731538800,\"sys.QName\":\"air.StoreSubscriptionProfileParams\"}, triggeredby=air.StoreSubscriptionProfile" src=pipeline.(*WiredOperator).doAsync:90 stage=ap woffset=48 poffset=5654 evqname=air.StoreSubscriptionProfile wsid=140737488487951 vapp=untill/airs-bp extension=ap.sys.ApplyRecoveryCheckpoint
```

that wastes much place in the grafana logs if the log is big. 

## What

* Actualizer runs the new projector through the whole log → do not log details with “triggeredby” at all. Let the new projector run silently
* eliminate `args` from msg. That is required for human. But all investigations will be made by AI agent in future so let’s avoid wasting place in the docker container logs
