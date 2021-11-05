package v1beta1

import (
	api "github.com/cert-manager/aws-privateca-issuer/pkg/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	schemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	//AddToScheme allows you to add v1beta1 to scheme
	AddToScheme = schemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(api.GroupVersion,
		&api.AWSPCAClusterIssuer{},
		&api.AWSPCAIssuer{},
	)

	metav1.AddToGroupVersion(scheme, api.GroupVersion)
	return nil
}
