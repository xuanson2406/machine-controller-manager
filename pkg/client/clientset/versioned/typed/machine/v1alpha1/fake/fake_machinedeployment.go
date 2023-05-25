/*
Copyright (c) SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file

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
	"context"

	v1alpha1 "github.com/xuanson2406/machine-controller-manager/pkg/apis/machine/v1alpha1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeMachineDeployments implements MachineDeploymentInterface
type FakeMachineDeployments struct {
	Fake *FakeMachineV1alpha1
	ns   string
}

var machinedeploymentsResource = schema.GroupVersionResource{Group: "machine.sapcloud.io", Version: "v1alpha1", Resource: "machinedeployments"}

var machinedeploymentsKind = schema.GroupVersionKind{Group: "machine.sapcloud.io", Version: "v1alpha1", Kind: "MachineDeployment"}

// Get takes name of the machineDeployment, and returns the corresponding machineDeployment object, and an error if there is any.
func (c *FakeMachineDeployments) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.MachineDeployment, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(machinedeploymentsResource, c.ns, name), &v1alpha1.MachineDeployment{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MachineDeployment), err
}

// List takes label and field selectors, and returns the list of MachineDeployments that match those selectors.
func (c *FakeMachineDeployments) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.MachineDeploymentList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(machinedeploymentsResource, machinedeploymentsKind, c.ns, opts), &v1alpha1.MachineDeploymentList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.MachineDeploymentList{ListMeta: obj.(*v1alpha1.MachineDeploymentList).ListMeta}
	for _, item := range obj.(*v1alpha1.MachineDeploymentList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested machineDeployments.
func (c *FakeMachineDeployments) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(machinedeploymentsResource, c.ns, opts))

}

// Create takes the representation of a machineDeployment and creates it.  Returns the server's representation of the machineDeployment, and an error, if there is any.
func (c *FakeMachineDeployments) Create(ctx context.Context, machineDeployment *v1alpha1.MachineDeployment, opts v1.CreateOptions) (result *v1alpha1.MachineDeployment, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(machinedeploymentsResource, c.ns, machineDeployment), &v1alpha1.MachineDeployment{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MachineDeployment), err
}

// Update takes the representation of a machineDeployment and updates it. Returns the server's representation of the machineDeployment, and an error, if there is any.
func (c *FakeMachineDeployments) Update(ctx context.Context, machineDeployment *v1alpha1.MachineDeployment, opts v1.UpdateOptions) (result *v1alpha1.MachineDeployment, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(machinedeploymentsResource, c.ns, machineDeployment), &v1alpha1.MachineDeployment{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MachineDeployment), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeMachineDeployments) UpdateStatus(ctx context.Context, machineDeployment *v1alpha1.MachineDeployment, opts v1.UpdateOptions) (*v1alpha1.MachineDeployment, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(machinedeploymentsResource, "status", c.ns, machineDeployment), &v1alpha1.MachineDeployment{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MachineDeployment), err
}

// Delete takes name of the machineDeployment and deletes it. Returns an error if one occurs.
func (c *FakeMachineDeployments) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(machinedeploymentsResource, c.ns, name, opts), &v1alpha1.MachineDeployment{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeMachineDeployments) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(machinedeploymentsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.MachineDeploymentList{})
	return err
}

// Patch applies the patch and returns the patched machineDeployment.
func (c *FakeMachineDeployments) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.MachineDeployment, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(machinedeploymentsResource, c.ns, name, pt, data, subresources...), &v1alpha1.MachineDeployment{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.MachineDeployment), err
}

// UpdateScale takes the representation of a scale and updates it. Returns the server's representation of the scale, and an error, if there is any.
func (c *FakeMachineDeployments) UpdateScale(ctx context.Context, machineDeploymentName string, scale *autoscalingv1.Scale, opts v1.UpdateOptions) (result *autoscalingv1.Scale, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(machinedeploymentsResource, "scale", c.ns, scale), &autoscalingv1.Scale{})

	if obj == nil {
		return nil, err
	}
	return obj.(*autoscalingv1.Scale), err
}
