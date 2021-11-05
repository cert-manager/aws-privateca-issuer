package v1beta1

import (
	api "github.com/cert-manager/aws-privateca-issuer/pkg/api/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

// Interface defines inteface for interacting with AWS PCA issuers
type Interface interface {
	AWSPCAIssuers(namespace string) AWSPCAIssuerInterface
	AWSPCAClusterIssuers() AWSPCAClusterIssuerInterface
}

//Client defines a client for interacting with AWS PCA issuers
type Client struct {
	restClient rest.Interface
}

//NewForConfig is a function which lets you configure pca issuer clientset
func NewForConfig(c *rest.Config) (*Client, error) {
	err := AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}

	config := *c
	config.ContentConfig.GroupVersion = &api.GroupVersion
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	return &Client{restClient: client}, nil
}

//AWSPCAIssuers is a function which lets you interact with AWSPCAIssuers
func (c *Client) AWSPCAIssuers(namespace string) AWSPCAIssuerInterface {
	return &awspcaIssuerClient{
		restClient: c.restClient,
		ns:         namespace,
	}
}

//AWSPCAClusterIssuers is a function which lets you interact with AWSPCAClusterIssuers
func (c *Client) AWSPCAClusterIssuers() AWSPCAClusterIssuerInterface {
	return &awspcaClusterIssuerClient{
		restClient: c.restClient,
	}
}
