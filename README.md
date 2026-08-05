## PodSet Operator

The objective of this Operator/Controller is to demonstrate a `ReplicaSet`-like resource
implementation using the Kubernetes controller pattern.

Another objective of this repo is to show how to build a controller from scratch, and what
challenges a developer could face along the way, so that a developer can appreciate the value
that frameworks like [KubeBuilder](https://github.com/kubernetes-sigs/kubebuilder) or the
[Operator SDK](https://github.com/operator-framework/operator-sdk) provide.

#### PodSet resource

Once a user applies a `PodSet` resource (`kubectl apply -f podset.yaml`), the controller spins up
the number of pods specified in the `replicas` field.

e.g. to spin up 3 pods:

```yaml
apiVersion: demo.k8s.io/v1alpha1
kind: PodSet
metadata:
  name: three-podset
spec:
  replicas: 3
```

### Prerequisites

* A Kubernetes cluster 1.35+ (e.g. [kind](https://kind.sigs.k8s.io/) works well for local testing)
* Go 1.25+

### Presentation

- [Slides (PDF)](https://raw.githubusercontent.com/hrishin/podset-operator/master/presentation.pdf)
- [Slides transcript (Markdown)](/docs/presentation.md)

### Tutorial

Check out the code according to the following instructions, and check the README on each branch
to follow the further instructions.

#### step 1
```
git checkout step-1
```
Covers basic code and scaffolding setup.

#### step 2
```
git checkout step-2
```
Covers how to define CRD types, register CRDs, and generate client APIs using `client-gen` and
friends. Includes a simple program that issues a `watch` request for the `PodSet` resource and
prints resource state changes to the console.

#### step 3
```
git checkout step-3
```
Covers a functional controller using the basic generated code. Shows how to issue watch requests
and bring a `PodSet` resource to its desired state (reconciliation).

#### step 4
```
git checkout step-4
```
Covers a fully functional controller using shared informers, listers, and workqueues, and how to
generate all of those objects. At this point one should be able to relate to why controllers are
written the way they are.

#### step 5
```
git checkout step-5
```
Completes the same functional operator, this time rewritten using
[KubeBuilder](https://github.com/kubernetes-sigs/kubebuilder). All of the boilerplate built by
hand in steps 2-4 (deepcopy/clientset/informer/lister generation, manual watch registration, the
workqueue, owner-reference bookkeeping) is now handled by
[controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) and the `kubebuilder`
scaffolding, while the reconcile logic stays functionally identical.

***Note: This code is intended for educational purposes. Less focus is given to code quality.***

### Credits
- [https://github.com/kubernetes/sample-controller](https://github.com/kubernetes/sample-controller)
- [Programming Kubernetes by Stefan Schimanski, Michael Hausenblas](https://learning.oreilly.com/library/view/programming-kubernetes/)
