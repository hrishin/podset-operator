package controller

import (
	"fmt"
	"os"
	"reflect"
	"sync"

	"github.com/hrishin/podset-operator/pkg/apis/demo/v1alpha1"
	"github.com/hrishin/podset-operator/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
)

const (
	APP_LABEL = "app"
)

type podSetController struct {
	kc  k8s.Interface
	psc versioned.Interface
	ns  string
}

func New(kc k8s.Interface, pc versioned.Interface, namespace string) *podSetController {
	return &podSetController{
		kc:  kc,
		psc: pc,
		ns:  namespace,
	}
}

func (c *podSetController) Run() error {
	var wg sync.WaitGroup
	wg.Add(2)

	// watch for the PodSet resources
	// Primary resource
	go func(c *podSetController) {
		defer wg.Done()
		psWatcher, err := c.psc.DemoV1alpha1().
			PodSets(c.ns).
			Watch(metav1.ListOptions{})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error in watching podsets: %v", err)
			os.Exit(1)
		}

		psCh := psWatcher.ResultChan()
		for event := range psCh {
			ps, ok := event.Object.(*v1alpha1.PodSet)
			if !ok {
				fmt.Errorf("Podset event error : %s, \n", err)
			}
			fmt.Printf("PodSet event type: %s,  name:%v \n", event.Type, ps.Name)

			c.reconcile(ps)
		}
	}(c)

	// watch for the Pod resources
	// Secondary resource
	go func(c *podSetController) {
		defer wg.Done()
		podWatcher, err := c.kc.CoreV1().
			Pods(c.ns).
			Watch(metav1.ListOptions{})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error in watching pods: %v", err)
			os.Exit(1)
		}

		podCh := podWatcher.ResultChan()
		for event := range podCh {
			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				fmt.Errorf("Pod event error : %s, \n", err)
			}
			fmt.Printf("Pod event type: %s,  name:%v \n", event.Type, pod.Name)
			ps := c.podSetOwnerFor(pod)
			if ps == nil {
				continue
			}
			c.reconcile(ps)
		}
	}(c)
	wg.Wait()

	return nil
}

// it tries to achive the desired state for PodSet
func (c *podSetController) reconcile(ps *v1alpha1.PodSet) {
	// get the existing pods
	pods, err := c.podCountByLabel(APP_LABEL, ps.Name)
	if err != nil {
		fmt.Println(err)
	}

	// compare it with desired state i.e spec.replicas
	// if less then spin up pods
	if int32(len(pods)) < ps.Spec.Replicas {
		pod := newPodForCR(ps)
		//TODO: add owner reference
		_, err := c.kc.CoreV1().Pods(c.ns).Create(pod)
		if err != nil {
			fmt.Println(err)
		}
	}
	// if more then delete the pods
	if diff := int32(len(pods)) - ps.Spec.Replicas; diff > 0 {
		pod := pods[0]
		err := c.kc.CoreV1().Pods(c.ns).Delete(pod, &metav1.DeleteOptions{})
		if err != nil {
			fmt.Println(err)
		}
	}

	// update the status (status.availablereplicas)
	status := v1alpha1.PodSetStatus{
		AvailableReplicas: int32(len(pods)),
	}
	if !reflect.DeepEqual(status, ps.Status) {
		ps.Status = status
		_, err := c.psc.DemoV1alpha1().PodSets(c.ns).Update(ps)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func (c *podSetController) podCountByLabel(key, value string) ([]string, error) {
	pNames := []string{}

	pods, err := c.kc.CoreV1().Pods(c.ns).List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", key, value),
	})

	if err != nil {
		return pNames, fmt.Errorf("Error in retriving pods\n")
	}

	for _, p := range pods.Items {
		if p.Status.Phase == corev1.PodPending || p.Status.Phase == corev1.PodRunning {
			pNames = append(pNames, p.Name)
		}
	}

	fmt.Printf("count : %d \n", len(pNames))

	return pNames, nil
}

// get owner PodSet for the given pod if any
func (c *podSetController) podSetOwnerFor(pod *corev1.Pod) *v1alpha1.PodSet {
	podSet, ok := pod.Labels[APP_LABEL]
	if !ok {
		return nil
	}

	ps, err := c.psc.DemoV1alpha1().PodSets(c.ns).Get(podSet, metav1.GetOptions{})
	if err != nil {
		return nil
	}

	return ps
}

func newPodForCR(ps *v1alpha1.PodSet) *corev1.Pod {
	labels := map[string]string{
		APP_LABEL: ps.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: ps.Name + "-pod",
			Namespace:    ps.Namespace,
			Labels:       labels,
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
