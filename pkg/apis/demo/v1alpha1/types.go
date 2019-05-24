package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PodSet is a specification for a PodSet resource
type PodSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PodSetSpec   `json:"spec"`
	Status PodSetStatus `json:"status"`
}

// PodSetSpec is the spec for a PodSet resource
type PodSetSpec struct {
	Replicas *int32 `json:"replicas"`
}

// PodSetStatus is the status for a PodSet resource
type PodSetStatus struct {
	AvailableReplicas int32 `json:"availableReplicas"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PodSetList is a list of PodSet resources
type PodSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []PodSet `json:"items"`
}
