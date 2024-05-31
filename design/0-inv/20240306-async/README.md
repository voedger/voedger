# IAppPartitions + async actualizers 

https://github.com/voedger/voedger/issues/1464

## Context

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
appdef.AppQName
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
istructs.ProjectorFactory -....- projectors.AsyncActualizerFactory

istructs.ProjectorFactory --> istructs.Projector


istructs.IAppStructsProvider -.- vvm.provideAsyncActualizersFactory
appparts.IAppPartitions -.- vvm.provideAsyncActualizersFactory
projectors.AsyncActualizerFactory -.- vvm.provideAsyncActualizersFactory
projectors.AsyncActualizerFactory --> pipeline.ISyncOperator3

state.ActualizerStateOptFunc -..- |"[]"| vvm.provideAppPartitionFactory
vvm.provideAsyncActualizersFactory ----> vvm.AsyncActualizersFactory
vvm.provideAppPartitionFactory --> vvm.AppPartitionFactory
appdef.AppQName -.- vvm.AppPartitionFactory
vvm.AsyncProjectorFactories -.- vvm.AppPartitionFactory

vvm.AppPartitionFactory -.- vvm.provideAppServiceFactory
vvm.AppPartitionFactory --> pipeline.ISyncOperator2


vvm.provideAppServiceFactory -.- vvm.provideOperatorAppServices
vvm.provideOperatorAppServices -.- vvm.provideServicePipeline

vvm.AsyncActualizersFactory --> pipeline.ISyncOperator
vvm.AsyncActualizersFactory -.- vvm.provideAppPartitionFactory

istructs.ProjectorFactory -.- vvm.AsyncProjectorFactories

appdef.AppQName -..- vvm.AsyncActualizersFactory
vvm.AsyncProjectorFactories -.- vvm.AsyncActualizersFactory
istructs.PartitionID -.- vvm.AsyncActualizersFactory

classDef S fill:#B5FFFF
```

## Analysis


- istructsmem.AppConfigType
```go
type AppConfigType struct {
  ...
	syncProjectorFactories  []istructs.ProjectorFactory
	asyncProjectorFactories []istructs.ProjectorFactory
	cudValidators           []istructs.CUDValidator
	eventValidators         []istructs.EventValidator
  ...
}
```  
- iextengine.IExtensionEngineFactories: `map[appdef.ExtensionEngineKind]IExtensionEngineFactory`
- iextengine.IExtensionEngineFactory: `New(ctx context.Context, packages []ExtensionPackage, config *ExtEngineConfig, numEngines int) ([]IExtensionEngine, error)`
- iextengine.ExtensionPackage
```go
type ExtensionPackage struct {
	QualifiedName  string
	ModuleUrl      *url.URL
	ExtensionNames []string
}

type IExtensionIO interface {
	istructs.IState
	istructs.IIntents
	istructs.IPkgNameResolver
}

type IExtensionEngine interface {
	SetLimits(limits ExtensionLimits)
	Invoke(ctx context.Context, extName ExtQName, io IExtensionIO) (err error)
	Close(ctx context.Context)
}
```
- istructs.Projector: `Func func(event IPLogEvent, state IState, intents IIntents) (err error)`
- Projector example: invite.provideAsyncProjectorApplyCancelAcceptedInviteFactory

## Proposal

### Projectors shall be accessible through ExtEngineKind_BuiltIn

- `istructs.IState.PLogEvent() IPLogEvent`
- state: sync projectors: Put event
- state: async projectors: Put event
- IAppPartition(s)
```go
type IAppPartitions interface {
	// Adds new application or update existing.
	//
	// If application with the same name exists, then its definition will be updated.
	DeployApp(name appdef.AppQName, def appdef.IAppDef, perPartitionEngines [cluster.ProcessorKind]int, numPartitions int)

```
- Use `IAppPartition.Invoke` in async actualizer
  - Wire: Register all projectors in `iextenginebuiltin.ProvideExtensionEngineFactory`
    - Wrapper around istructs.Projector that gets IPLogEvent and passes to istructs.Projector
  - `appparts.NewWithEngines(engfacts iextengine.IExtensionEngineFactories)`
  - IAppPartition.Invoke
```go
type IAppPartition interface {
	Invoke(qname istructs.QName, state istructs.IState, intents istructs.IIntents) (err error)
}  	
```
  - Use `Invoke()` in async actualizer
  - Use `Invoke()` in `IAppPartition.DoSyncActualizer`

### Migrate to Actualizers Processors

- ???
