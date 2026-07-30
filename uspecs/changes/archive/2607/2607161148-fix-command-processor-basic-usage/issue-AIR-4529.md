# Fix command processor TestBasicUsage

- URL: https://untill.atlassian.net/browse/AIR-4529
- ID: AIR-4529
- State: In Progress
- Author: Denis Gribanov
- Labels: none
- Assignees: Denis Gribanov
- Linked issues: AIR-4515 (is duplicated by)

## Description

<custom data-type="smartlink" data-id="id-0">https://github.com/voedger/voedger/actions/runs/29143933977/job/86522107439</custom> 

```
--- FAIL: TestBasicUsage (0.01s)
      impl_test.go:154: no log line contains all of the expected substrings
          log:
          time=2026-07-11T07:09:49.597Z level=INFO msg="" src=in10nmem.notifier:304 stage=n10n.notifier.start vapp=sys/voedger extension=sys._N10NBroker
          time=2026-07-11T07:09:49.597Z level=INFO msg="Heartbeat30Duration: 30s" src=in10nmem.(*N10nBroker).heartbeat30:422 stage=n10n.heartbeat.start vapp=sys/voedger extension=sys._N10NBroker
          time=2026-07-11T07:09:49.601Z level=INFO msg="" src=command.(*cmdProc).recovery:290 stage=cp.partition_recovery.start vapp=sys/voedger extension=sys._Recovery partid=1
          time=2026-07-11T07:09:49.601Z level=DEBUG msg="args={}" src=command.(*cmdProc).recovery:328 stage=cp.partition_recovery.reapply woffset=1 poffset=2 evqname=sys.CUD vapp=sys/voedger extension=sys._Recovery partid=1
          time=2026-07-11T07:09:49.601Z level=DEBUG msg="newfields={\"CreatedAtMs\":1783753789600,\"InitCompletedAtMs\":1783753789600,\"Status\":0,\"WSKind\":\"sys.TestWSKind\",\"WSName\":\"stub workspace\",\"sys.ID\":65537,\"sys.IsActive\":true,\"sys.QName\":\"sys.WorkspaceDescriptor\"}" src=command.(*cmdProc).recovery:328 stage=cp.partition_recovery.reapply.log_cud rectype=sys.WorkspaceDescriptor recid=65537 op=create woffset=1 poffset=2 evqname=sys.CUD vapp=sys/voedger extension=sys._Recovery partid=1
          time=2026-07-11T07:09:49.601Z level=DEBUG msg="" src=command.setUp.setUp.ProvideServiceFactory.func5.func6.2:82 stage=sp.success woffset=1 poffset=2 evqname=sys.CUD partid=1 vapp=sys/voedger extension=sys._Recovery
          time=2026-07-11T07:09:49.602Z level=INFO msg="completed, nextPLogOffset 3, workspaces {\"1\":{\"NextWLogOffset\":2},\"2\":{\"NextWLogOffset\":2}}" src=command.(*cmdProc).recovery:356 stage=cp.partition_recovery.complete extension=sys._Recovery partid=1 vapp=sys/voedger
          07/11 07:09:49.602: ---: [oldacl.IsOperationAllowed:30]: old-style OperationKind_Execute sys.Test for [Role sys.AuthenticatedUser,WSID 1;Role sys.System,WSID 1;Host ]:  -> deny
          time=2026-07-11T07:09:49.602Z level=DEBUG msg="args={\"Text\":\"hello\",\"sys.QName\":\"sys.TestParams\"}" src=command.logEventAndCUDs:384 stage=cp.plog_saved woffset=2 poffset=3 evqname=sys.Test
          time=2026-07-11T07:09:49.602Z level=DEBUG msg="" src=command.setUp.setUp.ProvideServiceFactory.func5.func6.2:82 stage=sp.success woffset=2 poffset=3 evqname=sys.Test
      --- FAIL: TestBasicUsage/basic_usage (0.00s)
```
