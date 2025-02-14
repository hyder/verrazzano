// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package multiclusterconfigmap

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
)

// Reconciler reconciles a MultiClusterConfigMap object
type Reconciler struct {
	client.Client
	Log          logr.Logger
	Scheme       *runtime.Scheme
	AgentChannel chan clusters.StatusUpdateMessage
}

const finalizerName = "multiclusterconfigmap.verrazzano.io"

// Reconcile reconciles a MultiClusterConfigMap resource. It fetches the embedded ConfigMap,
// mutates it based on the MultiClusterConfigMap, and updates the status of the
// MultiClusterConfigMap to reflect the success or failure of the changes to the embedded resource
// Currently it does NOT support Immutable ConfigMap resources
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("multiclusterconfigmap", req.NamespacedName)
	var mcConfigMap clustersv1alpha1.MultiClusterConfigMap
	result := reconcile.Result{}
	ctx := context.Background()
	err := r.fetchMultiClusterConfigMap(ctx, req.NamespacedName, &mcConfigMap)
	if err != nil {
		return result, clusters.IgnoreNotFoundWithLog("MultiClusterConfigMap", err, logger)
	}

	// delete the wrapped resource since MC is being deleted
	if !mcConfigMap.ObjectMeta.DeletionTimestamp.IsZero() {
		err = clusters.DeleteAssociatedResource(ctx, r.Client, &mcConfigMap, finalizerName, &corev1.ConfigMap{}, types.NamespacedName{Namespace: mcConfigMap.Namespace, Name: mcConfigMap.Name})
		if err != nil {
			logger.Error(err, "Failed to delete associated configmap and finalizer")
		}
		return ctrl.Result{}, err
	}

	oldState := clusters.SetEffectiveStateIfChanged(mcConfigMap.Spec.Placement, &mcConfigMap.Status)
	if !clusters.IsPlacedInThisCluster(ctx, r, mcConfigMap.Spec.Placement) {
		if oldState != mcConfigMap.Status.State {
			// This must be done whether the resource is placed in this cluster or not, because we
			// could be in an admin cluster and receive cluster level statuses from managed clusters,
			// which can change our effective state
			err = r.Status().Update(ctx, &mcConfigMap)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		// if this mc config map is no longer placed on this cluster, remove the associated config map
		err = clusters.DeleteAssociatedResource(ctx, r.Client, &mcConfigMap, finalizerName, &corev1.ConfigMap{}, types.NamespacedName{Namespace: mcConfigMap.Namespace, Name: mcConfigMap.Name})
		return ctrl.Result{}, err
	}

	logger.Info("MultiClusterConfigMap create or update with underlying ConfigMap",
		"ConfigMap", mcConfigMap.Spec.Template.Metadata.Name,
		"placement", mcConfigMap.Spec.Placement.Clusters[0].Name)
	// Immutable ConfigMaps are not supported - we need a webhook to validate, or add the support
	opResult, err := r.createOrUpdateConfigMap(ctx, mcConfigMap)

	// Add our finalizer if not already added
	if err == nil {
		_, err = clusters.AddFinalizer(ctx, r.Client, &mcConfigMap, finalizerName)
	}

	ctrlResult, updateErr := r.updateStatus(ctx, &mcConfigMap, opResult, err)

	// if an error occurred in createOrUpdate, return that error with a requeue
	// even if update status succeeded
	if err != nil {
		res := ctrl.Result{Requeue: true, RequeueAfter: clusters.GetRandomRequeueDelay()}
		return res, err
	}

	return ctrlResult, updateErr
}

// SetupWithManager registers our controller with the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clustersv1alpha1.MultiClusterConfigMap{}).
		Complete(r)
}

func (r *Reconciler) fetchMultiClusterConfigMap(ctx context.Context, name types.NamespacedName, mcConfigMap *clustersv1alpha1.MultiClusterConfigMap) error {
	return r.Get(ctx, name, mcConfigMap)
}

func (r *Reconciler) createOrUpdateConfigMap(ctx context.Context, mcConfigMap clustersv1alpha1.MultiClusterConfigMap) (controllerutil.OperationResult, error) {
	var configMap corev1.ConfigMap
	configMap.Namespace = mcConfigMap.Namespace
	configMap.Name = mcConfigMap.Name

	return controllerutil.CreateOrUpdate(ctx, r.Client, &configMap, func() error {
		r.mutateConfigMap(mcConfigMap, &configMap)
		return nil
	})
}

// mutateConfigMap mutates the K8S ConfigMap to reflect the contents of the parent MultiClusterConfigMap
func (r *Reconciler) mutateConfigMap(mcConfigMap clustersv1alpha1.MultiClusterConfigMap, configMap *corev1.ConfigMap) {
	configMap.Data = mcConfigMap.Spec.Template.Data
	configMap.BinaryData = mcConfigMap.Spec.Template.BinaryData
	configMap.Immutable = mcConfigMap.Spec.Template.Immutable
	configMap.Labels = mcConfigMap.Spec.Template.Metadata.Labels
	configMap.Annotations = mcConfigMap.Spec.Template.Metadata.Annotations
}

func (r *Reconciler) updateStatus(ctx context.Context, mcConfigMap *clustersv1alpha1.MultiClusterConfigMap, opResult controllerutil.OperationResult, err error) (ctrl.Result, error) {
	clusterName := clusters.GetClusterName(ctx, r.Client)
	newCondition := clusters.GetConditionFromResult(err, opResult, "ConfigMap")
	return clusters.UpdateStatus(mcConfigMap, &mcConfigMap.Status, mcConfigMap.Spec.Placement, newCondition, clusterName,
		r.AgentChannel, func() error { return r.Status().Update(ctx, mcConfigMap) })
}
