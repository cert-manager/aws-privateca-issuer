/*
Copyright 2021.

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

package v1beta1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AWSPCAIssuerSpec defines the desired state of AWSPCAIssuer
type AWSPCAIssuerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Specifies the ARN of the PCA resource
	Arn string `json:"arn,omitempty"`
	// Should contain the AWS region if it cannot be inferred
	// +optional
	Region string `json:"region,omitempty"`
	// Needs to be specified if you want to authorize with AWS using an access and secret key
	// +optional
	SecretRef v1.SecretReference `json:"secretRef,omitempty"`
}

// AWSPCAIssuerStatus defines the observed state of AWSPCAIssuer
type AWSPCAIssuerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// ConditionTypeReady is the default condition type for the CRs
const ConditionTypeReady = "Ready"

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// AWSPCAIssuer is the Schema for the awspcaissuers API
type AWSPCAIssuer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AWSPCAIssuerSpec   `json:"spec,omitempty"`
	Status AWSPCAIssuerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AWSPCAIssuerList contains a list of AWSPCAIssuer
type AWSPCAIssuerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AWSPCAIssuer `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// AWSPCAClusterIssuer is the Schema for the awspcaclusterissuers API
// +kubebuilder:resource:path=awspcaclusterissuers,scope=Cluster
type AWSPCAClusterIssuer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AWSPCAIssuerSpec   `json:"spec,omitempty"`
	Status AWSPCAIssuerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AWSPCAClusterIssuerList contains a list of AWSPCAClusterIssuer
type AWSPCAClusterIssuerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AWSPCAClusterIssuer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AWSPCAIssuer{}, &AWSPCAIssuerList{})
	SchemeBuilder.Register(&AWSPCAClusterIssuer{}, &AWSPCAClusterIssuerList{})
}
