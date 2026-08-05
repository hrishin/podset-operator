/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	demov1alpha1 "github.com/hrishin/podset-operator/api/v1alpha1"
)

// appLabel is the label key used to associate Pods with the PodSet that owns them.
const appLabel = "app"

// PodSetReconciler reconciles a PodSet object
type PodSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=demo.k8s.io,resources=podsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=demo.k8s.io,resources=podsets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=demo.k8s.io,resources=podsets/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;delete

// Reconcile brings the actual number of Pods owned by a PodSet in line with
// spec.replicas, and reports how many are currently running via
// status.availableReplicas. This mirrors the hand-written controller from
// step-4 (pkg/controller/controller.go), but relies on controller-runtime's
// client, cache and owner-reference-based Pod watch instead of hand-rolled
// informers/listers/workqueue.
func (r *PodSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var podSet demov1alpha1.PodSet
	if err := r.Get(ctx, req.NamespacedName, &podSet); err != nil {
		// The PodSet may have been deleted; owned Pods are garbage collected
		// by Kubernetes via their owner references, nothing more to do.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var podList corev1.PodList
	if err := r.List(ctx, &podList,
		client.InNamespace(podSet.Namespace),
		client.MatchingLabels{appLabel: podSet.Name},
	); err != nil {
		return ctrl.Result{}, err
	}

	var running []corev1.Pod
	for _, pod := range podList.Items {
		if pod.Status.Phase == corev1.PodPending || pod.Status.Phase == corev1.PodRunning {
			running = append(running, pod)
		}
	}
	existingPods := int32(len(running))

	if existingPods < podSet.Spec.Replicas {
		pod := newPod(&podSet)
		if err := controllerutil.SetControllerReference(&podSet, pod, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, pod); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("created pod", "pod", pod.Name)
	}

	if diff := existingPods - podSet.Spec.Replicas; diff > 0 {
		pod := &running[0]
		if err := r.Delete(ctx, pod); err != nil && !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		log.Info("deleted pod", "pod", pod.Name)
	}

	if podSet.Status.AvailableReplicas != existingPods {
		podSet.Status.AvailableReplicas = existingPods
		if err := r.Status().Update(ctx, &podSet); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func newPod(ps *demov1alpha1.PodSet) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: ps.Name + "-pod",
			Namespace:    ps.Namespace,
			Labels:       map[string]string{appLabel: ps.Name},
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

// SetupWithManager sets up the controller with the Manager.
func (r *PodSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&demov1alpha1.PodSet{}).
		Owns(&corev1.Pod{}).
		Named("podset").
		Complete(r)
}
