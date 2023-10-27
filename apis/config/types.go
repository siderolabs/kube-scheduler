package config

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EmissionsArgs defines the parameters for Emissions plugin.
type EmissionsArgs struct {
	metav1.TypeMeta `json:",inline"`
}
