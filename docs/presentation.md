# Building a Custom K8S Controller

A talk by Hrishikesh Shinde ([@hrishike8](https://twitter.com/hrishike8)) walking through this
repository step by step. This is a Markdown transcript of [`presentation.pdf`](../presentation.pdf);
see that file for the original slides (with images/GIFs).

## Controller(s)

Controllers maintain the **desired state** of a resource.

```yaml
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    .....
    spec:
      containers:
      - name: nginx
        image: nginx:1.7.9
```

`spec` → **Controller** → running Pods.

## Types Of Resource(s)

Built-in resources: `Pod`, `Deployment`, `Node`, ...

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 3
  .........
```

## Custom Resource(s)

A Custom Resource (CR) is roughly the equivalent of a database table/document, defined by a
`CustomResourceDefinition`:

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: podsets.demo.k8s.io
spec:
  group: demo.k8s.io
  version: v1alpha1
  names:
    kind: PodSet
```

...and as a programming entity, it's a plain Go struct:

```go
type PodSet struct {
    metav1.TypeMeta
    metav1.ObjectMeta

    Spec   PodSetSpec
    Status PodSetStatus
}
```

An instance of that CRD:

```yaml
apiVersion: demo/v1
kind: PodSet
metadata:
  name: two-podset
spec:
  replicas: 3
```

## Custom Controller(s)

```
CR (PodSet)  +  Controller to manage the CR's state  =  Custom Controller
```

## Use of Custom Controllers?

Extends the Kubernetes platform. Prometheus, Linkerd, cert-manager, Kubernetes itself, and Argo
are all built this way — culminating in the idea of **Operators**.

### Operators

> It's an autopilot for an application, where it extends the Kubernetes API to install, configure,
> scale, or recover application instances.
>
> It bakes application operational knowledge into an application-specific controller.

**Example: Run HA Redis**

- **DIY way** — you assemble Deployments, PVCs, Images, Services yourself, and you need expert
  devops knowledge of Redis HA to get it right.
- **Operator way** — you just declare the outcome you want:

```yaml
apiVersion: databases.spotahome.com/v1
kind: RedisFailover
metadata:
  name: redisfailover-persistent-keep
spec:
  sentinel:
    replicas: 3
    image: redis:4.0-alpine
  redis:
    replicas: 3
    image: redis:4.0-alpine
```

## PodSet Example

This repo's running example: apply a `PodSet` CR, and its controller spins up that many busybox
Pods.

```yaml
# cr.yaml
apiVersion: demo/v1
kind: PodSet
metadata:
  name: two-podset
spec:
  replicas: 3
```

```
$ kubectl apply cr.yaml
```

→ 3 busybox pods.

## Go Lang Dependencies

- `client-go` : `k8s.io/client-go`
- `apimachinery` : `k8s.io/apimachinery`
- `api` : `k8s.io/api`
- `code-generator` : `k8s.io/code-generator`

## Setup

```
git clone git@github.com:hrishin/podset-operator.git \
  $GOPATH/src/github.com/hrishin/podset-operator
```

## Define API Resource

```go
type PodSet struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   PodSetSpec   `json:"spec"`
    Status PodSetStatus `json:"status"`
}
```

## Golang API's ...

...to get, create, delete, watch the `PodSet` resource:

```go
watcher, err := client.DemoV1alpha1().
    PodSets("pods").
    Watch(metav1.ListOptions{})
```

Generated with:

```
~/go/src/k8s.io/code-generator/generate-groups.sh \
  "client" \
  github.com/hrishin/podset-operator/pkg/client \
  github.com/hrishin/podset-operator/pkg/apis \
  demo:v1alpha1
```

## Hit the wall

Running `generate-groups.sh` outside `$GOPATH` fails outright — `code-generator` (at the time of
this talk) required GOPATH-style module resolution:

```
$ ~/go/src/k8s.io/code-generator/generate-groups.sh "deepcopy,client" github.com/hrishin/podset-ope
rator/pkg/apis demo:v1alpha1
Generating deepcopy funcs
F0524 19:44:54.564606 1761 main.go:82] Error: Failed making a parser: unable to add directory "github....
unable to import "github.com/hrishin/podset-operator/pkg/apis/demo/v1alpha1": go/build: importGo it status 1
go: finding module for package github.com/hrishin/podset-operator/pkg/apis/demo/v1alpha1 latest
...
can't load package: package github.com/hrishin/podset-operator/pkg/apis/demo/v1alpha1: unknown import pat
h "...": cannot find module providing package github.com/hrishin/podset-operator/pkg/apis/demo/v1alpha1
```

## Boilerplate code

Define the custom types (CRD Go structs) first:

```
PODSET-OPERATOR
 └─ pkg
     └─ apis
         └─ demo
             └─ v1alpha1
                 ├─ doc.go
                 ├─ register.go
                 └─ types.go
```

## CRD and Clients

```
$ ~/go/src/k8s.io/code-generator/generate-groups.sh "deepcopy,client" \
  github.com/hrishin/podset-operator/pkg/client \
  github.com/hrishin/podset-operator/pkg/apis \
  demo:v1alpha1
Generating deepcopy funcs
Generating clientset for demo:v1alpha1 at github.com/hrishin/podset-operator/pkg/client/clientset
```

...produces the generated clientset (`pkg/client/clientset/versioned/{fake,scheme,typed}`) — in
place now.

## Hello "Watch" Demo

```
git checkout step-1
```

A minimal program that watches `PodSet` resources and prints state changes to the console.

## Controller Flow

The generic control loop:

```
API Server <--Register Event handler / Watch API--> [ Check desired state
                                                        -> Strive for desired state
                                                        -> Update resource status ] --(loop)
```

## PodSet Controller Flow

Applied to `PodSet`:

```
Register Event handler / Watch API (PodSet)
  -> pod = number of pods running
  -> if pods < replicas: spin up new pod
  -> if pods > replicas: delete existing pod
  -> Update resource AvailableReplicas status
  -> (repeat)
```

## Anything Missing?

Watching only the `PodSet` resource isn't enough — if a Pod it owns gets deleted out-of-band
(e.g. `kubectl delete pod`), nothing tells the controller to reconcile again.

## Let's Watch Pods

So watch `Pod` events too (filtered by `label:podset`), resolve the owning `PodSet` for each Pod
event, and reconcile the same way:

```
Register Event handler / Watch API (Pod, label:podset)
  -> get PodSet associated with Pod
  -> pod = number of pods running
  -> if pods < replicas: spin up new pod
  -> if pods > replicas: delete existing pod
  -> Update resource AvailableReplicas status
```

## Functional "PodSet Controller" Demo

```
git checkout step-3
```

## Not enough?

Watching both `Pod` and `PodSet` directly with raw `Watch()` calls works, but doesn't scale well
and re-lists from the API server too often.

## Informers

Introduce a local, cache-backed **Informer**: it watches the API server once and keeps an
in-memory, thread-safe store in sync, so reconcile logic reads from cache instead of hitting the
API server on every event.

## Workqueue

Layer a rate-limited **workqueue** in front of the reconcile logic: event handlers just enqueue a
key (`namespace/name`), and worker goroutines dequeue and process — decoupling event delivery from
processing and giving you retries, de-duplication, and backoff for free.

## client-go controller architecture

*(Credit: Devdutta Kulkarni)*

```
                         client-go
 ┌───────────────────────────────────────────────────────────────────┐
 │  Reflector ──1) List & Watch──────────────► Kubernetes API         │
 │     │ 2) Add Object                          (server-side)         │
 │     ▼                                                              │
 │  Delta FIFO queue                                                  │
 │     │ 3) Pop Object                                                │
 │     ▼                                                               │
 │  Informer ──4) Add Object──► Indexer ──5) Store Object & Key──►    │
 │     │                                          Thread-safe store   │
 │     │ 6) Dispatch Event Handler functions                          │
 │     ▼        (send object to custom controller)                    │
 │  Res Event Handlers reference                                      │
 └───────────────────────────────────────────────────────────────────┘
                         Custom Controller
 ┌───────────────────────────────────────────────────────────────────┐
 │  Resource Event Handlers ──7) Enqueue Object Key──► Workqueue      │
 │                                                          │ 8) Get Key
 │                                                          ▼          │
 │                                                     Process Item    │
 │                                                          │          │
 │  Indexer reference ◄──9) Get Object for Key── Handle Object ◄──────┘
 └───────────────────────────────────────────────────────────────────┘
```

## Generate informers and listers

```
$ ~/go/src/k8s.io/code-generator/generate-groups.sh \
  "deepcopy,client,informer,lister" \
  github.com/hrishin/podset-operator/pkg/client \
  github.com/hrishin/podset-operator/pkg/apis \
  demo:v1alpha1
```

## Final Controller!

```
git checkout step-4
```

The fully functional controller: shared informers, listers, and a workqueue tying it all
together.

## Still issues?

Yes — this is intentionally a from-scratch, educational implementation. See `gotchas.md` for the
rough edges left in on purpose, and the note in the README: *this code is for educational
purposes, with less focus on code quality.*

## Controller: the Unicorn way!

For production, write the same controller using [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder)
or [operator-sdk](https://github.com/operator-framework/operator-sdk) instead — they generate all
of the boilerplate covered in this talk for you.

## Thank you!!
