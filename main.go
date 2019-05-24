package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hrishin/podset-operator/pkg/apis/demo/v1alpha1"
	clientset "github.com/hrishin/podset-operator/pkg/client/clientset/versioned"
	sampleScheme "github.com/hrishin/podset-operator/pkg/client/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	kubeconfig := ""
	flag.StringVar(&kubeconfig, "kubeconfig", kubeconfig, "kubeconfig file")
	flag.Parse()

	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBECONFIG")
	}

	var (
		config *rest.Config
		err    error
	)

	if kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating client: %v", err)
		os.Exit(1)
	}

	client := clientset.NewForConfigOrDie(config)

	// To check if PodSet resource exist
	utilruntime.Must(sampleScheme.AddToScheme(scheme.Scheme))

	watcher, err := client.DemoV1alpha1().
		PodSets("pods").
		Watch(metav1.ListOptions{})

	if err != nil {
		fmt.Fprintf(os.Stderr, "error listing podsets: %v", err)
		os.Exit(1)
	}

	ch := watcher.ResultChan()
	for event := range ch {
		ps, ok := event.Object.(*v1alpha1.PodSet)
		if !ok {
			fmt.Printf("Event error : %s, \n", err)
		}
		fmt.Printf("Event type: %s,  name:%v \n", event.Type, ps.Name)
	}
}
