/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

This file was copied and modified from the kubernetes/kubernetes project
https://github.com/kubernetes/kubernetes/release-1.8/pkg/controller/deployment/deployment_controller.go

Modifications Copyright (c) 2017 SAP SE or an SAP affiliate company. All rights reserved.
*/

// Package controller is used to provide the core functionalities of machine-controller-manager
package controller

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"k8s.io/klog/v2"

	"github.com/xuanson2406/machine-controller-manager/pkg/apis/machine"
	"github.com/xuanson2406/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/xuanson2406/machine-controller-manager/pkg/apis/machine/validation"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"
)

// controllerKind contains the schema.GroupVersionKind for this controller type.
var controllerKind = v1alpha1.SchemeGroupVersion.WithKind("MachineDeployment")

// GroupVersionKind is the version kind used to identify objects managed by machine-controller-manager
var GroupVersionKind = "machine.sapcloud.io/v1alpha1"

func (dc *controller) addMachineDeployment(obj interface{}) {
	d := obj.(*v1alpha1.MachineDeployment)
	klog.V(4).Infof("Adding machine deployment %s", d.Name)
	dc.enqueueMachineDeployment(d)
}

func (dc *controller) updateMachineDeployment(old, cur interface{}) {
	oldD := old.(*v1alpha1.MachineDeployment)
	curD := cur.(*v1alpha1.MachineDeployment)
	klog.V(4).Infof("Updating machine deployment %s", oldD.Name)
	dc.enqueueMachineDeployment(curD)
}

func (dc *controller) deleteMachineDeployment(obj interface{}) {
	d, ok := obj.(*v1alpha1.MachineDeployment)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			//utilruntime.HandleError(fmt.Errorf("Couldn't get object from tombstone %#v", obj))
			return
		}
		d, ok = tombstone.Obj.(*v1alpha1.MachineDeployment)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("Tombstone contained object that is not a Machine Deployment %#v", obj))
			return
		}
	}
	klog.V(4).Infof("Deleting machine deployment %s", d.Name)
	dc.enqueueMachineDeployment(d)
}

// addMachineSet enqueues the deployment that manages a MachineSet when the MachineSet is created.
func (dc *controller) addMachineSetToDeployment(obj interface{}) {
	is := obj.(*v1alpha1.MachineSet)

	if is.DeletionTimestamp != nil {
		// On a restart of the controller manager, it's possible for an object to
		// show up in a state that is already pending deletion.
		dc.deleteMachineSetToDeployment(is)
		return
	}

	// If it has a ControllerRef, that's all that matters.
	if controllerRef := metav1.GetControllerOf(is); controllerRef != nil {
		d := dc.resolveDeploymentControllerRef(is.Namespace, controllerRef)
		if d == nil {
			return
		}
		klog.V(4).Infof("MachineSet %s added.", is.Name)
		dc.enqueueMachineDeployment(d)
		return
	}

	// Otherwise, it's an orphan. Get a list of all matching Deployments and sync
	// them to see if anyone wants to adopt it.
	ds := dc.getMachineDeploymentsForMachineSet(is)
	if len(ds) == 0 {
		return
	}
	klog.V(4).Infof("Orphan MachineSet %s added.", is.Name)
	for _, d := range ds {
		dc.enqueueMachineDeployment(d)
	}
}

// getDeploymentsForMachineSet returns a list of Deployments that potentially
// match a MachineSet.
func (dc *controller) getMachineDeploymentsForMachineSet(machineSet *v1alpha1.MachineSet) []*v1alpha1.MachineDeployment {
	deployments, err := dc.GetMachineDeploymentsForMachineSet(machineSet)
	if err != nil || len(deployments) == 0 {
		return nil
	}
	// Because all MachineSet's belonging to a deployment should have a unique label key,
	// there should never be more than one deployment returned by the above method.
	// If that happens we should probably dynamically repair the situation by ultimately
	// trying to clean up one of the controllers, for now we just return the older one
	if len(deployments) > 1 {
		// ControllerRef will ensure we don't do anything crazy, but more than one
		// item in this list nevertheless constitutes user error.
		klog.V(4).Infof("user error! more than one deployment is selecting machine set %s with labels: %#v, returning %s",
			machineSet.Name, machineSet.Labels, deployments[0].Name)
	}
	return deployments
}

// updateMachineSet figures out what deployment(s) manage a MachineSet when the MachineSet
// is updated and wake them up. If the anything of the MachineSets have changed, we need to
// awaken both the old and new deployments. old and cur must be *extensions.MachineSet
// types.
func (dc *controller) updateMachineSetToDeployment(old, cur interface{}) {
	curMachineSet := cur.(*v1alpha1.MachineSet)
	oldMachineSet := old.(*v1alpha1.MachineSet)
	if curMachineSet.ResourceVersion == oldMachineSet.ResourceVersion {
		// Periodic resync will send update events for all known machine sets.
		// Two different versions of the same machine set will always have different RVs.
		return
	}

	curControllerRef := metav1.GetControllerOf(curMachineSet)
	oldControllerRef := metav1.GetControllerOf(oldMachineSet)
	controllerRefChanged := !reflect.DeepEqual(curControllerRef, oldControllerRef)
	if controllerRefChanged && oldControllerRef != nil {
		// The ControllerRef was changed. Sync the old controller, if any.
		if d := dc.resolveDeploymentControllerRef(oldMachineSet.Namespace, oldControllerRef); d != nil {
			dc.enqueueMachineDeployment(d)
		}
	}

	// If it has a ControllerRef, that's all that matters.
	if curControllerRef != nil {
		d := dc.resolveDeploymentControllerRef(curMachineSet.Namespace, curControllerRef)
		if d == nil {
			return
		}
		klog.V(4).Infof("MachineSet %s updated.", curMachineSet.Name)
		dc.enqueueMachineDeployment(d)
		return
	}

	// Otherwise, it's an orphan. If anything changed, sync matching controllers
	// to see if anyone wants to adopt it now.
	labelChanged := !reflect.DeepEqual(curMachineSet.Labels, oldMachineSet.Labels)
	if labelChanged || controllerRefChanged {
		ds := dc.getMachineDeploymentsForMachineSet(curMachineSet)
		if len(ds) == 0 {
			return
		}
		klog.V(4).Infof("Orphan MachineSet %s updated.", curMachineSet.Name)
		for _, d := range ds {
			dc.enqueueMachineDeployment(d)
		}
	}
}

// deleteMachineSet enqueues the deployment that manages a MachineSet when
// the MachineSet is deleted. obj could be an *v1alpha1.MachineSet, or
// a DeletionFinalStateUnknown marker item.
func (dc *controller) deleteMachineSetToDeployment(obj interface{}) {
	machineSet, ok := obj.(*v1alpha1.MachineSet)

	// When a delete is dropped, the relist will notice a Machine in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the MachineSet
	// changed labels the new deployment will not be woken up till the periodic resync.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("Couldn't get object from tombstone %#v", obj))
			return
		}
		machineSet, ok = tombstone.Obj.(*v1alpha1.MachineSet)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("Tombstone contained object that is not a MachineSet %#v", obj))
			return
		}
	}

	controllerRef := metav1.GetControllerOf(machineSet)
	if controllerRef == nil {
		// No controller should care about orphans being deleted.
		return
	}
	d := dc.resolveDeploymentControllerRef(machineSet.Namespace, controllerRef)
	if d == nil {
		return
	}
	klog.V(4).Infof("MachineSet %s deleted.", machineSet.Name)
	dc.enqueueMachineDeployment(d)
}

// deleteMachine will enqueue a Recreate Deployment once all of its Machines have stopped running.
func (dc *controller) deleteMachineToMachineDeployment(ctx context.Context, obj interface{}) {
	machine, ok := obj.(*v1alpha1.Machine)

	// When a delete is dropped, the relist will notice a Machine in the store not
	// in the list, leading to the insertion of a tombstone object which contains
	// the deleted key/value. Note that this value might be stale. If the Machine
	// changed labels the new deployment will not be woken up till the periodic resync.
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("Couldn't get object from tombstone %#v", obj))
			return
		}
		machine, ok = tombstone.Obj.(*v1alpha1.Machine)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("Tombstone contained object that is not a machine %#v", obj))
			return
		}
	}
	klog.V(4).Infof("Machine %s deleted.", machine.Name)
	if d := dc.getMachineDeploymentForMachine(ctx, machine); d != nil && d.Spec.Strategy.Type == v1alpha1.RecreateMachineDeploymentStrategyType {
		// Sync if this Deployment now has no more Machines.
		machineSets, err := ListMachineSets(d, IsListFromClient(ctx, dc.controlMachineClient))
		if err != nil {
			return
		}
		machineMap, err := dc.getMachineMapForMachineDeployment(d, machineSets)
		if err != nil {
			return
		}
		numMachines := 0
		for _, machineList := range machineMap {
			numMachines += len(machineList.Items)
		}
		if numMachines == 0 {
			dc.enqueueMachineDeployment(d)
		}
	}
}

func (dc *controller) enqueueMachineDeployment(deployment *v1alpha1.MachineDeployment) {
	key, err := KeyFunc(deployment)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for object %#v: %v", deployment, err))
		return
	}

	dc.machineDeploymentQueue.Add(key)
}

func (dc *controller) enqueueRateLimited(deployment *v1alpha1.MachineDeployment) {
	key, err := KeyFunc(deployment)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for object %#v: %v", deployment, err))
		return
	}

	dc.machineDeploymentQueue.AddRateLimited(key)
}

// enqueueMachineDeploymentAfter will enqueue a deployment after the provided amount of time.
func (dc *controller) enqueueMachineDeploymentAfter(deployment *v1alpha1.MachineDeployment, after time.Duration) {
	key, err := KeyFunc(deployment)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for object %#v: %v", deployment, err))
		return
	}

	dc.machineDeploymentQueue.AddAfter(key, after)
}

// getDeploymentForMachine returns the deployment managing the given Machine.
func (dc *controller) getMachineDeploymentForMachine(ctx context.Context, machine *v1alpha1.Machine) *v1alpha1.MachineDeployment {
	// Find the owning machine set
	var is *v1alpha1.MachineSet
	var err error
	controllerRef := metav1.GetControllerOf(machine)
	if controllerRef == nil {
		// No controller owns this Machine.
		return nil
	}
	if controllerRef.Kind != "MachineSet" { //TODO: Remove hardcoded string
		// Not a Machine owned by a machine set.
		return nil
	}
	is, err = dc.controlMachineClient.MachineSets(machine.Namespace).Get(ctx, controllerRef.Name, metav1.GetOptions{})
	if err != nil || is.UID != controllerRef.UID {
		klog.V(4).Infof("Cannot get machineset %q for machine %q: %v", controllerRef.Name, machine.Name, err)
		return nil
	}

	// Now find the Deployment that owns that MachineSet.
	controllerRef = metav1.GetControllerOf(is)
	if controllerRef == nil {
		return nil
	}
	return dc.resolveDeploymentControllerRef(is.Namespace, controllerRef)
}

// resolveControllerRef returns the controller referenced by a ControllerRef,
// or nil if the ControllerRef could not be resolved to a matching controller
// of the correct Kind.
func (dc *controller) resolveDeploymentControllerRef(namespace string, controllerRef *metav1.OwnerReference) *v1alpha1.MachineDeployment {
	// We can't look up by UID, so look up by Name and then verify UID.
	// Don't even try to look up by Name if it's the wrong Kind.
	if controllerRef.Kind != controllerKind.Kind {
		return nil
	}
	d, err := dc.controlMachineClient.MachineDeployments(namespace).Get(context.TODO(), controllerRef.Name, metav1.GetOptions{})
	if err != nil {
		return nil
	}
	if d.UID != controllerRef.UID {
		// The controller we found with this Name is not the same one that the
		// ControllerRef points to.
		return nil
	}
	return d
}

// TODO: Remove this method later if not required., it is not being used anywhere.
func (dc *controller) handleErr(err error, key interface{}) {
	if err == nil {
		dc.machineDeploymentQueue.Forget(key)
		return
	}

	if dc.machineDeploymentQueue.NumRequeues(key) < maxRetries {
		klog.V(2).Infof("Error syncing deployment %v: %v", key, err)
		dc.machineDeploymentQueue.AddRateLimited(key)
		return
	}

	utilruntime.HandleError(err)
	klog.V(2).Infof("Dropping deployment %q out of the queue: %v", key, err)
	dc.machineDeploymentQueue.Forget(key)
}

// getMachineSetsForDeployment uses ControllerRefManager to reconcile
// ControllerRef by adopting and orphaning.
// It returns the list of MachineSets that this Deployment should manage.
func (dc *controller) getMachineSetsForMachineDeployment(ctx context.Context, d *v1alpha1.MachineDeployment) ([]*v1alpha1.MachineSet, error) {
	// List all MachineSets to find those we own but that no longer match our
	// selector. They will be orphaned by ClaimMachineSets().
	machineSets, err := dc.machineSetLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	deploymentSelector, err := metav1.LabelSelectorAsSelector(d.Spec.Selector)
	if err != nil {
		return nil, fmt.Errorf("machine deployment %s has invalid label selector: %v", d.Name, err)
	}
	// If any adoptions are attempted, we should first recheck for deletion with
	// an uncached quorum read sometime after listing MachineSets (see #42639).
	canAdoptFunc := RecheckDeletionTimestamp(func() (metav1.Object, error) {
		fresh, err := dc.controlMachineClient.MachineDeployments(d.Namespace).Get(ctx, d.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if fresh.UID != d.UID {
			return nil, fmt.Errorf("original Machine Deployment %v is gone: got uid %v, wanted %v", d.Name, fresh.UID, d.UID)
		}
		return fresh, nil
	})
	cm := NewMachineSetControllerRefManager(dc.machineSetControl, d, deploymentSelector, controllerKind, canAdoptFunc)
	ISes, err := cm.ClaimMachineSets(ctx, machineSets)
	return ISes, err
}

// getMachineMapForDeployment returns the Machines managed by a Deployment.
//
// It returns a map from MachineSet UID to a list of Machines controlled by that RS,
// according to the Machine's ControllerRef.
func (dc *controller) getMachineMapForMachineDeployment(d *v1alpha1.MachineDeployment, machineSets []*v1alpha1.MachineSet) (map[types.UID]*v1alpha1.MachineList, error) {
	// Get all Machines that potentially belong to this Deployment.
	selector, err := metav1.LabelSelectorAsSelector(d.Spec.Selector)
	if err != nil {
		return nil, err
	}
	machines, err := dc.machineLister.List(selector)
	if err != nil {
		return nil, err
	}
	// Group Machines by their controller (if it's in rsList).
	machineMap := make(map[types.UID]*v1alpha1.MachineList, len(machineSets))
	for _, is := range machineSets {
		machineMap[is.UID] = &v1alpha1.MachineList{}
	}
	for _, machine := range machines {
		// Do not ignore inactive Machines because Recreate Deployments need to verify that no
		// Machines from older versions are running before spinning up new Machines.
		controllerRef := metav1.GetControllerOf(machine)
		if controllerRef == nil {
			continue
		}
		// Only append if we care about this UID.
		if machineList, ok := machineMap[controllerRef.UID]; ok {
			machineList.Items = append(machineList.Items, *machine)
		}
	}
	return machineMap, nil
}

// reconcileClusterMachineDeployment will sync the deployment with the given key.
// This function is not meant to be invoked concurrently with the same key.
func (dc *controller) reconcileClusterMachineDeployment(key string) error {
	ctx := context.Background()
	startTime := time.Now()
	klog.V(4).Infof("Started syncing machine deployment %q (%v)", key, startTime)
	defer func() {
		klog.V(4).Infof("Finished syncing machine deployment %q (%v)", key, time.Since(startTime))
	}()

	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	deployment, err := dc.controlMachineClient.MachineDeployments(dc.namespace).Get(ctx, name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		klog.V(4).Infof("Deployment %v has been deleted", key)
		return nil
	}
	if err != nil {
		return err
	}

	klog.V(3).Infof("Processing the machinedeployment %q (with replicas %d)", deployment.Name, deployment.Spec.Replicas)

	// If MachineDeployment is frozen and no deletion timestamp, don't process it
	if deployment.Labels["freeze"] == "True" && deployment.DeletionTimestamp == nil {
		klog.V(3).Infof("MachineDeployment %q is frozen. However, it will still be processed if it there is an scale down event.", deployment.Name)
	}

	// Validate MachineDeployment
	internalMachineDeployment := &machine.MachineDeployment{}

	err = v1alpha1.Convert_v1alpha1_MachineDeployment_To_machine_MachineDeployment(deployment, internalMachineDeployment, nil)
	if err != nil {
		return err
	}

	validationerr := validation.ValidateMachineDeployment(internalMachineDeployment)
	if validationerr.ToAggregate() != nil && len(validationerr.ToAggregate().Errors()) > 0 {
		klog.Errorf("Validation of MachineDeployment failed %s", validationerr.ToAggregate().Error())
		return nil
	}

	// Resync the MachineDeployment after 10 minutes to avoid missing out on missed out events
	defer dc.enqueueMachineDeploymentAfter(deployment, 10*time.Minute)

	// Deep-copy otherwise we are mutating our cache.
	// TODO: Deep-copy only when needed.
	d := deployment.DeepCopy()

	// Manipulate finalizers
	if d.DeletionTimestamp == nil {
		dc.addMachineDeploymentFinalizers(ctx, d)
	}

	everything := metav1.LabelSelector{}
	if reflect.DeepEqual(d.Spec.Selector, &everything) {
		dc.recorder.Eventf(d, v1.EventTypeWarning, "SelectingAll", "This deployment is selecting all machines. A non-empty selector is required.")
		if d.Status.ObservedGeneration < d.Generation {
			d.Status.ObservedGeneration = d.Generation
			dc.controlMachineClient.MachineDeployments(d.Namespace).UpdateStatus(ctx, d, metav1.UpdateOptions{})
		}
		return nil
	}

	// List MachineSets owned by this Deployment, while reconciling ControllerRef
	// through adoption/orphaning.
	machineSets, err := dc.getMachineSetsForMachineDeployment(ctx, d)
	if err != nil {
		return err
	}
	// List all Machines owned by this Deployment, grouped by their MachineSet.
	// Current uses of the MachineMap are:
	//
	// * check if a Machine is labeled correctly with the Machine-template-hash label.
	// * check that no old Machines are running in the middle of Recreate Deployments.
	machineMap, err := dc.getMachineMapForMachineDeployment(d, machineSets)
	if err != nil {
		return err
	}

	if d.DeletionTimestamp != nil {
		if finalizers := sets.NewString(d.Finalizers...); !finalizers.Has(DeleteFinalizerName) {
			return nil
		}
		if len(machineSets) == 0 {
			dc.deleteMachineDeploymentFinalizers(ctx, d)
			return nil
		}
		klog.V(4).Infof("Deleting all child MachineSets as MachineDeployment %s has set deletionTimestamp", d.Name)
		dc.terminateMachineSets(ctx, machineSets, d)
		return dc.syncStatusOnly(ctx, d, machineSets, machineMap)
	}

	// Update deployment conditions with an Unknown condition when pausing/resuming
	// a deployment. In this way, we can be sure that we won't timeout when a user
	// resumes a Deployment with a set progressDeadlineSeconds.
	if err = dc.checkPausedConditions(ctx, d); err != nil {
		return err
	}

	if d.Spec.Paused {
		return dc.sync(ctx, d, machineSets, machineMap)
	}

	// rollback is not re-entrant in case the underlying machine sets are updated with a new
	// revision so we should ensure that we won't proceed to update machine sets until we
	// make sure that the deployment has cleaned up its rollback spec in subsequent enqueues.
	if d.Spec.RollbackTo != nil {
		return dc.rollback(ctx, d, machineSets, machineMap)
	}

	scalingEvent, err := dc.isScalingEvent(ctx, d, machineSets, machineMap)

	if err != nil {
		return err
	}
	if scalingEvent {
		return dc.sync(ctx, d, machineSets, machineMap)
	}

	switch d.Spec.Strategy.Type {
	case v1alpha1.RecreateMachineDeploymentStrategyType:
		return dc.rolloutRecreate(ctx, d, machineSets, machineMap)
	case v1alpha1.RollingUpdateMachineDeploymentStrategyType:
		return dc.rolloutRolling(ctx, d, machineSets, machineMap)
	}
	return fmt.Errorf("unexpected deployment strategy type: %s", d.Spec.Strategy.Type)
}

func (dc *controller) terminateMachineSets(ctx context.Context, machineSets []*v1alpha1.MachineSet, deployment *v1alpha1.MachineDeployment) {
	var (
		wg               sync.WaitGroup
		numOfMachinesets = len(machineSets)
	)
	wg.Add(numOfMachinesets)

	for _, machineSet := range machineSets {
		go func(machineSet *v1alpha1.MachineSet) {
			defer wg.Done()
			// Machine is already marked as 'to-be-deleted'
			if machineSet.DeletionTimestamp != nil {
				return
			}
			dc.controlMachineClient.MachineSets(machineSet.Namespace).Delete(ctx, machineSet.Name, metav1.DeleteOptions{})
		}(machineSet)
	}
	wg.Wait()
}

/*
	SECTION
	Manipulate Finalizers
*/

func (dc *controller) addMachineDeploymentFinalizers(ctx context.Context, machineDeployment *v1alpha1.MachineDeployment) {
	clone := machineDeployment.DeepCopy()

	if finalizers := sets.NewString(clone.Finalizers...); !finalizers.Has(DeleteFinalizerName) {
		finalizers.Insert(DeleteFinalizerName)
		dc.updateMachineDeploymentFinalizers(ctx, clone, finalizers.List())
	}
}

func (dc *controller) deleteMachineDeploymentFinalizers(ctx context.Context, machineDeployment *v1alpha1.MachineDeployment) {
	clone := machineDeployment.DeepCopy()

	if finalizers := sets.NewString(clone.Finalizers...); finalizers.Has(DeleteFinalizerName) {
		finalizers.Delete(DeleteFinalizerName)
		dc.updateMachineDeploymentFinalizers(ctx, clone, finalizers.List())
	}
}

func (dc *controller) updateMachineDeploymentFinalizers(ctx context.Context, machineDeployment *v1alpha1.MachineDeployment, finalizers []string) {
	// Get the latest version of the machineDeployment so that we can avoid conflicts
	machineDeployment, err := dc.controlMachineClient.MachineDeployments(machineDeployment.Namespace).Get(ctx, machineDeployment.Name, metav1.GetOptions{})
	if err != nil {
		return
	}

	clone := machineDeployment.DeepCopy()
	clone.Finalizers = finalizers
	_, err = dc.controlMachineClient.MachineDeployments(machineDeployment.Namespace).Update(ctx, clone, metav1.UpdateOptions{})
	if err != nil {
		// Keep retrying until update goes through
		klog.Warning("Updated failed, retrying")
		dc.updateMachineDeploymentFinalizers(ctx, machineDeployment, finalizers)
	}
}

const (
	GPUChartAlreadyInstalled = 0
	GPUChartNotInstalled     = 1
	InstallPrometheusStack   = 2
	InstallPrometheusAdapter = 3
)

// ProviderSpec is the spec to be used while parsing the calls.
type ProviderSpec struct {
	ClusterName        string `json:"cluster,omitempty"`
	WorkerPoolName     string `json:"workerPoolName,omitempty"`
	NetworkName        string `json:"networkName,omitempty"`
	IPMode             string `json:"ipMode,omitempty"`
	NetworkAdapterType string `json:"networkAdapterType,omitempty"`
	IsPrimary          bool   `json:"isPrimary,omitempty"`
	VappTemplate       string `json:"vAppTemplateName,omitempty"`
	Catalog            string `json:"catalogName,omitempty"`
	VCPU               int    `json:"vCPU,omitempty"`
	VGPU               string `json:"vGPU,omitempty"`
	VGpuId             string `json:"vGpuId,omitempty"`
	RAM                int    `json:"ram,omitempty"`
	Disk               int    `json:"disk,omitempty"`
	PublicKey          string `json:"publicKey,omitempty"`
	Storagepolicy      string `json:"storagePolicy,omitempty"`
	ApiUrl             string `json:"apiUrl,omitempty"`
	VpcID              string `json:"vpcID,omitempty"`
	BackendPortalToken string `json:"backendPortalToken,omitempty"`
	VpcToken           string `json:"vpcToken,omitempty"`
	Zone               string `json:"zone,omitempty"` // this field will use in future version
}

// func (c *controller) CheckGPUWorkerGroup(ctx context.Context, machinedeployment *v1alpha1.MachineDeployment) (bool, error) {

// 	machineClassInterface := c.controlMachineClient.MachineClasses(c.namespace)
// 	machineClass, err := machineClassInterface.Get(ctx, machinedeployment.Spec.Template.Spec.Class.Name, metav1.GetOptions{})
// 	if err != nil {
// 		klog.Errorf("MachineClass %s/%s not found. Skipping. %v", c.namespace, machineClass.Name, err)
// 		return false, err
// 	}
// 	providerSpec, err := DecodeProviderSpecFromMachineClass(machineClass)
// 	if err != nil {
// 		return false, err
// 	}
// 	if providerSpec.VGPU == "gpu" {
// 		klog.V(3).Infof("Worker group %s is enable GPU - waiting for install GPU chart to shoot", machinedeployment.Name)
// 		return true, nil
// 	}
// 	return false, nil
// }

// func (c *controller) InstallChartForShoot(ctx context.Context, machineDeployment *v1alpha1.MachineDeployment) error {
// 	var (
// 		url       = "https://registry.fke.fptcloud.com/chartrepo/xplat-fke"
// 		repoName  = "xplat-fke"
// 		shootName = machineDeployment.Namespace
// 		strategy  string
// 	)
// 	EnableGPU, err := c.CheckGPUWorkerGroup(ctx, machineDeployment)
// 	if err != nil {
// 		return err
// 	}
// 	if !EnableGPU {
// 		klog.V(3).Infof("Shoot cluster %s is disable GPU - skipping install helm chart to shoot\n", shootName)
// 		return nil
// 	}
// 	kubeconfigFile := os.Getenv("HOME") + "/kubeconfig/" + shootName
// 	err = os.MkdirAll(filepath.Dir(kubeconfigFile), os.ModePerm)
// 	if err != nil && !os.IsExist(err) {
// 		return fmt.Errorf("Could not create dir to store chart repository: [%v]", err)
// 	}
// 	kubeconfigSecret, err := c.controlCoreClient.CoreV1().Secrets(c.namespace).List(ctx, metav1.ListOptions{})
// 	if err != nil {
// 		return err
// 	}
// 	for _, secret := range kubeconfigSecret.Items {
// 		if strings.Contains(secret.Name, "user-kubeconfig") {
// 			err = ioutil.WriteFile(kubeconfigFile, []byte(string(secret.Data["kubeconfig"])), os.ModePerm)
// 			if err != nil {
// 				return fmt.Errorf("Could not create file to save kubeconfig of cluster %s: [%v]", shootName, err)
// 			}
// 		}
// 	}

// 	settings := CreateSetting("", kubeconfigFile)
// 	cleanUp := c.CleanReleaseFail(kubeconfigFile)
// 	if cleanUp != nil {
// 		return fmt.Errorf("Unable to clean Release fail in shoot cluster [%s]: %v", shootName, cleanUp)
// 	}
// 	checkInstalled, err := c.CheckChartInstalled(settings)
// 	if err != nil {
// 		return fmt.Errorf("Unable to check release GPU is installed in cluster %s: [%v]", shootName, err)
// 	}
// 	if checkInstalled == GPUChartAlreadyInstalled {
// 		klog.Infof("Shoot cluster %s have already installed charts GPU - skipping install helm chart to shoot\n", shootName)
// 		return nil
// 	}
// 	klog.V(3).Infof("Shoot cluster %s is enabled GPU - starting install helm chart to shoot\n", shootName)

// 	label := c.getLabelWorkerGroup(ctx, machineDeployment)
// 	if label == nil {
// 		strategy = "mig.strategy=none"
// 	} else {
// 		switch value := label["nvidia.com/mig.config"]; value {
// 		case "all-1g.6gb":
// 			strategy = "mig.strategy=single"
// 		case "all-2g.12gb":
// 			strategy = "mig.strategy=single"
// 		case "all-balanced":
// 			strategy = "mig.strategy=mixed"
// 		}
// 	}
// 	// Add helm repo
// 	repoAdd(repoName, url, settings)
// 	// Update charts from the helm repo
// 	repoUpdate(settings)
// 	if checkInstalled == GPUChartNotInstalled {
// 		// Install GPU Operator
// 		err = c.InstallGPUOperatorChart(repoName, strategy, kubeconfigFile)
// 		if err != nil {
// 			return fmt.Errorf("Unable to install chart gpu-operator to cluster %s: [%v]", shootName, err)
// 		}
// 		// Install kube-prometheus-stack
// 		err = c.InstallPrometheusStackChart(repoName, kubeconfigFile)
// 		if err != nil {
// 			return fmt.Errorf("Unable to install chart kube-prometheus-stack to cluster %s: [%v]", shootName, err)
// 		}
// 		// Install prometheus-adapter
// 		err = c.InstallPrometheusAdapterChart(repoName, kubeconfigFile)
// 		if err != nil {
// 			return fmt.Errorf("Unable to install chart prometheus-adapter to cluster %s: [%v]", shootName, err)
// 		}
// 	}
// 	if checkInstalled == InstallPrometheusStack {
// 		err = c.InstallPrometheusStackChart(repoName, kubeconfigFile)
// 		if err != nil {
// 			return fmt.Errorf("Unable to install chart kube-prometheus-stack to cluster %s: [%v]", shootName, err)
// 		}
// 		err = c.InstallPrometheusAdapterChart(repoName, kubeconfigFile)
// 		if err != nil {
// 			return fmt.Errorf("Unable to install chart prometheus-adapter to cluster %s: [%v]", shootName, err)
// 		}
// 	}
// 	if checkInstalled == InstallPrometheusAdapter {
// 		err = c.InstallPrometheusAdapterChart(repoName, kubeconfigFile)
// 		if err != nil {
// 			return fmt.Errorf("Unable to install chart prometheus-adapter to cluster %s: [%v]", shootName, err)
// 		}
// 	}
// 	return nil
// }
// func (c *controller) InstallGPUOperatorChart(repoName, value, kubeconfigFile string) error {
// 	klog.V(4).Infof("Strategy of GPU operator: %s", value)
// 	settings := CreateSetting("gpu-operator", kubeconfigFile)
// 	err := c.installChart(repoName, "gpu-operator", value, settings)
// 	time.Sleep(30 * time.Second)
// 	return err
// }
// func (c *controller) InstallPrometheusStackChart(repoName, kubeconfigFile string) error {
// 	settings := CreateSetting("prometheus", kubeconfigFile)
// 	err := c.installChart(repoName, "kube-prometheus-stack", "", settings)
// 	time.Sleep(45 * time.Second)
// 	return err
// }
// func (c *controller) InstallPrometheusAdapterChart(repoName, kubeconfigFile string) error {
// 	settings := CreateSetting("prometheus", kubeconfigFile)
// 	service, err := c.controlCoreClient.CoreV1().Services("prometheus").List(context.TODO(), metav1.ListOptions{
// 		LabelSelector: "app=kube-prometheus-stack-prometheus",
// 	})
// 	if err != nil {
// 		return fmt.Errorf("Unable to get svc with selector in cluster %s: [%v]", c.namespace, err)
// 	}
// 	prometheus_service := service.Items[0].Name
// 	value := "prometheus.url=http://" + prometheus_service + ".prometheus.svc.cluster.local"
// 	err = c.installChart(repoName, "prometheus-adapter", value, settings)
// 	time.Sleep(15 * time.Second)
// 	return err
// }

// func CreateSetting(namespace string, kubeconfig string) *cli.EnvSettings {
// 	os.Setenv("HELM_NAMESPACE", namespace)
// 	settings := cli.New()
// 	settings.KubeConfig = kubeconfig
// 	return settings
// }

// RepoAdd adds repo with given name and url
// func repoAdd(name, url string, settings *cli.EnvSettings) error {
// 	repoFile := settings.RepositoryConfig

// 	//Ensure the file directory exists as it is required for file locking
// 	err := os.MkdirAll(filepath.Dir(repoFile), os.ModePerm)
// 	if err != nil && !os.IsExist(err) {
// 		return fmt.Errorf("Could not create dir to store chart repository: %v", err)
// 	}

// 	// Acquire a file lock for process synchronization
// 	fileLock := flock.New(strings.Replace(repoFile, filepath.Ext(repoFile), ".lock", 1))
// 	lockCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
// 	defer cancel()
// 	locked, err := fileLock.TryLockContext(lockCtx, time.Second)
// 	if err == nil && locked {
// 		defer fileLock.Unlock()
// 	}
// 	if err != nil {
// 		return fmt.Errorf("Could not acquire a file lock for process synchronization: %v", err)
// 	}

// 	b, err := ioutil.ReadFile(repoFile)
// 	if err != nil && !os.IsNotExist(err) {
// 		return fmt.Errorf("Could not read chart repository file: %v", err)
// 	}

// 	var f repo.File
// 	if err := yaml.Unmarshal(b, &f); err != nil {
// 		return fmt.Errorf("Unable to unmarshal repository file: %v", err)
// 	}

// 	if f.Has(name) {
// 		// fmt.Printf("repository name (%s) already exists\n", name)
// 		klog.Infof("repository name (%s) already exists\n", name)
// 		return nil
// 	}

// 	c := repo.Entry{
// 		Name: name,
// 		URL:  url,
// 	}

// 	r, err := repo.NewChartRepository(&c, getter.All(settings))
// 	if err != nil {
// 		return fmt.Errorf("Could not construct chart repository: %v", err)
// 	}

// 	if _, err := r.DownloadIndexFile(); err != nil {
// 		err := charterrors.Wrapf(err, "looks like %q is not a valid chart repository or cannot be reached", url)
// 		return fmt.Errorf("Could not reach repository file: %v", err)
// 	}

// 	f.Update(&c)

// 	if err := f.WriteFile(repoFile, 0644); err != nil {
// 		return fmt.Errorf("Could not write content to chart repository file: %v", err)
// 	}
// 	// fmt.Printf("%q has been added to your repositories\n", name)
// 	return nil
// }

// // RepoUpdate updates charts for all helm repos
// func repoUpdate(settings *cli.EnvSettings) error {
// 	repoFile := settings.RepositoryConfig

// 	f, err := repo.LoadFile(repoFile)
// 	if os.IsNotExist(charterrors.Cause(err)) || len(f.Repositories) == 0 {
// 		return fmt.Errorf("Chart not exist: %v", err)
// 	}
// 	var repos []*repo.ChartRepository
// 	for _, cfg := range f.Repositories {
// 		r, err := repo.NewChartRepository(cfg, getter.All(settings))
// 		if err != nil {
// 			return fmt.Errorf("Unable to construct chart repository: %s\n", err.Error())
// 		}
// 		repos = append(repos, r)
// 	}

// 	// fmt.Printf("Hang tight while we grab the latest from your chart repositories...\n")
// 	var wg sync.WaitGroup
// 	for _, re := range repos {
// 		wg.Add(1)
// 		go func(re *repo.ChartRepository) {
// 			defer wg.Done()
// 			if _, err := re.DownloadIndexFile(); err != nil {
// 				klog.Errorf("...Unable to get an update from the %q chart repository (%s):\n\t%v", re.Config.Name, re.Config.URL, err.Error())
// 			} else {
// 				klog.Infof("...Successfully got an update from the %q chart repository\n", re.Config.Name)
// 			}
// 		}(re)
// 	}
// 	wg.Wait()
// 	// fmt.Printf("Update Complete. ⎈ Happy Helming!⎈\n")
// 	klog.Infof("Update Complete. ⎈ Happy Helming!⎈\n")
// 	return nil
// }

// // InstallChart
// func (c *controller) installChart(repo, chart, value string, settings *cli.EnvSettings) error {
// 	actionConfig := new(action.Configuration)
// 	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), debug); err != nil {
// 		// log.Fatal(err)
// 		return fmt.Errorf("Unable to init the action configuration: %v", err)
// 	}
// 	client := action.NewInstall(actionConfig)

// 	if client.Version == "" && client.Devel {
// 		client.Version = ">0.0.0-0"
// 	}
// 	//name, chart, err := client.NameAndChart(args)
// 	rand.Seed(time.Now().UnixNano())
// 	client.ReleaseName = fmt.Sprintf("%s-%d", chart, rand.Intn(1000000000))
// 	client.CreateNamespace = true
// 	client.GenerateName = true

// 	cp, err := client.ChartPathOptions.LocateChart(fmt.Sprintf("%s/%s", repo, chart), settings)
// 	if err != nil {
// 		// log.Fatal(err)
// 		return fmt.Errorf("Unable to locate chart repository: %v", err)
// 	}

// 	debug("CHART PATH: %s\n", cp)

// 	p := getter.All(settings)
// 	valueOpts := &values.Options{}
// 	vals, err := valueOpts.MergeValues(p)
// 	if err != nil {
// 		// log.Fatal(err)
// 		return fmt.Errorf("Unable to merge value from file values.yaml to chart: %v", err)
// 	}
// 	args := map[string]string{
// 		"set": value,
// 	}
// 	// Add args
// 	if value != "" {
// 		if err := strvals.ParseInto(args["set"], vals); err != nil {
// 			// log.Fatal(errors.Wrap(err, "failed parsing --set data"))
// 			return fmt.Errorf("Unable to parse argument to chart: %v", err)
// 		}
// 	}

// 	// Check chart dependencies to make sure all are present in /charts
// 	chartRequested, err := loader.Load(cp)
// 	if err != nil {
// 		return fmt.Errorf("Unable to load chart from chart path: %v", err)
// 	}

// 	validInstallableChart, err := isChartInstallable(chartRequested)
// 	if !validInstallableChart {
// 		// log.Fatal(err)
// 		return fmt.Errorf("Unable to install chart into shoot cluster: %v", err)
// 	}

// 	if req := chartRequested.Metadata.Dependencies; req != nil {
// 		// If CheckDependencies returns an error, we have unfulfilled dependencies.
// 		// As of Helm 2.4.0, this is treated as a stopping condition:
// 		// https://github.com/helm/helm/issues/2209
// 		if err := action.CheckDependencies(chartRequested, req); err != nil {
// 			if client.DependencyUpdate {
// 				man := &downloader.Manager{
// 					Out:              os.Stdout,
// 					ChartPath:        cp,
// 					Keyring:          client.ChartPathOptions.Keyring,
// 					SkipUpdate:       false,
// 					Getters:          p,
// 					RepositoryConfig: settings.RepositoryConfig,
// 					RepositoryCache:  settings.RepositoryCache,
// 				}
// 				if err := man.Update(); err != nil {
// 					return fmt.Errorf("Unable to update manager: %v", err)
// 				}
// 			} else {
// 				// log.Fatal(err)
// 				return fmt.Errorf("Unable to update dependency: %v", err)
// 			}
// 		}
// 	}

// 	client.Namespace = settings.Namespace()
// 	release, err := client.Run(chartRequested, vals)
// 	if err != nil {
// 		if err.Error() == "cannot re-use a name that is still in use" {
// 			klog.Infof("Chart %s already installed in shoot cluster", chart)
// 			return nil
// 		} else {
// 			return fmt.Errorf("Unable to install chart to shoot cluster %s: [%v]", c.namespace, err)
// 		}
// 	} else {
// 		klog.Infof("Successed install chart [%s] to shoot cluster [%s] by release [%s] !", chart, c.namespace, release.Name)
// 	}
// 	return nil
// }

// func UnInstallChart(name string, settings *cli.EnvSettings) error {
// 	actionConfig := new(action.Configuration)
// 	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), debug); err != nil {
// 		return fmt.Errorf("Unable to initialize action config to uninstall release [%s]: %v", name, err)
// 	}
// 	client := action.NewUninstall(actionConfig)
// 	_, err := client.Run(name)
// 	if err != nil {
// 		return fmt.Errorf("Unable to uninstall release [%s]: %v", name, err)
// 	}
// 	klog.Infof("Release %s has been uninstalled\n", name)
// 	return nil
// }

// func isChartInstallable(ch *hchart.Chart) (bool, error) {
// 	switch ch.Metadata.Type {
// 	case "", "application":
// 		return true, nil
// 	}
// 	return false, charterrors.Errorf("%s charts are not installable", ch.Metadata.Type)
// }

// func debug(format string, v ...interface{}) {
// 	format = fmt.Sprintf("[debug] %s\n", format)
// 	log.Output(2, fmt.Sprintf(format, v...))
// }
// func (c *controller) CheckChartInstalled(settings *cli.EnvSettings) (int, error) {
// 	actionConfig := new(action.Configuration)
// 	if err := actionConfig.Init(settings.RESTClientGetter(), "", os.Getenv("HELM_DRIVER"), debug); err != nil {
// 		return -1, fmt.Errorf("Unable to init action config to list releases in shoot cluster %s: [%v]", c.namespace, err)
// 	}
// 	client := action.NewList(actionConfig)

// 	client.AllNamespaces = true
// 	client.Deployed = true
// 	release, err := client.Run()
// 	if err != nil {
// 		return -1, fmt.Errorf("Unable to list releases in shoot cluster %s: [%v]\n", c.namespace, err)
// 	}
// 	var CASE = GPUChartNotInstalled
// 	for _, r := range release {
// 		if strings.Contains(r.Name, "prometheus-adapter") {
// 			CASE = GPUChartAlreadyInstalled
// 			break
// 		}
// 		if strings.Contains(r.Name, "gpu-operator") {
// 			CASE = InstallPrometheusStack
// 		}
// 		if strings.Contains(r.Name, "kube-prometheus-stack") {
// 			CASE = InstallPrometheusAdapter
// 		}
// 	}
// 	return CASE, nil
// }
// func (c *controller) CleanReleaseFail(kubeconfigFile string) error {
// 	settings := CreateSetting("", kubeconfigFile)
// 	actionConfig := new(action.Configuration)
// 	if err := actionConfig.Init(settings.RESTClientGetter(), "", os.Getenv("HELM_DRIVER"), debug); err != nil {
// 		return fmt.Errorf("Unable to init action config to list releases in shoot cluster %s: [%v]", c.namespace, err)
// 	}
// 	client := action.NewList(actionConfig)

// 	client.AllNamespaces = true
// 	// client.Failed = true
// 	// client.Pending = true
// 	// client.Deployed = false
// 	release, err := client.Run()
// 	if err != nil {
// 		return fmt.Errorf("Unable to list releases in shoot cluster %s: [%v]", c.namespace, err)
// 	}
// 	for _, r := range release {
// 		if r.Info.Status != "deployed" {
// 			settings = CreateSetting(r.Namespace, kubeconfigFile)
// 			err = UnInstallChart(r.Name, settings)
// 			if err != nil {
// 				klog.Infof("Uninstall release [%s] in namespace [%s] fail", r.Name, r.Namespace)
// 				return err
// 			}
// 		}
// 	}
// 	return nil
// }

// func DecodeProviderSpecFromMachineClass(machineClass *v1alpha1.MachineClass) (*ProviderSpec, error) {
// 	// Extract providerSpec
// 	var providerSpec *ProviderSpec

// 	if machineClass == nil {
// 		return nil, charterrors.New("MachineClass provided is nil")
// 	}

// 	jsonErr := json.Unmarshal(machineClass.ProviderSpec.Raw, &providerSpec)
// 	if jsonErr != nil {
// 		return nil, fmt.Errorf("Failed to parse JSON data provided as ProviderSpec: %v", jsonErr)
// 	}

// 	return providerSpec, nil
// }
// func (c *controller) getLabelWorkerGroup(ctx context.Context, machineDeployment *v1alpha1.MachineDeployment) map[string]string {
// 	// machineDeployment := c.getMachineDeploymentForMachine(ctx, machine)
// 	label := make(map[string]string)
// 	label = machineDeployment.Spec.Template.Spec.NodeTemplateSpec.Labels
// 	return label
// }
