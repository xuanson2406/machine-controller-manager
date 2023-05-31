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

package internalversion

import (
	"github.com/xuanson2406/machine-controller-manager/pkg/client/clientset/internalversion/scheme"
	rest "k8s.io/client-go/rest"
)

type MachineInterface interface {
	RESTClient() rest.Interface
	AWSMachineClassesGetter
	AlicloudMachineClassesGetter
	AzureMachineClassesGetter
	GCPMachineClassesGetter
	MachinesGetter
	MachineClassesGetter
	MachineDeploymentsGetter
	MachineSetsGetter
	MachineTemplatesGetter
	OpenStackMachineClassesGetter
	PacketMachineClassesGetter
}

// MachineClient is used to interact with features provided by the machine.sapcloud.io group.
type MachineClient struct {
	restClient rest.Interface
}

func (c *MachineClient) AWSMachineClasses(namespace string) AWSMachineClassInterface {
	return newAWSMachineClasses(c, namespace)
}

func (c *MachineClient) AlicloudMachineClasses(namespace string) AlicloudMachineClassInterface {
	return newAlicloudMachineClasses(c, namespace)
}

func (c *MachineClient) AzureMachineClasses(namespace string) AzureMachineClassInterface {
	return newAzureMachineClasses(c, namespace)
}

func (c *MachineClient) GCPMachineClasses(namespace string) GCPMachineClassInterface {
	return newGCPMachineClasses(c, namespace)
}

func (c *MachineClient) Machines(namespace string) MachineInterface {
	return newMachines(c, namespace)
}

func (c *MachineClient) MachineClasses(namespace string) MachineClassInterface {
	return newMachineClasses(c, namespace)
}

func (c *MachineClient) MachineDeployments(namespace string) MachineDeploymentInterface {
	return newMachineDeployments(c, namespace)
}

func (c *MachineClient) MachineSets(namespace string) MachineSetInterface {
	return newMachineSets(c, namespace)
}

func (c *MachineClient) MachineTemplates(namespace string) MachineTemplateInterface {
	return newMachineTemplates(c, namespace)
}

func (c *MachineClient) OpenStackMachineClasses(namespace string) OpenStackMachineClassInterface {
	return newOpenStackMachineClasses(c, namespace)
}

func (c *MachineClient) PacketMachineClasses(namespace string) PacketMachineClassInterface {
	return newPacketMachineClasses(c, namespace)
}

// NewForConfig creates a new MachineClient for the given config.
func NewForConfig(c *rest.Config) (*MachineClient, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &MachineClient{client}, nil
}

// NewForConfigOrDie creates a new MachineClient for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *MachineClient {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new MachineClient for the given RESTClient.
func New(c rest.Interface) *MachineClient {
	return &MachineClient{c}
}

func setConfigDefaults(config *rest.Config) error {
	config.APIPath = "/apis"
	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}
	if config.GroupVersion == nil || config.GroupVersion.Group != scheme.Scheme.PrioritizedVersionsForGroup("machine.sapcloud.io")[0].Group {
		gv := scheme.Scheme.PrioritizedVersionsForGroup("machine.sapcloud.io")[0]
		config.GroupVersion = &gv
	}
	config.NegotiatedSerializer = scheme.Codecs

	if config.QPS == 0 {
		config.QPS = 5
	}
	if config.Burst == 0 {
		config.Burst = 10
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *MachineClient) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
