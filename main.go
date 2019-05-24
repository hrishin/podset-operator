package main

import (
	"flag"
	"fmt"
	"os"

	clientset "github.com/hrishin/podset-operator/pkg/client/clientset/versioned"
	sampleScheme "github.com/hrishin/podset-operator/pkg/client/clientset/versioned/scheme"
	poc "github.com/hrishin/podset-operator/pkg/controller"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
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

	k8sClient := kubernetes.NewForConfigOrDie(config)
	psClient := clientset.NewForConfigOrDie(config)

	// To check if PodSet resource exist
	utilruntime.Must(sampleScheme.AddToScheme(scheme.Scheme))

	psc := poc.New(k8sClient, psClient, "pods")
	if err := psc.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error running controller: %v", err)
		os.Exit(1)
	}
}
