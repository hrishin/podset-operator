Steps 1:
Generates the deep copy and client go

```
~/go/src/k8s.io/code-generator/generate-groups.sh "deepcopy,client,informer,lister" \
github.com/hrishin/podset-operator/pkg/client \
github.com/hrishin/podset-operator/pkg/apis \
demo:v1alpha1
```