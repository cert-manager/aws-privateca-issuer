package v1beta1

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cert-manager/aws-privateca-issuer/pkg/api/v1beta1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

var (
	awspcaissuers        = "awspcaissuers"
	awspcaclusterissuers = "awspcaclusterissuers"
)

// AWSPCAIssuerInterface is a interface for interacting with a AWSPCAIssuer
type AWSPCAIssuerInterface interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1beta1.AWSPCAIssuer, error)
	Create(ctx context.Context, issuer *v1beta1.AWSPCAIssuer, opts metav1.CreateOptions) (*v1beta1.AWSPCAIssuer, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
}

// AWSPCAClusterIssuerInterface is a interface for interacting with a AWSPCAClusterIssuer
type AWSPCAClusterIssuerInterface interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1beta1.AWSPCAClusterIssuer, error)
	Create(ctx context.Context, issuer *v1beta1.AWSPCAClusterIssuer, opts metav1.CreateOptions) (*v1beta1.AWSPCAClusterIssuer, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
}

type awspcaIssuerClient struct {
	restClient rest.Interface
	ns         string
}

type awspcaClusterIssuerClient struct {
	restClient rest.Interface
}

func (c *awspcaIssuerClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1beta1.AWSPCAIssuer, error) {
	result := v1beta1.AWSPCAIssuer{}
	err := c.restClient.Get().
		Namespace(c.ns).
		Resource(awspcaissuers).
		VersionedParams(&opts, scheme.ParameterCodec).
		Name(name).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *awspcaClusterIssuerClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1beta1.AWSPCAClusterIssuer, error) {
	result := v1beta1.AWSPCAClusterIssuer{}
	err := c.restClient.Get().
		Resource(awspcaclusterissuers).
		VersionedParams(&opts, scheme.ParameterCodec).
		Name(name).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *awspcaIssuerClient) Create(ctx context.Context, issuer *v1beta1.AWSPCAIssuer, opts metav1.CreateOptions) (*v1beta1.AWSPCAIssuer, error) {
	result := v1beta1.AWSPCAIssuer{}
	err := c.restClient.Post().
		Namespace(c.ns).
		Resource(awspcaissuers).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(issuer).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *awspcaClusterIssuerClient) Create(ctx context.Context, issuer *v1beta1.AWSPCAClusterIssuer, opts metav1.CreateOptions) (*v1beta1.AWSPCAClusterIssuer, error) {
	result := v1beta1.AWSPCAClusterIssuer{}
	err := c.restClient.Post().
		Resource(awspcaclusterissuers).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(issuer).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *awspcaIssuerClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.restClient.Delete().
		Namespace(c.ns).
		Resource(awspcaissuers).
		VersionedParams(&opts, scheme.ParameterCodec).
		Name(name).
		Do(ctx).
		Error()
}

func (c *awspcaClusterIssuerClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.restClient.Delete().
		Resource(awspcaclusterissuers).
		VersionedParams(&opts, scheme.ParameterCodec).
		Name(name).
		Do(ctx).
		Error()
}

func (c *awspcaIssuerClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.restClient.Get().
		Resource(awspcaissuers).
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

func (c *awspcaClusterIssuerClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.restClient.Get().
		Resource(awspcaclusterissuers).
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}
