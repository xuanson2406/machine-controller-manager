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

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/xuanson2406/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// OpenStackMachineClassLister helps list OpenStackMachineClasses.
type OpenStackMachineClassLister interface {
	// List lists all OpenStackMachineClasses in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.OpenStackMachineClass, err error)
	// OpenStackMachineClasses returns an object that can list and get OpenStackMachineClasses.
	OpenStackMachineClasses(namespace string) OpenStackMachineClassNamespaceLister
	OpenStackMachineClassListerExpansion
}

// openStackMachineClassLister implements the OpenStackMachineClassLister interface.
type openStackMachineClassLister struct {
	indexer cache.Indexer
}

// NewOpenStackMachineClassLister returns a new OpenStackMachineClassLister.
func NewOpenStackMachineClassLister(indexer cache.Indexer) OpenStackMachineClassLister {
	return &openStackMachineClassLister{indexer: indexer}
}

// List lists all OpenStackMachineClasses in the indexer.
func (s *openStackMachineClassLister) List(selector labels.Selector) (ret []*v1alpha1.OpenStackMachineClass, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.OpenStackMachineClass))
	})
	return ret, err
}

// OpenStackMachineClasses returns an object that can list and get OpenStackMachineClasses.
func (s *openStackMachineClassLister) OpenStackMachineClasses(namespace string) OpenStackMachineClassNamespaceLister {
	return openStackMachineClassNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// OpenStackMachineClassNamespaceLister helps list and get OpenStackMachineClasses.
type OpenStackMachineClassNamespaceLister interface {
	// List lists all OpenStackMachineClasses in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.OpenStackMachineClass, err error)
	// Get retrieves the OpenStackMachineClass from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.OpenStackMachineClass, error)
	OpenStackMachineClassNamespaceListerExpansion
}

// openStackMachineClassNamespaceLister implements the OpenStackMachineClassNamespaceLister
// interface.
type openStackMachineClassNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all OpenStackMachineClasses in the indexer for a given namespace.
func (s openStackMachineClassNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.OpenStackMachineClass, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.OpenStackMachineClass))
	})
	return ret, err
}

// Get retrieves the OpenStackMachineClass from the indexer for a given namespace and name.
func (s openStackMachineClassNamespaceLister) Get(name string) (*v1alpha1.OpenStackMachineClass, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("openstackmachineclass"), name)
	}
	return obj.(*v1alpha1.OpenStackMachineClass), nil
}
