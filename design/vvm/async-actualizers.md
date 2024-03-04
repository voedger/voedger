# Design partition + async actualizers 

https://github.com/voedger/voedger/issues/1464

## Current design

- [vvm.provideAsyncActualizersFactory()](https://github.com/voedger/voedger/blob/43469aef2ed4878dfa3cb7dca304c87350547a8d/pkg/vvm/wire_gen.go#L518)
- vvm.provideOperatorAppServices()
  - forks appServices per apps
  - [appsAmount]appServices

```mermaid
graph

%% Entities ====================

pipeline.ISyncOperator
pipeline.ISyncOperator2["pipeline.ISyncOperator"]
pipeline.ISyncOperator3["pipeline.ISyncOperator"]

state.ActualizerStateOptFunc

istructs.QName
istructs.AppQName
istructs.PartitionID
istructs.IAppStructsProvider
istructs.ProjectorFactory(["istructs.ProjectorFactory()"])
istructs.Projector
istructs.ProjectorFunc

projectors.SyncActualizerConf
projectors.AppStructsFunc(["projectors.AppStructsFunc()"])
projectors.AsyncActualizerFactory(["projectors.AsyncActualizerFactory()"])
projectors.ProvideAsyncActualizerFactory(["projectors.ProvideAsyncActualizerFactory()"])

appparts.IAppPartitions

vvm.provideAsyncActualizersFactory(["vvm.provideAsyncActualizersFactory()"]):::S
vvm.provideAppPartitionFactory(["vvm.provideAppPartitionFactory()"])
vvm.provideAppServiceFactory(["vvm.provideAppServiceFactory()"])
vvm.provideOperatorAppServices(["vvm.provideOperatorAppServices()"])
vvm.provideServicePipeline(["vvm.provideServicePipeline()"])
vvm.AsyncActualizersFactory(["vvm.AsyncActualizersFactory()"])
vvm.AsyncProjectorFactories["[]vvm.AsyncProjectorFactories"]
vvm.AppPartitionFactory(["vvm.AppPartitionFactory()"])

%% Relations ====================

istructs.ProjectorFunc -..- istructs.Projector
istructs.QName -.- istructs.Projector

istructs.PartitionID -..- istructs.ProjectorFactory
projectors.AppStructsFunc -.- projectors.SyncActualizerConf
istructs.PartitionID -.- projectors.SyncActualizerConf
projectors.ProvideAsyncActualizerFactory --> projectors.AsyncActualizerFactory
projectors.SyncActualizerConf -.- projectors.AsyncActualizerFactory
istructs.ProjectorFactory -.- projectors.AsyncActualizerFactory

istructs.ProjectorFactory --> istructs.Projector


istructs.IAppStructsProvider -.- vvm.provideAsyncActualizersFactory
appparts.IAppPartitions -.- vvm.provideAsyncActualizersFactory
projectors.AsyncActualizerFactory -.- vvm.provideAsyncActualizersFactory
projectors.AsyncActualizerFactory --> pipeline.ISyncOperator3


state.ActualizerStateOptFunc -..- |"[]"| vvm.provideAppPartitionFactory
vvm.provideAsyncActualizersFactory ----> vvm.AsyncActualizersFactory
vvm.provideAppPartitionFactory --> vvm.AppPartitionFactory
istructs.AppQName -.- vvm.AppPartitionFactory
vvm.AsyncProjectorFactories -.- vvm.AppPartitionFactory


vvm.AppPartitionFactory -.- vvm.provideAppServiceFactory
vvm.AppPartitionFactory --> pipeline.ISyncOperator2


vvm.provideAppServiceFactory -.- vvm.provideOperatorAppServices
vvm.provideOperatorAppServices -.- vvm.provideServicePipeline

vvm.AsyncActualizersFactory --> pipeline.ISyncOperator
vvm.AsyncActualizersFactory -.- vvm.provideAppPartitionFactory

istructs.ProjectorFactory -.- vvm.AsyncProjectorFactories

istructs.AppQName -..- vvm.AsyncActualizersFactory
vvm.AsyncProjectorFactories -.- vvm.AsyncActualizersFactory
istructs.PartitionID -.- vvm.AsyncActualizersFactory

classDef S fill:#B5FFFF
```

## Analysis

- ??? projectors.SyncActualizerConf - ProjectorFactory

## Proposal

