package main

import (
	"flag"
	"fmt"
	"time"
	"os"

	clientset "github.com/hrishin/podset-operator/pkg/client/clientset/versioned"
	sampleScheme "github.com/hrishin/podset-operator/pkg/client/clientset/versioned/scheme"
	poc "github.com/hrishin/podset-operator/pkg/controller"
	"github.com/hrishin/podset-operator/pkg/signals"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	kubeinformers "k8s.io/client-go/informers"
	psinformers "github.com/hrishin/podset-operator/pkg/client/informers/externalversions"
)

func main() {
	kubeconfig := ""
	flag.StringVar(&kubeconfig, "kubeconfig", kubeconfig, "kubeconfig file")
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

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

	k8sInformerFactory := kubeinformers.NewSharedInformerFactory(k8sClient, time.Minute * 10)
	psInformerFactory := psinformers.NewSharedInformerFactory(psClient, time.Minute * 10)

	psc := poc.New(k8sClient, psClient, 
		k8sInformerFactory.Core().V1().Pods(), 
		psInformerFactory.Demo().V1alpha1().PodSets())

	k8sInformerFactory.Start(stopCh)
	psInformerFactory.Start(stopCh)

	if err := psc.Run(1, stopCh); err != nil {
		fmt.Fprintf(os.Stderr, "error running controller: %v", err)
		os.Exit(1)
	}
}
