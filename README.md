## PodSet Operator

Objective of this Operator/Controller is demonstrate `ReplicaSet` kind of resource
implementation using Kubernetes controller pattern.
Another objective of this repo. is to show how to build the controller from scratch and what challenges a developer could face.
So that a developer could understand the beauty [KuberBuilder](https://github.com/kubernetes-sigs/kubebuilder) or [Operator SDK](https://github.com/operator-framework/operator-sdk) frameworks


#### PodSet resource
Once user applies the `PodSet` (`kubectl apply -f podset.yaml`) resource, controller could spin up
number of pods mentioned as per `replicas` filed.

e.g. User want to spin up 3 pod

```yaml
apiVersion: demo.k8s.io/v1alpha1
kind: PodSet
metadata:
  name: three-podset
spec:
  replicas: 3
```

### Prerequisites

* Kubernetes cluster 1.9 + (minikube also works)
* golang 1.11 +
* set `GO111MODULE="on"` env if source code is in `$GOPATH`

### Presentation

[Presentation deck](/presentation.pdf)

### Tutorial
Check out the code according to following instruction and check the README file to follow the further instructions.

#### step 1
```
git checkout step-1
```
covers basic code and scaffolding setup

#### step 2
```
git checkout step-2
```
Covers how to define CRD types, register CRD's and generate client API's using `go-client` and `generators`
It covers simple program that issues `watch` request to `PodSet` resource and print's resource state changes on console.

#### step 3
```
git checkout step-3
```
Covers functional controller using basic generated code. It shows how to issue watch requests and bring `PodSet` resource to desired state (reconciliation).

#### step 4
```
git checkout step-4
```
Covers fully functional controller using shared informers, listers and workqueues. It shows how to generate all those objects.
At this point one could able to relate why controllers are written in particular way.

***Note: This code is intended for educational purpose. While less focus is given on code quality aspect.***

### Credits
- [https://github.com/kubernetes/sample-controller](https://github.com/kubernetes/sample-controller)
- [programming kubernetes by Stefan Schimanski, Michael Hausenblas](https://learning.oreilly.com/library/view/programming-kubernetes/)

