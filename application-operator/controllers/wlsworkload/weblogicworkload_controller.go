// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package wlsworkload

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	"github.com/go-logr/logr"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	wls "github.com/verrazzano/verrazzano/application-operator/apis/weblogic/v8"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers"
	"github.com/verrazzano/verrazzano/application-operator/controllers/logging"
	"github.com/verrazzano/verrazzano/application-operator/controllers/metricstrait"
	vznav "github.com/verrazzano/verrazzano/application-operator/controllers/navigation"
	istionet "istio.io/api/networking/v1alpha3"
	istioclient "istio.io/client-go/pkg/apis/networking/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	specField                 = "spec"
	destinationRuleAPIVersion = "networking.istio.io/v1alpha3"
	destinationRuleKind       = "DestinationRule"
)

const defaultMonitoringExporterData = `
  {
    "configuration": {
      "domainQualifier": true,
      "metricsNameSnakeCase": true,
      "queries": [
        {
           "key": "name",
           "keyName": "location",
           "prefix": "wls_server_",
           "applicationRuntimes": {
              "key": "name",
              "keyName": "app",
              "componentRuntimes": {
                 "prefix": "wls_webapp_config_",
                 "type": "WebAppComponentRuntime",
                 "key": "name",
                 "values": [
                    "deploymentState",
                    "contextRoot",
                    "sourceInfo",
                    "sessionsOpenedTotalCount",
                    "openSessionsCurrentCount",
                    "openSessionsHighCount"
                 ],
                 "servlets": {
                    "prefix": "wls_servlet_",
                    "key": "servletName"
                 }
              }
           }
        },
        {
           "JVMRuntime": {
              "prefix": "wls_jvm_",
              "key": "name"
           }
        },
        {
           "executeQueueRuntimes": {
              "prefix": "wls_socketmuxer_",
              "key": "name",
              "values": [
                 "pendingRequestCurrentCount"
              ]
           }
        },
        {
           "workManagerRuntimes": {
              "prefix": "wls_workmanager_",
              "key": "name",
              "values": [
                 "stuckThreadCount",
                 "pendingRequests",
                 "completedRequests"
              ]
           }
        },
        {
           "threadPoolRuntime": {
              "prefix": "wls_threadpool_",
              "key": "name",
              "values": [
                 "executeThreadTotalCount",
                 "queueLength",
                 "stuckThreadCount",
                 "hoggingThreadCount"
              ]
           }
        },
        {
           "JMSRuntime": {
              "key": "name",
              "keyName": "jmsruntime",
              "prefix": "wls_jmsruntime_",
              "JMSServers": {
                 "prefix": "wls_jms_",
                 "key": "name",
                 "keyName": "jmsserver",
                 "destinations": {
                    "prefix": "wls_jms_dest_",
                    "key": "name",
                    "keyName": "destination"
                 }
              }
           }
        },
        {
           "persistentStoreRuntimes": {
              "prefix": "wls_persistentstore_",
              "key": "name"
           }
        },
        {
           "JDBCServiceRuntime": {
              "JDBCDataSourceRuntimeMBeans": {
                 "prefix": "wls_datasource_",
                 "key": "name"
              }
           }
        },
        {
           "JTARuntime": {
              "prefix": "wls_jta_",
              "key": "name"
           }
        }
      ]
    },
    "imagePullPolicy": "IfNotPresent"
  }
`

var specServerPodFields = []string{specField, "serverPod"}
var specServerPodLabelsFields = append(specServerPodFields, "labels")
var specServerPodContainersFields = append(specServerPodFields, "containers")
var specServerPodVolumesFields = append(specServerPodFields, "volumes")
var specServerPodVolumeMountsFields = append(specServerPodFields, "volumeMounts")
var specConfigurationIstioEnabledFields = []string{specField, "configuration", "istio", "enabled"}
var specConfigurationRuntimeEncryptionSecret = []string{specField, "configuration", "model", "runtimeEncryptionSecret"}
var specMonitoringExporterFields = []string{specField, "monitoringExporter"}

// this struct allows us to extract information from the unstructured WebLogic spec
// so we can interface with the FLUENTD code
type containersMountsVolumes struct {
	Containers   []corev1.Container
	Volumes      []corev1.Volume
	VolumeMounts []corev1.VolumeMount
}

// Reconciler reconciles a VerrazzanoWebLogicWorkload object
type Reconciler struct {
	client.Client
	Log     logr.Logger
	Scheme  *runtime.Scheme
	Metrics *metricstrait.Reconciler
}

// SetupWithManager registers our controller with the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vzapi.VerrazzanoWebLogicWorkload{}).
		Complete(r)
}

// Reconcile reconciles a VerrazzanoWebLogicWorkload resource. It fetches the embedded WebLogic CR, mutates it to add
// scopes and traits, and then writes out the CR (or deletes it if the workload is being deleted).
// +kubebuilder:rbac:groups=oam.verrazzano.io,resources=verrazzanoweblogicworkloads,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=oam.verrazzano.io,resources=verrazzanoweblogicworkloads/status,verbs=get;update;patch
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("verrazzanoweblogicworkload", req.NamespacedName)
	log.Info("Reconciling verrazzano weblogic workload")

	// fetch the workload and unwrap the WebLogic resource
	workload, err := r.fetchWorkload(ctx, req.NamespacedName)
	if err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	u, err := vznav.ConvertRawExtensionToUnstructured(&workload.Spec.Template)
	if err != nil {
		return reconcile.Result{}, err
	}

	// make sure the namespace is set to the namespace of the component
	if err = unstructured.SetNestedField(u.Object, req.NamespacedName.Namespace, "metadata", "namespace"); err != nil {
		return reconcile.Result{}, err
	}

	// the embedded resource doesn't have an API version or kind, so add them
	gvk := vznav.APIVersionAndKindToContainedGVK(workload.APIVersion, workload.Kind)
	if gvk == nil {
		return reconcile.Result{}, errors.New("unable to determine contained GroupVersionKind for workload")
	}

	apiVersion, kind := gvk.ToAPIVersionAndKind()
	u.SetAPIVersion(apiVersion)
	u.SetKind(kind)

	// mutate the WebLogic domain resource, copy labels, add logging, etc.
	if err = copyLabels(log, workload.ObjectMeta.GetLabels(), u); err != nil {
		return reconcile.Result{}, err
	}

	// Attempt to get the existing Domain. This is used in the case where we don't want to update the Fluentd image.
	// In this case we obtain the previous Fluentd image and set that on the new Domain.
	var existingDomain wls.Domain
	domainKey := types.NamespacedName{Name: u.GetName(), Namespace: workload.Namespace}
	if err := r.Get(ctx, domainKey, &existingDomain); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("No existing domain found")
		} else {
			log.Error(err, "An error occurred trying to obtain an existing domain")
			return reconcile.Result{}, err
		}
	}
	// upgradeApp indicates whether the user has indicated that it is ok to update the application to use the latest
	// resource values from Verrazzano. An example of this is the Fluentd image used by logging.
	upgradeApp := controllers.IsWorkloadMarkedForUpgrade(workload.Labels, workload.Status.CurrentUpgradeVersion)

	// Add the Fluentd sidecar container required for logging to the Domain
	if err = r.addLogging(ctx, log, workload, upgradeApp, u, &existingDomain); err != nil {
		return reconcile.Result{}, err
	}

	// Add the monitoringExporter to the spec if not already present
	if err = addDefaultMonitoringExporter(u); err != nil {
		return reconcile.Result{}, err
	}

	// Get the namespace resource that the VerrazzanoWebLogicWorkload resource is deployed to
	namespace := &corev1.Namespace{}
	if err = r.Client.Get(ctx, client.ObjectKey{Namespace: "", Name: req.NamespacedName.Namespace}, namespace); err != nil {
		return reconcile.Result{}, err
	}

	// Set the domain resource configuration.istio.enabled value
	if err = updateIstioEnabled(namespace.Labels, u); err != nil {
		return reconcile.Result{}, err
	}

	// set controller reference so the WebLogic domain CR gets deleted when the workload is deleted
	if err = controllerutil.SetControllerReference(workload, u, r.Scheme); err != nil {
		log.Error(err, "Unable to set controller ref")
		return reconcile.Result{}, err
	}

	// create the RuntimeEncryptionSecret if specified and the secret does not exist
	secret, found, err := unstructured.NestedString(u.Object, specConfigurationRuntimeEncryptionSecret...)
	if err != nil {
		return reconcile.Result{}, err
	}
	if found {
		err = r.createRuntimeEncryptionSecret(ctx, log, namespace.Name, secret, workload.ObjectMeta.Labels)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// make a copy of the WebLogic spec since u.Object will get overwritten in CreateOrUpdate
	// if the WebLogic CR exists
	specCopy, _, err := unstructured.NestedFieldCopy(u.Object, specField)
	if err != nil {
		log.Error(err, "Unable to make a copy of the WebLogic spec")
		return reconcile.Result{}, err
	}

	// write out the WebLogic resource
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, u, func() error {
		return unstructured.SetNestedField(u.Object, specCopy, specField)
	})
	if err != nil {
		log.Error(err, "Error creating or updating WebLogic CR")
		return reconcile.Result{}, err
	}

	if err = r.createDestinationRule(ctx, log, namespace.Name, namespace.Labels, workload.ObjectMeta.Labels); err != nil {
		return reconcile.Result{}, err
	}

	if err = r.updateUpgradeVersionInStatus(ctx, workload); err != nil {
		return reconcile.Result{}, err
	}

	log.Info("Successfully created WebLogic domain")
	return reconcile.Result{}, nil
}

// fetchWorkload fetches the VerrazzanoWebLogicWorkload data given a namespaced name
func (r *Reconciler) fetchWorkload(ctx context.Context, name types.NamespacedName) (*vzapi.VerrazzanoWebLogicWorkload, error) {
	var workload vzapi.VerrazzanoWebLogicWorkload
	if err := r.Get(ctx, name, &workload); err != nil {
		if k8serrors.IsNotFound(err) {
			r.Log.Info("VerrazzanoWebLogicWorkload has been deleted", "name", name)
		} else {
			r.Log.Error(err, "Failed to fetch VerrazzanoWebLogicWorkload", "name", name)
		}
		return nil, err
	}

	return &workload, nil
}

// copyLabels copies specific labels from the Verrazzano workload to the contained WebLogic resource
func copyLabels(log logr.Logger, workloadLabels map[string]string, weblogic *unstructured.Unstructured) error {
	// the WebLogic domain spec/serverPod/labels field has labels that get propagated to the pods
	labels, found, _ := unstructured.NestedStringMap(weblogic.Object, specServerPodLabelsFields...)
	if !found {
		labels = map[string]string{}
	}

	// copy the oam component and app name labels
	if componentName, ok := workloadLabels[oam.LabelAppComponent]; ok {
		labels[oam.LabelAppComponent] = componentName
	}

	if appName, ok := workloadLabels[oam.LabelAppName]; ok {
		labels[oam.LabelAppName] = appName
	}

	// Set the label indicating this is WebLogic workload
	labels[constants.LabelWorkloadType] = constants.WorkloadTypeWeblogic

	err := unstructured.SetNestedStringMap(weblogic.Object, labels, specServerPodLabelsFields...)
	if err != nil {
		log.Error(err, "Unable to set labels in spec serverPod")
		return err
	}
	return nil
}

// addLogging adds a FLUENTD sidecar and updates the WebLogic spec if there is an associated LogInfo
func (r *Reconciler) addLogging(ctx context.Context, log logr.Logger, workload *vzapi.VerrazzanoWebLogicWorkload, upgradeApp bool, weblogic *unstructured.Unstructured, existingDomain *wls.Domain) error {
	// If the Domain already exists and we don't want to update the Fluentd image, obtain the Fluentd image from the
	// current Domain
	var existingFluentdImage string
	if !upgradeApp {
		for _, container := range existingDomain.Spec.ServerPod.Containers {
			if container.Name == logging.FluentdStdoutSidecarName {
				existingFluentdImage = container.Image
				break
			}
		}
	}

	// if we're running in a managed cluster, use the multicluster ES URL and secret, and if we're
	// not the fields will be empty and we will set these fields to defaults below
	scope, err := logging.NewLogInfo(existingFluentdImage)
	if err != nil {
		return err
	}

	if scope == nil {
		log.Info("No logging scope found for workload, nothing to do")
		return nil
	}

	// extract just enough of the WebLogic data into concrete types so we can merge with
	// the FLUENTD data
	var extracted containersMountsVolumes
	if serverPod, found, _ := unstructured.NestedMap(weblogic.Object, specServerPodFields...); found {
		if err = runtime.DefaultUnstructuredConverter.FromUnstructured(serverPod, &extracted); err != nil {
			return errors.New("unable to extract containers, volumes, and volume mounts from WebLogic spec")
		}
	}

	name, found, _ := unstructured.NestedString(weblogic.Object, "metadata", "name")
	if !found {
		return errors.New("expected to find metadata name in WebLogic spec")
	}

	// fluentdPod starts with what's in the spec and we add in the FLUENTD things when Apply is
	// called on the fluentdManager
	fluentdPod := &logging.FluentdPod{
		Containers:   extracted.Containers,
		Volumes:      extracted.Volumes,
		VolumeMounts: extracted.VolumeMounts,
		LogPath:      logging.BuildWLSLogPath(name),
		HandlerEnv:   logging.GetWlsSpecificContainerEnv(),
	}
	fluentdManager := logging.GetFluentd(ctx, r.Log, r.Client)

	// fluentdManager.Apply wants a QRR but it only cares about the namespace (at least for
	// this use case)
	resource := vzapi.QualifiedResourceRelation{Namespace: workload.Namespace}

	// note that this call has the side effect of creating a FLUENTD config map if one
	// does not already exist in the namespace
	if _, err = fluentdManager.Apply(scope, resource, fluentdPod); err != nil {
		return err
	}

	// convert the containers, volumes, and mounts in fluentdPod to unstructured and set
	// the values in the spec
	fluentdPodUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(fluentdPod)
	if err != nil {
		return err
	}

	err = unstructured.SetNestedSlice(weblogic.Object, fluentdPodUnstructured["containers"].([]interface{}), specServerPodContainersFields...)
	if err != nil {
		log.Error(err, "Unable to set serverPod containers")
		return err
	}
	err = unstructured.SetNestedSlice(weblogic.Object, fluentdPodUnstructured["volumes"].([]interface{}), specServerPodVolumesFields...)
	if err != nil {
		log.Error(err, "Unable to set serverPod volumes")
		return err
	}
	err = unstructured.SetNestedField(weblogic.Object, fluentdPodUnstructured["volumeMounts"].([]interface{}), specServerPodVolumeMountsFields...)
	if err != nil {
		log.Error(err, "Unable to set serverPod volumeMounts")
		return err
	}

	// logHome and logHomeEnabled fields need to be set to turn on logging
	err = unstructured.SetNestedField(weblogic.Object, logging.BuildWLSLogHome(name), specField, "logHome")
	if err != nil {
		log.Error(err, "Unable to set logHome")
		return err
	}
	err = unstructured.SetNestedField(weblogic.Object, true, specField, "logHomeEnabled")
	if err != nil {
		log.Error(err, "Unable to set logHomeEnabled")
		return err
	}

	return nil
}

// createRuntimeEncryptionSecret creates the runtimeEncryptionSecret specified in the domain spec if it does not exist.
func (r *Reconciler) createRuntimeEncryptionSecret(ctx context.Context, log logr.Logger, namespaceName string, secretName string, workloadLabels map[string]string) error {
	appName, ok := workloadLabels[oam.LabelAppName]
	if !ok {
		return errors.New("OAM app name label missing from metadata, unable to create owner reference to appconfig")
	}

	// Create the secret if it does not already exist
	secret := &corev1.Secret{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespaceName, Name: secretName}, secret)
	if err != nil && k8serrors.IsNotFound(err) {
		thePassword, err := genPassword(128)
		if err != nil {
			return err
		}
		secret = &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespaceName,
				Name:      secretName,
			},
			Data: map[string][]byte{
				"password": []byte(thePassword),
			},
		}

		// Set the owner reference.
		appConfig := &v1alpha2.ApplicationConfiguration{}
		err = r.Get(context.TODO(), types.NamespacedName{Namespace: namespaceName, Name: appName}, appConfig)
		if err != nil {
			return err
		}
		err = controllerutil.SetControllerReference(appConfig, secret, r.Scheme)
		if err != nil {
			return err
		}

		log.Info(fmt.Sprintf("Creating secret %s:%s", namespaceName, secretName))
		err = r.Create(ctx, secret)
		if err != nil {
			return err
		}

	} else if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Secret %s:%s already exist", namespaceName, secretName))

	return nil
}

// createDestinationRule creates an Istio destinationRule required by WebLogic servers.
// The destinationRule is only created when the namespace has the label istio-injection=enabled.
func (r *Reconciler) createDestinationRule(ctx context.Context, log logr.Logger, namespace string, namespaceLabels map[string]string, workloadLabels map[string]string) error {
	istioEnabled := false
	value, ok := namespaceLabels["istio-injection"]
	if ok && value == "enabled" {
		istioEnabled = true
	}

	if !istioEnabled {
		return nil
	}

	appName, ok := workloadLabels[oam.LabelAppName]
	if !ok {
		return errors.New("OAM app name label missing from metadata, unable to generate destination rule name")
	}

	// Create a destination rule if it does not already exist
	destinationRule := &istioclient.DestinationRule{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: appName}, destinationRule)
	if err != nil && k8serrors.IsNotFound(err) {
		destinationRule = &istioclient.DestinationRule{
			TypeMeta: metav1.TypeMeta{
				APIVersion: destinationRuleAPIVersion,
				Kind:       destinationRuleKind},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      appName,
			},
		}
		destinationRule.Spec.Host = fmt.Sprintf("*.%s.svc.cluster.local", namespace)
		destinationRule.Spec.TrafficPolicy = &istionet.TrafficPolicy{
			Tls: &istionet.ClientTLSSettings{
				Mode: istionet.ClientTLSSettings_ISTIO_MUTUAL,
			},
		}

		// Set the owner reference.
		appConfig := &v1alpha2.ApplicationConfiguration{}
		err := r.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: appName}, appConfig)
		if err != nil {
			return err
		}
		err = controllerutil.SetControllerReference(appConfig, destinationRule, r.Scheme)
		if err != nil {
			return err
		}

		log.Info(fmt.Sprintf("Creating Istio destination rule %s:%s", namespace, appName))
		err = r.Create(ctx, destinationRule)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Istio destination rule %s:%s already exist", namespace, appName))

	return nil
}

func (r *Reconciler) updateUpgradeVersionInStatus(ctx context.Context, workload *vzapi.VerrazzanoWebLogicWorkload) error {
	if workload.Labels[constants.LabelUpgradeVersion] != workload.Status.CurrentUpgradeVersion {
		workload.Status.CurrentUpgradeVersion = workload.Labels[constants.LabelUpgradeVersion]
		return r.Status().Update(ctx, workload)
	}
	return nil
}

// updateIstioEnabled sets the domain resource configuration.istio.enabled value based
// on the namespace label istio-injection
func updateIstioEnabled(labels map[string]string, u *unstructured.Unstructured) error {
	istioEnabled := false
	value, ok := labels["istio-injection"]
	if ok && value == "enabled" {
		istioEnabled = true
	}

	return unstructured.SetNestedField(u.Object, istioEnabled, specConfigurationIstioEnabledFields...)
}

func genPassword(passSize int) (string, error) {
	const passwordChars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	result := make([]byte, passSize)
	for i := 0; i < passSize; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(passwordChars))))
		if err != nil {
			return "", err
		}
		result[i] = passwordChars[num.Int64()]
	}
	return string(result), nil
}

// addDefaultMonitoringExporter adds monitoringExporter to the WebLogic spec if there is not one present
func addDefaultMonitoringExporter(weblogic *unstructured.Unstructured) error {
	if _, found, _ := unstructured.NestedFieldNoCopy(weblogic.Object, specMonitoringExporterFields...); !found {
		defaultMonitoringExporter, err := getDefaultMonitoringExporter()
		if err != nil {
			return err
		}
		err = unstructured.SetNestedField(weblogic.Object, defaultMonitoringExporter, specMonitoringExporterFields...)
		if err != nil {
			return err
		}
	}
	return nil
}

func getDefaultMonitoringExporter() (interface{}, error) {
	bytes := []byte(defaultMonitoringExporterData)
	var monitoringExporter map[string]interface{}
	json.Unmarshal(bytes, &monitoringExporter)
	result, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&monitoringExporter)
	if err != nil {
		return nil, err
	}
	return result, nil
}
