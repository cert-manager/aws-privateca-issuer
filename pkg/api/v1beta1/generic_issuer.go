/*
Copyright 2021 The Kubernetes Authors.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// GenericIssuer is an interface for both issuer types to simplify controller code
// +k8s:deepcopy-gen=false
type GenericIssuer interface {
	runtime.Object
	metav1.Object

	GetObjectMeta() *metav1.ObjectMeta
	GetSpec() *AWSPCAIssuerSpec
	GetStatus() *AWSPCAIssuerStatus
}

var _ GenericIssuer = &AWSPCAIssuer{}
var _ GenericIssuer = &AWSPCAClusterIssuer{}

// GetObjectMeta returns the k8s object metadata
func (c *AWSPCAClusterIssuer) GetObjectMeta() *metav1.ObjectMeta {
	return &c.ObjectMeta
}

// GetSpec returns the issuer spec
func (c *AWSPCAClusterIssuer) GetSpec() *AWSPCAIssuerSpec {
	return &c.Spec
}

// GetStatus returns the issuer status
func (c *AWSPCAClusterIssuer) GetStatus() *AWSPCAIssuerStatus {
	return &c.Status
}

// SetSpec sets the issuer spec
func (c *AWSPCAClusterIssuer) SetSpec(spec AWSPCAIssuerSpec) {
	c.Spec = spec
}

// SetStatus sets the issuer status
func (c *AWSPCAClusterIssuer) SetStatus(status AWSPCAIssuerStatus) {
	c.Status = status
}

// Copy deep copies the issuer
func (c *AWSPCAClusterIssuer) Copy() GenericIssuer {
	return c.DeepCopy()
}

// GetObjectMeta returns the k8s object metadata
func (c *AWSPCAIssuer) GetObjectMeta() *metav1.ObjectMeta {
	return &c.ObjectMeta
}

// GetSpec returns the issuer spec
func (c *AWSPCAIssuer) GetSpec() *AWSPCAIssuerSpec {
	return &c.Spec
}

// GetStatus returns the issuer status
func (c *AWSPCAIssuer) GetStatus() *AWSPCAIssuerStatus {
	return &c.Status
}

// SetSpec sets the issuer spec
func (c *AWSPCAIssuer) SetSpec(spec AWSPCAIssuerSpec) {
	c.Spec = spec
}

// SetStatus sets the issuer status
func (c *AWSPCAIssuer) SetStatus(status AWSPCAIssuerStatus) {
	c.Status = status
}

// Copy deep copies the issuer
func (c *AWSPCAIssuer) Copy() GenericIssuer {
	return c.DeepCopy()
}
