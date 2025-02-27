// Copyright 2019, Verizon Media Inc.
// Licensed under the terms of the 3-Clause BSD license. See LICENSE file in
// github.com/yahoo/k8s-athenz-istio-auth for terms.
// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	athenz_v1 "github.com/yahoo/k8s-athenz-istio-auth/pkg/apis/athenz/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeAthenzDomains implements AthenzDomainInterface
type FakeAthenzDomains struct {
	Fake *FakeAthenzV1
	ns   string
}

var athenzdomainsResource = schema.GroupVersionResource{Group: "athenz", Version: "v1", Resource: "athenzdomains"}

var athenzdomainsKind = schema.GroupVersionKind{Group: "athenz", Version: "v1", Kind: "AthenzDomain"}

// Get takes name of the athenzDomain, and returns the corresponding athenzDomain object, and an error if there is any.
func (c *FakeAthenzDomains) Get(name string, options v1.GetOptions) (result *athenz_v1.AthenzDomain, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(athenzdomainsResource, c.ns, name), &athenz_v1.AthenzDomain{})

	if obj == nil {
		return nil, err
	}
	return obj.(*athenz_v1.AthenzDomain), err
}

// List takes label and field selectors, and returns the list of AthenzDomains that match those selectors.
func (c *FakeAthenzDomains) List(opts v1.ListOptions) (result *athenz_v1.AthenzDomainList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(athenzdomainsResource, athenzdomainsKind, c.ns, opts), &athenz_v1.AthenzDomainList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &athenz_v1.AthenzDomainList{ListMeta: obj.(*athenz_v1.AthenzDomainList).ListMeta}
	for _, item := range obj.(*athenz_v1.AthenzDomainList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested athenzDomains.
func (c *FakeAthenzDomains) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(athenzdomainsResource, c.ns, opts))

}

// Create takes the representation of a athenzDomain and creates it.  Returns the server's representation of the athenzDomain, and an error, if there is any.
func (c *FakeAthenzDomains) Create(athenzDomain *athenz_v1.AthenzDomain) (result *athenz_v1.AthenzDomain, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(athenzdomainsResource, c.ns, athenzDomain), &athenz_v1.AthenzDomain{})

	if obj == nil {
		return nil, err
	}
	return obj.(*athenz_v1.AthenzDomain), err
}

// Update takes the representation of a athenzDomain and updates it. Returns the server's representation of the athenzDomain, and an error, if there is any.
func (c *FakeAthenzDomains) Update(athenzDomain *athenz_v1.AthenzDomain) (result *athenz_v1.AthenzDomain, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(athenzdomainsResource, c.ns, athenzDomain), &athenz_v1.AthenzDomain{})

	if obj == nil {
		return nil, err
	}
	return obj.(*athenz_v1.AthenzDomain), err
}

// Delete takes name of the athenzDomain and deletes it. Returns an error if one occurs.
func (c *FakeAthenzDomains) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(athenzdomainsResource, c.ns, name), &athenz_v1.AthenzDomain{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeAthenzDomains) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(athenzdomainsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &athenz_v1.AthenzDomainList{})
	return err
}

// Patch applies the patch and returns the patched athenzDomain.
func (c *FakeAthenzDomains) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *athenz_v1.AthenzDomain, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(athenzdomainsResource, c.ns, name, data, subresources...), &athenz_v1.AthenzDomain{})

	if obj == nil {
		return nil, err
	}
	return obj.(*athenz_v1.AthenzDomain), err
}
