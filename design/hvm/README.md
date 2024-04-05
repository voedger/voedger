# Functional design
```go
pkgSys := DefinePackage().
	WithSchemas(...),
	WithFuncs(WithFunc(
		QName,
		...
		WithRate(...),
	))
appRegistry := DefineApp(appQName, WithPackages(
	pkgSys,
))
appAirsBP := DefineApp(appQName, WithPackages(
	pkgSys,
	pkgAirsBP,
	pkgUnTill,
))
ProvideCluster(WithApps(
	appAirsBP,
	appRegistry,
))
type Cluster struct {
	ServicePipeline
	VVMAPI

}
```
# Technical design

```mermaid
erDiagram

	VVMConfig ||..|| AppPartitionConfig: has
	VVMConfig ||..|| VVM: "used to build"
	VVMConfig ||..|| os_Args: "built using e.g."
	VVM ||..|{ AppPartition: contains

	AppPartitionConfig {
		CommandProcessorsAmount int
		QueryProcessorAmount int
		Low_Medium_High_rates irates_BucketState
	}

	AppPartition {
		Number int
		Buckets irates_IBucket
		ServiceChannelFactory func_serviceChannelType_serviceChannelNum
	}
```

```mermaid
erDiagram

	VVMConfig ||..|| DefaultAppPartitionConfig: has
	VVMConfig ||..|| VVM: "used to build"
	VVMConfig ||..|| os_Args: "built using e.g."
	VVM ||..|{ AppPartition: contains

	App{

	}

	DefaultAppPartitionConfig {
		CommandProcessorsAmount int
		QueryProcessorAmount int
		func_serviceChannelType_serviceChannelNum ServiceChannelFactory
	}

	AppPartition {
		Number int
		Buckets irates_IBucket
		CommandProcessors IOperator
		QueryPrcoessors IOperator
		AppStructs IAppStructs
	}
```

```mermaid
erDiagram

VVM ||--|| ServicePipeline: has
VVM ||--|| VVMAPI: "has struct of interfaces"
VVM ||--|{ IVVMApp: "has per app"
VVM ||--|| MetricsServicePort: "has func"
VVM ||..|| struct: is

ServicePipeline ||..|| ISyncPipeline: is
ServicePipeline ||--|| ForkOperator_SP : "has single"

ForkOperator_SP ||--|| ForkOperator_Processors: "has branch"
ForkOperator_SP ||--|| ServiceOperator_Router: "has branch"
ForkOperator_SP ||--|| ServiceOperator_MetricsHTTPService: "has branch "

QueryProcessorsCount ||..|| x10_q: "hardcoded"
QueryProcessorsCount ||..|| QueryProcessors: "defines branches amount of"
ForkOperator_Processors ||--|| QueryProcessors: "has branch"
ForkOperator_Processors ||--|| AppServices: "has branch"
ForkOperator_Processors ||--|| CommandProcessors: "has branch"

QueryProcessors ||..|| ForkOperator_QP: is
QueryProcessors ||--|{ QueryProcessor: "has branches"


QueryProcessor ||..|| IService_QP: is
QueryProcessor ||..|| ServiceOperator_QP: is
QueryProcessor }|--|| QueryChannel: "sits on same"
QueryProcessor }|--|| IAppStructsProvider: "has same"
QueryChannel ||..|| ubuffered: is

CommandProcessorsCount ||..|| CommandProcessors: "branches amount is"
CommandProcessors ||..|| ForkOperator_CP: is
CommandProcessors ||--|{ CommandProcessor: "has branches"
CommandProcessor ||--|| CommandChannel: "sits on"
CommandProcessor ||..|| IService_CP: is
CommandProcessor ||..|| ServiceOperator_CP: is
CommandChannel ||..|| PartitionID: per
CommandChannel ||..|| CommandProcessorsCount: "has buffer size equal to"
CommandProcessorsCount ||..|| PartitionsCount: "equals to"
PartitionsCount ||..|| PartitionID: "defines amount of"
CommandProcessor ||--|| PartitionID: per
CommandProcessor }|--|| IAppStructsProvider: "has same"
CommandProcessorsCount ||..|| x10_c: "hardcoded"


```