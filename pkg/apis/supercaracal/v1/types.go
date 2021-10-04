package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient

// FooBar is
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type FooBar struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FooBarSpec   `json:"spec"`
	Status FooBarStatus `json:"status"`
}

// FooBarSpec is
type FooBarSpec struct {
	// message to log
	Message string `json:"message"`
}

// FooBarStatus is
type FooBarStatus struct {
	Succeeded bool
}

// FooBarList is
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type FooBarList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []FooBar `json:"items"`
}
