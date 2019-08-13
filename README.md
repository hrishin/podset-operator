## PodSet Operator 

Objective of this Operator/Controller is demonstrate `ReplicaSet` kind of resource
implementation using Kubernetes controller pattern.
Another objective of this repo. is to show how to build the controller from scratch and what challenges/gotchas developer could face. 
So a developer could understand the beauty [KuberBuilder](https://github.com/kubernetes-sigs/kubebuilder) or [Operator SDK](https://github.com/operator-framework/operator-sdk) frameworks


#### PodSet resource

Once user applies the `PodSet` (`kubectl apply -f podset.yaml`) resource, controller could spin up
number of pods mentioned as `replicas` filed.

e.g. User want to spin up 3 pod

```yaml
apiVersion: demo.k8s.io/v1alpha1
kind: PodSet
metadata:
  name: three-podset
spec:
  replicas: 3
```

###Step-1:

setup 

```
git clone git@github.com:hrishin/podset-operator.git hrishin/podset-operator
```

Need an API to interact with `PodSet` resource in order to create, update, delete and **watch** the resource state