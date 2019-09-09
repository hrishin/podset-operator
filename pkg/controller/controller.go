package controller

import (
	"fmt"
	"time"

	"github.com/hrishin/podset-operator/pkg/apis/demo/v1alpha1"
	"github.com/hrishin/podset-operator/pkg/client/clientset/versioned"
	psinformers "github.com/hrishin/podset-operator/pkg/client/informers/externalversions/demo/v1alpha1"
	pslister "github.com/hrishin/podset-operator/pkg/client/listers/demo/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	podinformers "k8s.io/client-go/informers/core/v1"
	k8s "k8s.io/client-go/kubernetes"
	podlister "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	APP_LABEL = "app"
)

type podSetController struct {
	kc           k8s.Interface
	psc          versioned.Interface
	podLister    podlister.PodLister
	podHasSynced cache.InformerSynced
	psLister     pslister.PodSetLister
	psHasSynced  cache.InformerSynced
	workqueue    workqueue.RateLimitingInterface
	ns           string
}

func New(kc k8s.Interface,
	pc versioned.Interface,
	podInformer podinformers.PodInformer,
	psInformer psinformers.PodSetInformer) *podSetController {

	psc := &podSetController{
		kc:           kc,
		psc:          pc,
		podLister:    podInformer.Lister(),
		podHasSynced: podInformer.Informer().HasSynced,
		psLister:     psInformer.Lister(),
		psHasSynced:  psInformer.Informer().HasSynced,
		workqueue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "PodSets"),
	}

	// watch the PodSet resources events
	// Primary resource
	psInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: psc.enqueuePodSet,
		UpdateFunc: func(old, new interface{}) {
			psc.enqueuePodSet(new)
		},
	})

	// watch the Pod resources events
	// Secondary resource
	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: psc.handlePodObject,
		UpdateFunc: func(old, new interface{}) {
			newDepl := new.(*corev1.Pod)
			oldDepl := old.(*corev1.Pod)
			if newDepl.ResourceVersion == oldDepl.ResourceVersion {
				return
			}
			psc.handlePodObject(new)
		},
		DeleteFunc: psc.handlePodObject,
	})

	return psc
}

// enqueuePodSet adds objects to workqueue
func (c *podSetController) enqueuePodSet(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// enqueuePodSet adds objects to workqueue
func (c *podSetController) handlePodObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		_, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
	}

	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a PodSet, we should not do anything more
		// with it.
		if ownerRef.Kind != "PodSet" {
			return
		}

		ps, err := c.psLister.PodSets(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			fmt.Printf("ignoring orphaned object '%s' of podset '%s' \n", object.GetSelfLink(), ownerRef.Name)
			return
		}

		c.enqueuePodSet(ps)
		return
	}
}

func (c *podSetController) Run(threads int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	fmt.Println("Starting podset controller")

	// sync informer caches
	if ok := cache.WaitForCacheSync(stopCh, c.podHasSynced, c.psHasSynced); !ok {
		return fmt.Errorf("Failed sync the caches")
	}

	// start worker to process workqueue items
	for i := 0; i < threads; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *podSetController) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the eventHandler.
func (c *podSetController) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}

	// Wrapped this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("Expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// PodSet resource to be synced.
		if err := c.eventHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("Error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		fmt.Printf("Successfully synced '%s' \n", key)

		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// handleEvent compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the PodSet resource
// with the current status of the resource.
func (c *podSetController) eventHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the PodSet resource with this namespace/name
	ps, err := c.psLister.PodSets(namespace).Get(name)
	if err != nil {
		// The PodSet resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("Podset '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	return c.reconcile(ps)
}

// reconcile tries to achieve the desired state for PodSet
func (c *podSetController) reconcile(ps *v1alpha1.PodSet) error {
	// get the existing pods
	pods, err := c.podCountByLabel(APP_LABEL, ps.Name)
	if err != nil {
		return err
	}
	existingPods := int32(len(pods))

	// compare it with desired state i.e spec.replicas
	// if less then spin up pods
	if existingPods < ps.Spec.Replicas {
		pod := newPod(ps)
		_, err := c.kc.CoreV1().
			Pods(ps.Namespace).
			Create(pod)
		if err != nil {
			return err
		}
	}
	// if more then delete the pods
	if diff := existingPods - ps.Spec.Replicas; diff > 0 {
		pod := pods[0]
		err := c.kc.CoreV1().
			Pods(ps.Namespace).
			Delete(pod, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	// update the status (status.availablereplicas)
	psCopy := ps.DeepCopy()
	psCopy.Status.AvailableReplicas = existingPods
	_, err = c.psc.DemoV1alpha1().
		PodSets(psCopy.Namespace).
		Update(psCopy)
	if err != nil {
		return err
	}

	return nil
}

func (c *podSetController) podCountByLabel(key, value string) ([]string, error) {
	pNames := []string{}

	lbls := labels.Set{
		key: value,
	}
	pods, err := c.podLister.List(labels.SelectorFromSet(lbls))
	if err != nil {
		return pNames, fmt.Errorf("Error in retriving pods\n")
	}

	for _, p := range pods {
		if p.Status.Phase == corev1.PodPending || p.Status.Phase == corev1.PodRunning {
			pNames = append(pNames, p.Name)
		}
	}

	fmt.Printf("count : %d \n", len(pNames))

	return pNames, nil
}

func newPod(ps *v1alpha1.PodSet) *corev1.Pod {
	labels := map[string]string{
		APP_LABEL: ps.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: ps.Name + "-pod",
			Namespace:    ps.Namespace,
			Labels:       labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(ps, v1alpha1.SchemeGroupVersion.WithKind("PodSet")),
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: []string{"sleep", "3600"},
				},
			},
		},
	}
}
