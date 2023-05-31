/*
Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file

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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	machine "github.com/xuanson2406/machine-controller-manager/pkg/apis/machine"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeMachineDeployments implements MachineDeploymentInterface
type FakeMachineDeployments struct {
	Fake *FakeMachine
	ns   string
}

var machinedeploymentsResource = schema.GroupVersionResource{Group: "machine.sapcloud.io", Version: "", Resource: "machinedeployments"}

var machinedeploymentsKind = schema.GroupVersionKind{Group: "machine.sapcloud.io", Version: "", Kind: "MachineDeployment"}

// Get takes name of the machineDeployment, and returns the corresponding machineDeployment object, and an error if there is any.
func (c *FakeMachineDeployments) Get(name string, options v1.GetOptions) (result *machine.MachineDeployment, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(machinedeploymentsResource, c.ns, name), &machine.MachineDeployment{})

	if obj == nil {
		return nil, err
	}
	return obj.(*machine.MachineDeployment), err
}

// List takes label and field selectors, and returns the list of MachineDeployments that match those selectors.
func (c *FakeMachineDeployments) List(opts v1.ListOptions) (result *machine.MachineDeploymentList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(machinedeploymentsResource, machinedeploymentsKind, c.ns, opts), &machine.MachineDeploymentList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &machine.MachineDeploymentList{ListMeta: obj.(*machine.MachineDeploymentList).ListMeta}
	for _, item := range obj.(*machine.MachineDeploymentList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested machineDeployments.
func (c *FakeMachineDeployments) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(machinedeploymentsResource, c.ns, opts))

}

// Create takes the representation of a machineDeployment and creates it.  Returns the server's representation of the machineDeployment, and an error, if there is any.
func (c *FakeMachineDeployments) Create(machineDeployment *machine.MachineDeployment) (result *machine.MachineDeployment, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(machinedeploymentsResource, c.ns, machineDeployment), &machine.MachineDeployment{})

	if obj == nil {
		return nil, err
	}
	return obj.(*machine.MachineDeployment), err
}

// Update takes the representation of a machineDeployment and updates it. Returns the server's representation of the machineDeployment, and an error, if there is any.
func (c *FakeMachineDeployments) Update(machineDeployment *machine.MachineDeployment) (result *machine.MachineDeployment, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(machinedeploymentsResource, c.ns, machineDeployment), &machine.MachineDeployment{})

	if obj == nil {
		return nil, err
	}
	return obj.(*machine.MachineDeployment), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeMachineDeployments) UpdateStatus(machineDeployment *machine.MachineDeployment) (*machine.MachineDeployment, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(machinedeploymentsResource, "status", c.ns, machineDeployment), &machine.MachineDeployment{})

	if obj == nil {
		return nil, err
	}
	return obj.(*machine.MachineDeployment), err
}

// Delete takes name of the machineDeployment and deletes it. Returns an error if one occurs.
func (c *FakeMachineDeployments) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(machinedeploymentsResource, c.ns, name), &machine.MachineDeployment{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeMachineDeployments) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(machinedeploymentsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &machine.MachineDeploymentList{})
	return err
}

// Patch applies the patch and returns the patched machineDeployment.
func (c *FakeMachineDeployments) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *machine.MachineDeployment, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(machinedeploymentsResource, c.ns, name, pt, data, subresources...), &machine.MachineDeployment{})

	if obj == nil {
		return nil, err
	}
	return obj.(*machine.MachineDeployment), err
}

// GetScale takes name of the machineDeployment, and returns the corresponding scale object, and an error if there is any.
func (c *FakeMachineDeployments) GetScale(machineDeploymentName string, options v1.GetOptions) (result *machine.Scale, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetSubresourceAction(machinedeploymentsResource, c.ns, "scale", machineDeploymentName), &machine.Scale{})

	if obj == nil {
		return nil, err
	}
	return obj.(*machine.Scale), err
}

// UpdateScale takes the representation of a scale and updates it. Returns the server's representation of the scale, and an error, if there is any.
func (c *FakeMachineDeployments) UpdateScale(machineDeploymentName string, scale *machine.Scale) (result *machine.Scale, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(machinedeploymentsResource, "scale", c.ns, scale), &machine.Scale{})

	if obj == nil {
		return nil, err
	}
	return obj.(*machine.Scale), err
}
