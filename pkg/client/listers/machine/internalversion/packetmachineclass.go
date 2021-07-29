/*
Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file

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

// Code generated by lister-gen. DO NOT EDIT.

package internalversion

import (
	machine "github.com/gardener/machine-controller-manager/pkg/apis/machine"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// PacketMachineClassLister helps list PacketMachineClasses.
// All objects returned here must be treated as read-only.
type PacketMachineClassLister interface {
	// List lists all PacketMachineClasses in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*machine.PacketMachineClass, err error)
	// PacketMachineClasses returns an object that can list and get PacketMachineClasses.
	PacketMachineClasses(namespace string) PacketMachineClassNamespaceLister
	PacketMachineClassListerExpansion
}

// packetMachineClassLister implements the PacketMachineClassLister interface.
type packetMachineClassLister struct {
	indexer cache.Indexer
}

// NewPacketMachineClassLister returns a new PacketMachineClassLister.
func NewPacketMachineClassLister(indexer cache.Indexer) PacketMachineClassLister {
	return &packetMachineClassLister{indexer: indexer}
}

// List lists all PacketMachineClasses in the indexer.
func (s *packetMachineClassLister) List(selector labels.Selector) (ret []*machine.PacketMachineClass, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*machine.PacketMachineClass))
	})
	return ret, err
}

// PacketMachineClasses returns an object that can list and get PacketMachineClasses.
func (s *packetMachineClassLister) PacketMachineClasses(namespace string) PacketMachineClassNamespaceLister {
	return packetMachineClassNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// PacketMachineClassNamespaceLister helps list and get PacketMachineClasses.
// All objects returned here must be treated as read-only.
type PacketMachineClassNamespaceLister interface {
	// List lists all PacketMachineClasses in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*machine.PacketMachineClass, err error)
	// Get retrieves the PacketMachineClass from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*machine.PacketMachineClass, error)
	PacketMachineClassNamespaceListerExpansion
}

// packetMachineClassNamespaceLister implements the PacketMachineClassNamespaceLister
// interface.
type packetMachineClassNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all PacketMachineClasses in the indexer for a given namespace.
func (s packetMachineClassNamespaceLister) List(selector labels.Selector) (ret []*machine.PacketMachineClass, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*machine.PacketMachineClass))
	})
	return ret, err
}

// Get retrieves the PacketMachineClass from the indexer for a given namespace and name.
func (s packetMachineClassNamespaceLister) Get(name string) (*machine.PacketMachineClass, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(machine.Resource("packetmachineclass"), name)
	}
	return obj.(*machine.PacketMachineClass), nil
}
