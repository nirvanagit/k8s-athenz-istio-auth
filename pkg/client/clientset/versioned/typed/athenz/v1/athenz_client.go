// Copyright 2019, Verizon Media Inc.
// Licensed under the terms of the 3-Clause BSD license. See LICENSE file in
// github.com/yahoo/k8s-athenz-istio-auth for terms.
// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/yahoo/k8s-athenz-istio-auth/pkg/apis/athenz/v1"
	"github.com/yahoo/k8s-athenz-istio-auth/pkg/client/clientset/versioned/scheme"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer"
	rest "k8s.io/client-go/rest"
)

type AthenzV1Interface interface {
	RESTClient() rest.Interface
	AthenzDomainsGetter
}

// AthenzV1Client is used to interact with features provided by the athenz group.
type AthenzV1Client struct {
	restClient rest.Interface
}

func (c *AthenzV1Client) AthenzDomains(namespace string) AthenzDomainInterface {
	return newAthenzDomains(c, namespace)
}

// NewForConfig creates a new AthenzV1Client for the given config.
func NewForConfig(c *rest.Config) (*AthenzV1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &AthenzV1Client{client}, nil
}

// NewForConfigOrDie creates a new AthenzV1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *AthenzV1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new AthenzV1Client for the given RESTClient.
func New(c rest.Interface) *AthenzV1Client {
	return &AthenzV1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *AthenzV1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
