### Kubernetes.Pod

- https://cloud.google.com/kubernetes-engine/docs/concepts/pod
- https://kubernetes.io/docs/concepts/workloads/pods/

```mermaid
erDiagram
  Pod ||..|{ Container : "contains"
  Pod ||..|{ Resource : "describes"
  Container ||..|{ Resource : "shares"
  Pod ||..|| SmallestDeployableUnit : "is"
  Pod ||..|| Deployment: "'s deployment described by"
  Resource ||..|{ Storage : "can be"
  Resource ||..|{ Network : "can be"
  Resource ||..|{ CPU : "can be"
  Resource ||..|{ Memory : "can be"
```

### Kubernetes.Deployment

```mermaid
erDiagram
  Deployment ||..|{ DeploymentSpec : "has spec"
  Deployment ||..|{ ObjectMeta : "has metadata"
  Deployment ||..|{ DeploymentStatus : "has status"
  DeploymentSpec ||..|| PodTemplateSpec : "has template"
  PodTemplateSpec ||..|| PodSpec : "has spec"
  PodSpec ||..|{ ContainerSpec : "has containers []"
```

>A Deployment provides declarative updates for Pods and ReplicaSets.
> You describe a desired state in a Deployment, and the Deployment Controller changes the actual state to the desired state at a controlled rate. You can define Deployments to create new ReplicaSets, or to remove existing Deployments and adopt all their resources with new Deployments.


[Creating a multiple pod Deployment](https://www.reddit.com/r/kubernetes/comments/ci4297/creating_a_multiple_pod_deployment/)
- I want to have a single deployment file that needs to run my containers in different pods. Right now, the deployment is creating a single pod with two containers inside. Is there a way i could deploy two pods, with different containers from the same deployment file?
- Deployment only support a single pod, a pod can have multiple containers
  - But you can put multiple deployments in a single file, use --- to separate multiple documents




#### Deployment

- [OpenAPI.Deployment](https://elements-demo.stoplight.io/?spec=https://raw.githubusercontent.com/kubernetes/kubernetes/master/api/openapi-spec/swagger.json#/schemas/io.k8s.api.apps.v1.Deployment)
- [Kubernetes Deployment spec example](https://www.tutorialworks.com/kubernetes-deployment-spec-examples/):
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-rolling
  labels:
    app: nginx-rolling
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx-rolling
  strategy:
    type: RollingUpdate   # Upgrade this application with a rolling strategy
    rollingUpdate:
      maxSurge: 1         # maximum number of pods that can be scheduled
                          # above the desired number of pods (replicas)
      maxUnavailable: 0   # the maximum number of pods that can be unavailable
                          # during the update
  template:
    metadata:
      labels:
        app: nginx-rolling
    spec:
      containers:
      - image: nginx
        name: nginx
        ports:
        - containerPort: 8080
```       


#### ObjectMeta

ObjectMeta is metadata that all persisted resources must have, which includes all objects users must create.

```mermaid
erDiagram
metadata{
  generation int64  "A sequence number representing a specific generation of the desired state."
  deletionGracePeriodSeconds int64
  labels map_string_string
  name string "Name must be unique within a namespace"
  ownerReferences arr-of-object "List of objects depended by this object"
  resourceVersion string
  uid string "UID is the unique in time and space value for this object"
}
```
- ownerReferences. List of objects depended by this object. If ALL objects in the list have been deleted, this object will be garbage collected. If this object is managed by a controller, then an entry in this list will point to this controller, with the controller field set to true. There cannot be more than one managing controller.
- resourceVersion. An opaque value that represents the internal version of this object that can be used by clients to determine when objects have changed.

#### DeploymentSpec
```mermaid
erDiagram
DeploymentSpec {
  strategy object "how to replace existing pods with new ones"
  selector object "A label selector is a label query over a set of resources. "
  replicas int32 "Number of desired pods"  
  template PodTemplateSpec "describes the data a pod should have when created from a template"
}
```
#### PodTemplateSpec

```mermaid
erDiagram
PodTemplateSpec  {
  metadata  ObjectMeta 
  spec      PodSpec "PodSpec is a description of a pod"
}  
```  

#### PodSpec
```mermaid
erDiagram
PodTemplateSpec  {
  metadata  object
  affinity  Affinity
  containers arr_of_Container
  hostname string
  initContainers arr_of_Container
  nodeName string "request to schedule this pod onto a specific node"
}  
```  

- [Resource Management for Pods and Containers](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: frontend
spec:
  containers:
  - name: app
    image: images.my-company.example/app:v4
    resources:
      requests:
        memory: "64Mi"
        cpu: "250m"
      limits:
        memory: "128Mi"
        cpu: "500m"
  - name: log-aggregator
    image: images.my-company.example/log-aggregator:v6
    resources:
      requests:
        memory: "64Mi"
        cpu: "250m"
      limits:
        memory: "128Mi"
        cpu: "500m"
```        
#### Deployment.status (DeploymentStatus)

```mermaid
erDiagram
status{
  availableReplicas   int32 "The number of available replicas (ready for at least minReadySeconds) for this deployment."
  observedGeneration  int64 "The generation observed by the deployment controller"
  readyReplicas       int32 "The number of ready replicas (ready for at least minReadySeconds) for this deployment"
  replicas            int32 "Total number of non-terminated pods targeted by this deployment (their labels match the selector)"
  updatedReplicas     int32 "The number of replicas that have a desired version matching the deployment's version"
}
```

