### Abstract

How to deploy Heeus application into some Heeus federation

### Principles

- No intermediate package managers like artifactory, jfrog etc.
- Application is deployed using Application Images (AppImage) + Deployment Descriptor (Deployment)
- AppImage presents on every server node (is downloaded by Agent)
- Application is executed by Application Partitions

### Best practices

- [Swarm](10-review-existing-1-swarm.md)
- [Nomad](10-review-existing-2-nomad.md)
- [K8s](10-review-existing-3-k8s.md)
- [Google: Borg, Omega, and Kubernetes](google/README.md)

### Concepts

- [Deployment Artifacts](#deployment-artifacts)
- [Deployment](#deployment)
- [Agents](#agents)

### Detailed design

- [Application Partition Deployment](app-deployment-2.md)

### Deployment Artifacts
- Deployment (Deployment Descriptor, Дескриптор развертывания)
- AppImage

```mermaid
erDiagram
  Deployment ||..|| AppImage : "has a reference to"
  AppImage ||..|{ Package : "has a copy of"
  Package |{..|| Module: "is kept in"
  Module |{..|| Repository: "is kept in"

```

### Job

```mermaid
erDiagram
  Job ||..|{ Group : "has"
  Group ||..|{ Task : "has"
  Group |{..|| Host : "tasks are placed (colocated)  on the same"

  Group{
    numReplicas int "number of desired replicas"
  }

  Task{
    driver string  "AppPartition, docker, exec, java"
    config map "driver-specific, image = 'hashicorp/web-frontend'"
    env map "DB_HOST = 'db01.example.com'"
    resources map "cpu  = 500 #MHz, memory = 1024 #MB"
  }
```
### Agents
```mermaid
erDiagram
  Job ||..|{ Group : "has"
  Group ||..|{ Task : "has"
  Group ||..|{ Replica : "scheduled to"
  Replica || .. |{ Scheduler : "scheduled by"
  Replica |{ .. || Agent: "executed by"
  Replica |{ .. || Host: "scheduled to"
  Agent || .. || HostAgent: "can be"
  Agent || .. || VVMAgent: "can be"
  Host ||..|| HostAgent: "runs one"
  Host ||..|| VVM: "runs one"
  VVM ||..|| VVMAgent: "has one"
  HostAgent ||..|{ Executable: "controls"
  HostAgent ||..|{ Container: "controls"
  VVMAgent ||..|{ AppPartition: "controls"
  Executable ||..|| VVM : "can be"
```
