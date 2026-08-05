## PodSet Operator

The objective of this Operator/Controller is to demonstrate a `ReplicaSet`-like resource
implementation using the Kubernetes controller pattern.

Another objective of this repo is to show how to build a controller from scratch, and what
challenges/gotchas a developer could face along the way, so that a developer can appreciate the
value that frameworks like [KubeBuilder](https://github.com/kubernetes-sigs/kubebuilder) or the
[Operator SDK](https://github.com/operator-framework/operator-sdk) provide.

Steps 1-4 (see the `step-1`..`step-4` branches) build that controller by hand: custom types,
generated clientsets, raw `Watch()` calls, and finally hand-rolled informers/listers/workqueues.

**Step 5 (this branch) is the payoff** — the same `PodSet` controller, rewritten with
[KubeBuilder](https://github.com/kubernetes-sigs/kubebuilder), aka *"Controller: the Unicorn
way!"* from the [presentation](https://github.com/hrishin/podset-operator/blob/master/docs/presentation.md#controller-the-unicorn-way). All of the
boilerplate from steps 2-4 (deepcopy/clientset/informer/lister generation, manual watch
registration, the workqueue, owner-reference bookkeeping) is now handled by
[controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) and the `kubebuilder`
scaffolding.

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

### What's different from steps 1-4

| | Hand-rolled (steps 1-4) | KubeBuilder (step 5) |
|---|---|---|
| API types + deepcopy | generated via `code-generator` | generated via `controller-gen` (`make generate`) |
| Client | generated typed clientset | `sigs.k8s.io/controller-runtime` client.Client |
| Watching Pods owned by a PodSet | manual `handlePodObject` + owner-ref lookup | `Owns(&corev1.Pod{})` |
| Informers / listers / workqueue | hand-written (`pkg/controller/controller.go`) | provided by the controller-runtime manager |
| CRD manifest | hand-written YAML | generated via `make manifests` |

`internal/controller/podset_controller.go`'s `Reconcile` implements the exact same logic as the
step-4 controller: count running Pods labeled for this PodSet, create/delete one Pod to converge
on `spec.replicas`, and write `status.availableReplicas`.

### Prerequisites

* A Kubernetes cluster 1.35+ (e.g. [kind](https://kind.sigs.k8s.io/) works well for local testing)
* Go 1.25+
* kubebuilder 4.x (only needed if you want to re-scaffold or add new APIs; not required to just
  build/run — see [Installing kubebuilder](#installing-kubebuilder) below)

### Installing kubebuilder

macOS (Homebrew):
```
$ brew install kubebuilder
```

Linux/macOS (official install script, picks the right binary for your OS/arch):
```
$ curl -L -o kubebuilder "https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH)"
$ chmod +x kubebuilder
$ sudo mv kubebuilder /usr/local/bin/
```

Verify:
```
$ kubebuilder version
```

See the [kubebuilder quick-start](https://book.kubebuilder.io/quick-start.html#installation) for
other install options (e.g. Windows).

### Presentation

- [Slides (PDF)](https://raw.githubusercontent.com/hrishin/podset-operator/master/presentation.pdf)
- [Slides transcript (Markdown)](https://github.com/hrishin/podset-operator/blob/master/docs/presentation.md)

### Running step 5

* Install the CRD into your cluster:
```
$ make install
```

* Run the controller locally, against whatever cluster your current kubeconfig context points at:
```
$ make run
```

* In another terminal, create a PodSet:
```
$ kubectl apply -f resources/cr.yaml
```
(or `config/samples/demo_v1alpha1_podset.yaml`, which `make install`/`kustomize` also knows about)

* Watch the pods come up, and the PodSet converge to the desired replica count:
```
$ kubectl get pods -w
$ kubectl get podset example-podset -o wide
```

* Clean up:
```
$ kubectl delete -f resources/cr.yaml
$ make uninstall
```

To regenerate the CRD/deepcopy code after editing `api/v1alpha1/podset_types.go`:
```
$ make manifests generate
```

***Note: This code is intended for educational purposes. Less focus is given to code quality.***

### Credits
- [https://github.com/kubernetes/sample-controller](https://github.com/kubernetes/sample-controller)
- [https://github.com/kubernetes-sigs/kubebuilder](https://github.com/kubernetes-sigs/kubebuilder)
- [Programming Kubernetes by Stefan Schimanski, Michael Hausenblas](https://learning.oreilly.com/library/view/programming-kubernetes/)
