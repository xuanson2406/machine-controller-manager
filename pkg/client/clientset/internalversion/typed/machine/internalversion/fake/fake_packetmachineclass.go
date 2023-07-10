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

	machine "github.com/xuanson2406/machine-controller-manager/pkg/apis/machine"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakePacketMachineClasses implements PacketMachineClassInterface
type FakePacketMachineClasses struct {
	Fake *FakeMachine
	ns   string
}

var packetmachineclassesResource = schema.GroupVersionResource{Group: "machine.sapcloud.io", Version: "", Resource: "packetmachineclasses"}

var packetmachineclassesKind = schema.GroupVersionKind{Group: "machine.sapcloud.io", Version: "", Kind: "PacketMachineClass"}

// Get takes name of the packetMachineClass, and returns the corresponding packetMachineClass object, and an error if there is any.
func (c *FakePacketMachineClasses) Get(ctx context.Context, name string, options v1.GetOptions) (result *machine.PacketMachineClass, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(packetmachineclassesResource, c.ns, name), &machine.PacketMachineClass{})

	if obj == nil {
		return nil, err
	}
	return obj.(*machine.PacketMachineClass), err
}

// List takes label and field selectors, and returns the list of PacketMachineClasses that match those selectors.
func (c *FakePacketMachineClasses) List(ctx context.Context, opts v1.ListOptions) (result *machine.PacketMachineClassList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(packetmachineclassesResource, packetmachineclassesKind, c.ns, opts), &machine.PacketMachineClassList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &machine.PacketMachineClassList{ListMeta: obj.(*machine.PacketMachineClassList).ListMeta}
	for _, item := range obj.(*machine.PacketMachineClassList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested packetMachineClasses.
func (c *FakePacketMachineClasses) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(packetmachineclassesResource, c.ns, opts))

}

// Create takes the representation of a packetMachineClass and creates it.  Returns the server's representation of the packetMachineClass, and an error, if there is any.
func (c *FakePacketMachineClasses) Create(ctx context.Context, packetMachineClass *machine.PacketMachineClass, opts v1.CreateOptions) (result *machine.PacketMachineClass, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(packetmachineclassesResource, c.ns, packetMachineClass), &machine.PacketMachineClass{})

	if obj == nil {
		return nil, err
	}
	return obj.(*machine.PacketMachineClass), err
}

// Update takes the representation of a packetMachineClass and updates it. Returns the server's representation of the packetMachineClass, and an error, if there is any.
func (c *FakePacketMachineClasses) Update(ctx context.Context, packetMachineClass *machine.PacketMachineClass, opts v1.UpdateOptions) (result *machine.PacketMachineClass, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(packetmachineclassesResource, c.ns, packetMachineClass), &machine.PacketMachineClass{})

	if obj == nil {
		return nil, err
	}
	return obj.(*machine.PacketMachineClass), err
}

// Delete takes name of the packetMachineClass and deletes it. Returns an error if one occurs.
func (c *FakePacketMachineClasses) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(packetmachineclassesResource, c.ns, name), &machine.PacketMachineClass{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakePacketMachineClasses) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(packetmachineclassesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &machine.PacketMachineClassList{})
	return err
}

// Patch applies the patch and returns the patched packetMachineClass.
func (c *FakePacketMachineClasses) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *machine.PacketMachineClass, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(packetmachineclassesResource, c.ns, name, pt, data, subresources...), &machine.PacketMachineClass{})

	if obj == nil {
		return nil, err
	}
	return obj.(*machine.PacketMachineClass), err
}
