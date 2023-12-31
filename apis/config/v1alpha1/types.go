package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:defaulter-gen=true

// EmissionsArgs defines the parameters for Emissions plugin.
type EmissionsArgs struct {
	metav1.TypeMeta `json:",inline"`

	// WattTimeUsername is the WattTime username.
	WattTimeUsername *string `json:"wattTimeUsername,omitempty"`
	// WattTimePassword is the WattTime password.
	WattTimePassword *string `json:"wattTimePassword,omitempty"`
	// WattTimeBA is the WattTime BA.
	WattTimeBA *string `json:"wattTimeBA,omitempty"`
}
