/*
Copyright (c) 2017 SAP SE or an SAP affiliate company. All rights reserved.

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

// Package controller is used to provide the core functionalities of machine-controller-manager
package controller

import (
	"bytes"
	"context"
	"io"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func (c *controller) nodeAdd(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		klog.Errorf("Couldn't get key for object %+v: %v", obj, err)
		return
	}
	c.nodeQueue.Add(key)
}

func (c *controller) nodeUpdate(oldObj, newObj interface{}) {
	c.nodeAdd(newObj)
}

func (c *controller) nodeDelete(obj interface{}) {
	node, ok := obj.(*v1.Node)
	if node == nil || !ok {
		return
	}

}

// Not being used at the moment, saving it for a future use case.
func (c *controller) reconcileClusterNodeKey(key string) error {
	ctx := context.Background()
	node, err := c.nodeLister.Get(key)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		klog.Errorf("ClusterNode %q: Unable to retrieve object from store: %v", key, err)
		return err
	}
	klog.V(4).Infof("Reconcile Node [%s]", node.Name)
	err = c.reconcileClusterNode(ctx, node)
	if err != nil {
		// Re-enqueue after a 30s window
		c.enqueueNodeAfter(node, 30*time.Second)
	} else {
		// Re-enqueue periodically to avoid missing of events
		// TODO: Get ride of this logic
		c.enqueueNodeAfter(node, 3*time.Minute)
	}
	return nil
}

func (c *controller) reconcileClusterNode(ctx context.Context, node *v1.Node) error {
	if node.Labels["worker.fptcloud/type"] == "gpu" {
		list, err := c.controlCoreClient.CoreV1().Secrets(c.namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			klog.Warning(err)
		}
		var kubecfg v1.Secret
		for _, secret := range list.Items {
			if strings.Contains(secret.Name, "user-kubeconfig") {
				kubecfg = secret
				break
			}
		}
		shootConfig := string(kubecfg.Data["kubeconfig"])
		config, err := clientcmd.RESTConfigFromKubeConfig([]byte(shootConfig))
		if err != nil {
			panic(err.Error())
		}

		// Create a clientset using the Config object
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}
		pods, err := clientset.CoreV1().Pods("fptcloud-gpu-operator").List(ctx, metav1.ListOptions{LabelSelector: "app=nvidia-mig-manager"})
		if err != nil {
			return err
		}
		for _, p := range pods.Items {
			if p.Spec.NodeName == node.Name && p.Status.Phase == v1.PodRunning {
				klog.V(4).Infof("pod [%s] in node [%s]", p.Name, node.Name)
				podLog := clientset.CoreV1().Pods("fptcloud-gpu-operator").GetLogs(p.Name,
					&v1.PodLogOptions{Container: "nvidia-mig-manager"})
				log, err := podLog.Stream(ctx)
				if err != nil {
					klog.Warning(err)
				}
				defer log.Close()
				buf := new(bytes.Buffer)
				_, err = io.Copy(buf, log)
				if err != nil {
					klog.Warning(err)
				}
				logOutput := buf.String()
				klog.V(4).Infof("Log of Pod %s: %s", p.Name, logOutput)
				if strings.Contains(logOutput, "error setting MIGConfig: error attempting multiple config orderings: all orderings failed") {
					label := make(map[string]string)
					for k, v := range node.Labels {
						if !strings.Contains(k, "nvidia.com/mig.config") {
							label[k] = v
						}
					}
					// nodeP, _ := c.targetCoreClient.CoreV1().Nodes().Get(ctx, n.Name, metav1.GetOptions{})
					klog.V(4).Infof("Updating label for node %s", node.Name)
					node.Labels = label
					_, err := clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
					if err != nil {
						klog.V(4).Infof("Error when updating node %s: %v", node.Name, err.Error())
						return err
					}
					clientset.CoreV1().Pods("fptcloud-gpu-operator").Delete(ctx, p.Name, metav1.DeleteOptions{})
				}

			}
		}
	}
	return nil
}
func (c *controller) enqueueNodeAfter(obj interface{}, after time.Duration) {
	if toBeEnqueued, key := c.isToBeEnqueued(obj); toBeEnqueued {
		klog.V(5).Infof("Adding node object to the queue %q after %s", key, after)
		c.machineQueue.AddAfter(key, after)
	}
}
