// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package installjob

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// TestNipIoInstallDefaults tests the creation of an nip.io install default configuration
// GIVEN a verrazzano.install.verrazzano.io custom resource
//  WHEN I call GetInstallConfig
//  THEN the nip.io install configuration is created and verified
func TestNipIoInstallDefaults(t *testing.T) {
	vz := installv1alpha1.Verrazzano{}
	config, err := GetInstallConfig(&vz)
	assert.NoError(t, err)
	assert.Equalf(t, "default", config.EnvironmentName, "Expected environment name did not match")
	assert.Equalf(t, installv1alpha1.Prod, config.Profile, "Expected profile did not match")
	assert.Equalf(t, DNSTypeWildcard, config.DNS.Type, "Expected DNS type did not match")
	assert.Equalf(t, IngressTypeLoadBalancer, config.Ingress.Type, "Expected Ingress type did not match")
	assert.Equalf(t, CertIssuerTypeCA, config.Certificates.IssuerType, "Expected certification issuer type did not match")
	assert.Equalf(t, "cert-manager", config.Certificates.CA.ClusterResourceNamespace, "Expected namespace did not match")
	assert.Equalf(t, "verrazzano-ca-certificate-secret", config.Certificates.CA.SecretName, "Expected CA secret name did not match")
	assert.Equalf(t, 0, len(config.Keycloak.KeycloakInstallArgs), "Expected keycloakInstallArgs length did not match")
	assert.Equalf(t, 0, len(config.Keycloak.MySQL.MySQLInstallArgs), "Expected mySqlInstallArgs length did not match")
	assert.Equalf(t, "true", config.Keycloak.Enabled, "Expected Keycloak enabled did not match")
	assert.Equalf(t, "true", config.Rancher.Enabled, "Expected Rancher enabled did not match")
	assert.Equalf(t, "true", config.Fluentd.Enabled, "Expected Fluentd enabled did not match")
}

// TestNipIoInstallNonDefaults tests the creation of an nip.io install non-default configuration
// GIVEN a verrazzano.install.verrazzano.io custom resource
//  WHEN I call GetInstallConfig
//  THEN the nip.io install configuration is created and verified
func TestNipIoInstallNonDefaults(t *testing.T) {
	enabled := false
	vz := installv1alpha1.Verrazzano{
		Spec: installv1alpha1.VerrazzanoSpec{
			Profile:         "dev",
			EnvironmentName: "testEnv",
			Components: installv1alpha1.ComponentSpec{
				DNS: &installv1alpha1.DNSComponent{
					Wildcard: &installv1alpha1.Wildcard{
						Domain: "nip.io",
					},
				},
				Ingress: &installv1alpha1.IngressNginxComponent{
					Type: installv1alpha1.LoadBalancer,
					NGINXInstallArgs: []installv1alpha1.InstallArgs{
						{
							Name:  "name1",
							Value: "value1",
						},
					},
					Ports: []corev1.ServicePort{
						{
							Name:       "port1",
							Protocol:   corev1.ProtocolTCP,
							Port:       8000,
							TargetPort: intstr.FromInt(8000),
							NodePort:   30500,
						},
					},
				},
				Istio: &installv1alpha1.IstioComponent{
					IstioInstallArgs: []installv1alpha1.InstallArgs{
						{
							Name:  "name2",
							Value: "value2",
						},
					},
				},
				CertManager: &installv1alpha1.CertManagerComponent{
					Certificate: installv1alpha1.Certificate{
						CA: installv1alpha1.CA{
							SecretName:               "customSecret",
							ClusterResourceNamespace: "customNamespace",
						},
					},
				},
				Rancher: &installv1alpha1.RancherComponent{
					Enabled: &enabled,
				},
				Fluentd: &installv1alpha1.FluentdComponent{
					Enabled: &enabled,
				},
				Keycloak: &installv1alpha1.KeycloakComponent{
					KeycloakInstallArgs: []installv1alpha1.InstallArgs{
						{
							Name:  "keycloak-name",
							Value: "keycloak-value",
						},
					},
					MySQL: installv1alpha1.MySQLComponent{
						MySQLInstallArgs: []installv1alpha1.InstallArgs{
							{
								Name:  "mysql-name",
								Value: "mysql-value",
							},
						},
					},
					Enabled: &enabled,
				},
				Elasticsearch: &installv1alpha1.ElasticsearchComponent{
					MonitoringComponent: installv1alpha1.MonitoringComponent{
						Enabled: &enabled,
					},
				},
				Kibana: &installv1alpha1.KibanaComponent{
					MonitoringComponent: installv1alpha1.MonitoringComponent{
						Enabled: &enabled,
					},
				},
				Prometheus: &installv1alpha1.PrometheusComponent{
					MonitoringComponent: installv1alpha1.MonitoringComponent{
						Enabled: &enabled,
					},
				},
				Grafana: &installv1alpha1.GrafanaComponent{
					MonitoringComponent: installv1alpha1.MonitoringComponent{
						Enabled: &enabled,
					},
				},
				Console: &installv1alpha1.ConsoleComponent{
					MonitoringComponent: installv1alpha1.MonitoringComponent{
						Enabled: &enabled,
					},
				},
			},
		},
	}

	config, err := GetInstallConfig(&vz)
	assert.NoError(t, err)
	assert.Equalf(t, "testEnv", config.EnvironmentName, "Expected environment name did not match")
	assert.Equalf(t, installv1alpha1.Dev, config.Profile, "Expected profile did not match")
	assert.Equalf(t, DNSTypeWildcard, config.DNS.Type, "Expected DNS type did not match")
	assert.Equalf(t, "nip.io", config.DNS.Wildcard.Domain, "Expected domain did not match")

	assert.Equalf(t, IngressTypeLoadBalancer, config.Ingress.Type, "Expected Ingress type did not match")
	assert.Equalf(t, 1, len(config.Ingress.Verrazzano.NginxInstallArgs), "Expected nginxInstallArgs length did not match")
	assert.Equalf(t, "name1", config.Ingress.Verrazzano.NginxInstallArgs[0].Name, "Expected nginxInstallArg name did not match")
	assert.Equalf(t, "value1", config.Ingress.Verrazzano.NginxInstallArgs[0].Value, "Expected nginxInstallArg value did not match")
	assert.Equalf(t, 1, len(config.Ingress.Verrazzano.Ports), "Expected ports length did not match")
	assert.Equalf(t, "port1", config.Ingress.Verrazzano.Ports[0].Name, "Expected port name did not match")
	assert.Equalf(t, "TCP", config.Ingress.Verrazzano.Ports[0].Protocol, "Expected port protocol did not match")
	assert.Equalf(t, int32(8000), config.Ingress.Verrazzano.Ports[0].Port, "Expected port did not match")
	assert.Equalf(t, int32(8000), config.Ingress.Verrazzano.Ports[0].TargetPort, "Expected target port did not match")
	assert.Equalf(t, int32(30500), config.Ingress.Verrazzano.Ports[0].NodePort, "Expected node port did not match")
	assert.Equalf(t, 1, len(config.Ingress.Application.IstioInstallArgs), "Expected istioInstallArgs length did not match")
	assert.Equalf(t, "name2", config.Ingress.Application.IstioInstallArgs[0].Name, "Expected istioInstallArg name did not match")
	assert.Equalf(t, "value2", config.Ingress.Application.IstioInstallArgs[0].Value, "Expected istioInstallArg name did not match")

	assert.Equalf(t, CertIssuerTypeCA, config.Certificates.IssuerType, "Expected certification issuer type did not match")
	assert.Equalf(t, "customNamespace", config.Certificates.CA.ClusterResourceNamespace, "Expected namespace did not match")
	assert.Equalf(t, "customSecret", config.Certificates.CA.SecretName, "Expected CA secret name did not match")

	assert.Equalf(t, "false", config.Rancher.Enabled, "Expected Rancher enabled did not match")
	assert.Equalf(t, "false", config.Fluentd.Enabled, "Expected Fluentd enabled did not match")

	assert.Equalf(t, 1, len(config.Keycloak.KeycloakInstallArgs), "Expected keycloakInstallArgs length did not match")
	assert.Equalf(t, "keycloak-name", config.Keycloak.KeycloakInstallArgs[0].Name, "Expected keycloakInstallArgs name did not match")
	assert.Equalf(t, "keycloak-value", config.Keycloak.KeycloakInstallArgs[0].Value, "Expected keycloakInstallArgs value did not match")
	assert.Equalf(t, 1, len(config.Keycloak.MySQL.MySQLInstallArgs), "Expected mysqlInstallArgs length did not match")
	assert.Equalf(t, "mysql-name", config.Keycloak.MySQL.MySQLInstallArgs[0].Name, "Expected mysqlInstallArgs name did not match")
	assert.Equalf(t, "mysql-value", config.Keycloak.MySQL.MySQLInstallArgs[0].Value, "Expected mysqlInstallArgs value did not match")
	assert.Equalf(t, "false", config.Keycloak.Enabled, "Expected Keycloak enabled did not match")

	assert.Equalf(t, 5, len(config.VzInstallArgs), "Expected VzInstallArgs length did not match")
	assert.Equalf(t, "elasticSearch.enabled", config.VzInstallArgs[0].Name, "Expected name did not match")
	assert.Equalf(t, "false", config.VzInstallArgs[0].Value, "Expected value did not match")
	assert.Equalf(t, "prometheus.enabled", config.VzInstallArgs[1].Name, "Expected name did not match")
	assert.Equalf(t, "false", config.VzInstallArgs[1].Value, "Expected value did not match")
	assert.Equalf(t, "kibana.enabled", config.VzInstallArgs[2].Name, "Expected name did not match")
	assert.Equalf(t, "false", config.VzInstallArgs[1].Value, "Expected value did not match")
	assert.Equalf(t, "grafana.enabled", config.VzInstallArgs[3].Name, "Expected name did not match")
	assert.Equalf(t, "false", config.VzInstallArgs[1].Value, "Expected value did not match")
	assert.Equalf(t, "console.enabled", config.VzInstallArgs[4].Name, "Expected name did not match")
	assert.Equalf(t, "false", config.VzInstallArgs[1].Value, "Expected value did not match")
}

// TestExternalInstall tests the creation of an external install configuration
// GIVEN a verrazzano.install.verrazzano.io custom resource
//  WHEN I call GetInstallConfig
//  THEN the external install configuration is created and verified
func TestExternalInstall(t *testing.T) {
	vz := installv1alpha1.Verrazzano{
		Spec: installv1alpha1.VerrazzanoSpec{
			Profile:         "prod",
			EnvironmentName: "external",
			Components: installv1alpha1.ComponentSpec{
				DNS: &installv1alpha1.DNSComponent{
					External: &installv1alpha1.External{
						Suffix: "abc.def.com",
					},
				},
				Ingress: &installv1alpha1.IngressNginxComponent{
					Type: installv1alpha1.LoadBalancer,
					NGINXInstallArgs: []installv1alpha1.InstallArgs{
						{
							Name:  "name1",
							Value: "value1",
						},
						{
							Name:  "name2",
							Value: "value2",
						},
						{
							Name: "name3",
							ValueList: []string{
								"valueList3-1",
								"valueList3-2",
							},
						},
					},
					Ports: []corev1.ServicePort{
						{
							Name:       "port1",
							Protocol:   corev1.ProtocolTCP,
							Port:       8000,
							TargetPort: intstr.FromInt(8000),
							NodePort:   30500,
						},
						{
							Name:       "port2",
							Protocol:   corev1.ProtocolUDP,
							Port:       8010,
							TargetPort: intstr.FromString("8011"),
						},
						{
							Name:     "port3",
							Protocol: corev1.ProtocolSCTP,
							Port:     8020,
							NodePort: 30600,
						},
					},
				},
				Istio: &installv1alpha1.IstioComponent{
					IstioInstallArgs: []installv1alpha1.InstallArgs{
						{
							Name:  "name4",
							Value: "value4",
						},
						{
							Name: "name5",
							ValueList: []string{
								"valueList5-1",
							},
						},
					},
				},
			},
		},
	}

	config, err := GetInstallConfig(&vz)
	assert.NoError(t, err)
	assert.Equalf(t, "external", config.EnvironmentName, "Expected environment name did not match")
	assert.Equalf(t, installv1alpha1.Prod, config.Profile, "Expected profile did not match")

	assert.Equalf(t, DNSTypeExternal, config.DNS.Type, "Expected DNS type did not match")
	assert.Equalf(t, "abc.def.com", config.DNS.External.Suffix, "Expected DNS external suffix did not match")

	assert.Equalf(t, IngressTypeLoadBalancer, config.Ingress.Type, "Expected Ingress type did not match")
	assert.Equalf(t, 4, len(config.Ingress.Verrazzano.NginxInstallArgs), "Expected nginxInstallArgs length did not match")
	assert.Equalf(t, "name1", config.Ingress.Verrazzano.NginxInstallArgs[0].Name, "Expected nginxInstallArg name did not match")
	assert.Equalf(t, "value1", config.Ingress.Verrazzano.NginxInstallArgs[0].Value, "Expected nginxInstallArg value did not match")
	assert.Equalf(t, "name2", config.Ingress.Verrazzano.NginxInstallArgs[1].Name, "Expected nginxInstallArg name did not match")
	assert.Equalf(t, "value2", config.Ingress.Verrazzano.NginxInstallArgs[1].Value, "Expected nginxInstallArg value did not match")
	assert.Equalf(t, fmt.Sprintf("%s[0]", "name3"), config.Ingress.Verrazzano.NginxInstallArgs[2].Name, "Expected nginxInstallArg name did not match")
	assert.Equalf(t, "valueList3-1", config.Ingress.Verrazzano.NginxInstallArgs[2].Value, "Expected nginxInstallArg value did not match")
	assert.Equalf(t, fmt.Sprintf("%s[1]", "name3"), config.Ingress.Verrazzano.NginxInstallArgs[3].Name, "Expected nginxInstallArg name did not match")
	assert.Equalf(t, "valueList3-2", config.Ingress.Verrazzano.NginxInstallArgs[3].Value, "Expected nginxInstallArg value did not match")
	assert.Equalf(t, 3, len(config.Ingress.Verrazzano.Ports), "Expected ports length did not match")
	assert.Equalf(t, "port1", config.Ingress.Verrazzano.Ports[0].Name, "Expected port name did not match")
	assert.Equalf(t, "TCP", config.Ingress.Verrazzano.Ports[0].Protocol, "Expected port protocol did not match")
	assert.Equalf(t, int32(8000), config.Ingress.Verrazzano.Ports[0].Port, "Expected port did not match")
	assert.Equalf(t, int32(8000), config.Ingress.Verrazzano.Ports[0].TargetPort, "Expected target port did not match")
	assert.Equalf(t, int32(30500), config.Ingress.Verrazzano.Ports[0].NodePort, "Expected node port did not match")
	assert.Equalf(t, "port2", config.Ingress.Verrazzano.Ports[1].Name, "Expected port name did not match")
	assert.Equalf(t, "UDP", config.Ingress.Verrazzano.Ports[1].Protocol, "Expected port protocol did not match")
	assert.Equalf(t, int32(8010), config.Ingress.Verrazzano.Ports[1].Port, "Expected port did not match")
	assert.Equalf(t, int32(8011), config.Ingress.Verrazzano.Ports[1].TargetPort, "Expected target port did not match")
	assert.Equalf(t, "port3", config.Ingress.Verrazzano.Ports[2].Name, "Expected port name did not match")
	assert.Equalf(t, "SCTP", config.Ingress.Verrazzano.Ports[2].Protocol, "Expected port protocol did not match")
	assert.Equalf(t, int32(8020), config.Ingress.Verrazzano.Ports[2].Port, "Expected port did not match")
	assert.Equalf(t, int32(30600), config.Ingress.Verrazzano.Ports[2].NodePort, "Expected node port did not match")
	assert.Equalf(t, 2, len(config.Ingress.Application.IstioInstallArgs), "Expected istioInstallArgs length did not match")
	assert.Equalf(t, "name4", config.Ingress.Application.IstioInstallArgs[0].Name, "Expected istioInstallArg name did not match")
	assert.Equalf(t, "value4", config.Ingress.Application.IstioInstallArgs[0].Value, "Expected istioInstallArg name did not match")
	assert.Equalf(t, fmt.Sprintf("%s[0]", "name5"), config.Ingress.Application.IstioInstallArgs[1].Name, "Expected istioInstallArg name did not match")
	assert.Equalf(t, "valueList5-1", config.Ingress.Application.IstioInstallArgs[1].Value, "Expected istioInstallArg name did not match")

	assert.Equalf(t, CertIssuerTypeCA, config.Certificates.IssuerType, "Expected certification issuer type did not match")
	assert.Equalf(t, "cert-manager", config.Certificates.CA.ClusterResourceNamespace, "Expected namespace did not match")
	assert.Equalf(t, "verrazzano-ca-certificate-secret", config.Certificates.CA.SecretName, "Expected CA secret name did not match")
}

// TestOCIDNSInstall tests the creation of an OCI DNS install configuration
// GIVEN a verrazzano.install.verrazzano.io custom resource
//  WHEN I call GetInstallConfig
//  THEN the OCI DNS install configuration is created and verified
func TestOCIDNSInstall(t *testing.T) {
	vz := installv1alpha1.Verrazzano{
		Spec: installv1alpha1.VerrazzanoSpec{
			Profile:         "prod",
			EnvironmentName: "oci",
			Components: installv1alpha1.ComponentSpec{
				CertManager: &installv1alpha1.CertManagerComponent{
					Certificate: installv1alpha1.Certificate{
						Acme: installv1alpha1.Acme{
							Provider:     installv1alpha1.LetsEncrypt,
							EmailAddress: "someguy@foo.com",
						},
					},
				},
				DNS: &installv1alpha1.DNSComponent{
					OCI: &installv1alpha1.OCI{
						OCIConfigSecret:        "oci-config-secret",
						DNSZoneCompartmentOCID: "test-dns-zone-compartment-ocid",
						DNSZoneOCID:            "test-dns-zone-ocid",
						DNSZoneName:            "test-dns-zone-name",
					},
				},
				Ingress: &installv1alpha1.IngressNginxComponent{
					Type: installv1alpha1.NodePort,
					NGINXInstallArgs: []installv1alpha1.InstallArgs{
						{
							Name:  "name1",
							Value: "value1",
						},
					},
					Ports: []corev1.ServicePort{
						{
							Name:       "port1",
							Protocol:   corev1.ProtocolTCP,
							Port:       8000,
							TargetPort: intstr.FromInt(8000),
							NodePort:   30500,
						},
					},
				},
				Istio: &installv1alpha1.IstioComponent{
					IstioInstallArgs: []installv1alpha1.InstallArgs{
						{
							Name:  "name2",
							Value: "value2",
						},
					},
				},
				Fluentd: &installv1alpha1.FluentdComponent{
					ExtraVolumeMounts: []installv1alpha1.VolumeMount{{Source: "/u01/data"}},
				},
			},
		},
	}

	config, err := GetInstallConfig(&vz)
	assert.NoError(t, err)
	assert.Equalf(t, "oci", config.EnvironmentName, "Expected environment name did not match")
	assert.Equalf(t, installv1alpha1.Prod, config.Profile, "Expected profile did not match")

	assert.Equalf(t, DNSTypeOci, config.DNS.Type, "Expected DNS type did not match")
	assert.Equalf(t, "test-dns-zone-compartment-ocid", config.DNS.Oci.DNSZoneCompartmentOcid, "Expected dns zone compartment ocid did not match")
	assert.Equalf(t, "test-dns-zone-ocid", config.DNS.Oci.DNSZoneOcid, "Expected dns zone ocid did not match")
	assert.Equalf(t, "test-dns-zone-name", config.DNS.Oci.DNSZoneName, "Expected dns zone name did not match")

	assert.Equalf(t, IngressTypeNodePort, config.Ingress.Type, "Expected Ingress type did not match")
	assert.Equalf(t, 1, len(config.Ingress.Verrazzano.NginxInstallArgs), "Expected nginxInstallArgs length did not match")
	assert.Equalf(t, "name1", config.Ingress.Verrazzano.NginxInstallArgs[0].Name, "Expected nginxInstallArg name did not match")
	assert.Equalf(t, "value1", config.Ingress.Verrazzano.NginxInstallArgs[0].Value, "Expected nginxInstallArg value did not match")
	assert.Equalf(t, 1, len(config.Ingress.Verrazzano.Ports), "Expected ports length did not match")
	assert.Equalf(t, "port1", config.Ingress.Verrazzano.Ports[0].Name, "Expected port name did not match")
	assert.Equalf(t, "TCP", config.Ingress.Verrazzano.Ports[0].Protocol, "Expected port protocol did not match")
	assert.Equalf(t, int32(8000), config.Ingress.Verrazzano.Ports[0].Port, "Expected port did not match")
	assert.Equalf(t, int32(8000), config.Ingress.Verrazzano.Ports[0].TargetPort, "Expected target port did not match")
	assert.Equalf(t, int32(30500), config.Ingress.Verrazzano.Ports[0].NodePort, "Expected node port did not match")
	assert.Equalf(t, 1, len(config.Ingress.Application.IstioInstallArgs), "Expected istioInstallArgs length did not match")
	assert.Equalf(t, "name2", config.Ingress.Application.IstioInstallArgs[0].Name, "Expected istioInstallArg name did not match")
	assert.Equalf(t, "value2", config.Ingress.Application.IstioInstallArgs[0].Value, "Expected istioInstallArg name did not match")

	assert.Equalf(t, CertIssuerTypeAcme, config.Certificates.IssuerType, "Expected certification issuer type did not match")
	assert.Equalf(t, "LetsEncrypt", config.Certificates.ACME.Provider, "Expected cert provider did not match")
	assert.Equalf(t, "someguy@foo.com", config.Certificates.ACME.EmailAddress, "Expected email address did not match")
	assert.Equalf(t, "/u01/data", config.Fluentd.FluentdInstallArgs[0].Value, "Expected hostPathMount did not match")
	assert.Equalf(t, "/u01/data", config.Fluentd.FluentdInstallArgs[1].Value, "Expected hostPathMount did not match")
	assert.Equalf(t, "true", config.Fluentd.FluentdInstallArgs[2].Value, "Expected hostPathMount did not match")
}

// TestNodePortInstall tests the creation of a kind install configuration
// GIVEN a verrazzano.install.verrazzano.io custom resource
//  WHEN I call GetInstallConfig
//  THEN the kind install configuration is created and verified
func TestNodePortInstall(t *testing.T) {
	vz := installv1alpha1.Verrazzano{
		Spec: installv1alpha1.VerrazzanoSpec{
			Profile:         "dev",
			EnvironmentName: "kind",
			Components: installv1alpha1.ComponentSpec{
				CertManager: &installv1alpha1.CertManagerComponent{},
				DNS: &installv1alpha1.DNSComponent{
					Wildcard: &installv1alpha1.Wildcard{},
				},
				Ingress: &installv1alpha1.IngressNginxComponent{
					Type: installv1alpha1.NodePort,
					NGINXInstallArgs: []installv1alpha1.InstallArgs{
						{
							Name:      "name1",
							Value:     "value1",
							SetString: false,
						},
						{
							Name:      "name2",
							Value:     "true",
							SetString: true,
						},
						{
							Name: "name3",
							ValueList: []string{
								"valueList3-1",
								"valueList3-2",
							},
						},
						{
							Name:  "name4",
							Value: "value4",
						},
					},
				},
				Istio: &installv1alpha1.IstioComponent{},
			},
		},
	}

	config, err := GetInstallConfig(&vz)
	assert.NoError(t, err)
	assert.Equalf(t, "kind", config.EnvironmentName, "Expected environment name did not match")
	assert.Equalf(t, installv1alpha1.Dev, config.Profile, "Expected profile did not match")

	assert.Equalf(t, DNSTypeWildcard, config.DNS.Type, "Expected DNS type did not match")

	assert.Equalf(t, IngressTypeNodePort, config.Ingress.Type, "Expected Ingress type did not match")
	assert.Equalf(t, 5, len(config.Ingress.Verrazzano.NginxInstallArgs), "Expected nginxInstallArgs length did not match")
	assert.Equalf(t, "name1", config.Ingress.Verrazzano.NginxInstallArgs[0].Name, "Expected nginxInstallArg name did not match")
	assert.Equalf(t, "value1", config.Ingress.Verrazzano.NginxInstallArgs[0].Value, "Expected nginxInstallArg value did not match")
	assert.Equalf(t, false, config.Ingress.Verrazzano.NginxInstallArgs[0].SetString, "Expected nginxInstallArg SetString did not match")
	assert.Equalf(t, "name2", config.Ingress.Verrazzano.NginxInstallArgs[1].Name, "Expected nginxInstallArg name did not match")
	assert.Equalf(t, "true", config.Ingress.Verrazzano.NginxInstallArgs[1].Value, "Expected nginxInstallArg value did not match")
	assert.Equalf(t, true, config.Ingress.Verrazzano.NginxInstallArgs[1].SetString, "Expected nginxInstallArg SetString did not match")
	assert.Equalf(t, fmt.Sprintf("%s[0]", "name3"), config.Ingress.Verrazzano.NginxInstallArgs[2].Name, "Expected nginxInstallArg name did not match")
	assert.Equalf(t, "valueList3-1", config.Ingress.Verrazzano.NginxInstallArgs[2].Value, "Expected nginxInstallArg value did not match")
	assert.Equalf(t, false, config.Ingress.Verrazzano.NginxInstallArgs[2].SetString, "Expected nginxInstallArg SetString did not match")
	assert.Equalf(t, fmt.Sprintf("%s[1]", "name3"), config.Ingress.Verrazzano.NginxInstallArgs[3].Name, "Expected nginxInstallArg name did not match")
	assert.Equalf(t, "valueList3-2", config.Ingress.Verrazzano.NginxInstallArgs[3].Value, "Expected nginxInstallArg value did not match")
	assert.Equalf(t, false, config.Ingress.Verrazzano.NginxInstallArgs[3].SetString, "Expected nginxInstallArg SetString did not match")
	assert.Equalf(t, "name4", config.Ingress.Verrazzano.NginxInstallArgs[4].Name, "Expected nginxInstallArg name did not match")
	assert.Equalf(t, "value4", config.Ingress.Verrazzano.NginxInstallArgs[4].Value, "Expected nginxInstallArg value did not match")
	assert.Equalf(t, false, config.Ingress.Verrazzano.NginxInstallArgs[4].SetString, "Expected nginxInstallArg SetString did not match")

	assert.Equalf(t, CertIssuerTypeCA, config.Certificates.IssuerType, "Expected certification issuer type did not match")
	assert.Equalf(t, "cert-manager", config.Certificates.CA.ClusterResourceNamespace, "Expected namespace did not match")
	assert.Equalf(t, "verrazzano-ca-certificate-secret", config.Certificates.CA.SecretName, "Expected CA secret name did not match")
}

// TestFindVolumeTemplate Test the findVolumeTemplate utility function
// GIVEN a call to findVolumeTemplate
// WHEN valid or invalid arguments are given
// THEN true and the found template are is returned if found, nil/false otherwise
func TestFindVolumeTemplate(t *testing.T) {

	specTemplateList := []installv1alpha1.VolumeClaimSpecTemplate{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "default"},
			Spec: corev1.PersistentVolumeClaimSpec{
				VolumeName: "defVolume",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "template1"},
			Spec: corev1.PersistentVolumeClaimSpec{
				VolumeName: "temp1Volume",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "template2"},
			Spec: corev1.PersistentVolumeClaimSpec{
				VolumeName: "temp2Volume",
			},
		},
	}
	// Test boundary conditions
	invalidName, found := findVolumeTemplate("blah", specTemplateList)
	assert.Nil(t, invalidName)
	assert.False(t, found)
	emptyName, found2 := findVolumeTemplate("", specTemplateList)
	assert.Nil(t, emptyName)
	assert.False(t, found2)
	nilList, found3 := findVolumeTemplate("default", nil)
	assert.Nil(t, nilList)
	assert.False(t, found3)
	emptyList, found4 := findVolumeTemplate("default", []installv1alpha1.VolumeClaimSpecTemplate{})
	assert.Nil(t, emptyList)
	assert.False(t, found4)

	// Test normal behavior
	defTemplate, found := findVolumeTemplate("default", specTemplateList)
	assert.True(t, found)
	assert.Equal(t, "defVolume", defTemplate.VolumeName)
	temp1, found := findVolumeTemplate("template1", specTemplateList)
	assert.True(t, found)
	assert.Equal(t, "temp1Volume", temp1.VolumeName)
	temp2, found := findVolumeTemplate("template2", specTemplateList)
	assert.True(t, found)
	assert.Equal(t, "temp2Volume", temp2.VolumeName)

}

// TestGetVerrazzanoInstallArgsNilDefaultVolumeSource Test the getVerrazzanoInstallArgs  function
// GIVEN a call to getVerrazzanoInstallArgs
// WHEN No default volume source is specified (nil)
// THEN the args list is empty and no error is returned
func TestGetVerrazzanoInstallArgsNilDefaultVolumeSource(t *testing.T) {

	vzSpec := installv1alpha1.VerrazzanoSpec{
		DefaultVolumeSource: nil,
	}
	args, err := getVerrazzanoInstallArgs(&vzSpec)
	assert.Len(t, args, 0)
	assert.Nil(t, err)
}

// TestGetVerrazzanoInstallArgsUnspportedVolumeSource Test the getVerrazzanoInstallArgs  function
// GIVEN a call to getVerrazzanoInstallArgs
// WHEN an unsupported volume source is specified as the defaultVolumeSource
// THEN the args list is empty and an error is returned
func TestGetVerrazzanoInstallArgsUnspportedVolumeSource(t *testing.T) {
	vzSpec := installv1alpha1.VerrazzanoSpec{
		DefaultVolumeSource: &corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{},
		},
	}
	args, err := getVerrazzanoInstallArgs(&vzSpec)
	assert.Len(t, args, 0)
	assert.Nil(t, err)
}

// TestGetVerrazzanoInstallArgsEmptydirDefaultVolumeSource Test the getVerrazzanoInstallArgs  function
// GIVEN a call to getVerrazzanoInstallArgs
// WHEN with an EmptyDirVolumeSource
// THEN the args list specifies helm args with empty strings for the ES/Grafana/Prometheus storage settings
func TestGetVerrazzanoInstallArgsEmptydirDefaultVolumeSource(t *testing.T) {
	vzSpec := installv1alpha1.VerrazzanoSpec{
		DefaultVolumeSource: &corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	args, err := getVerrazzanoInstallArgs(&vzSpec)
	assert.Len(t, args, 3)
	assert.Nil(t, err)
	assert.Equal(t, esStorageValueName, args[0].Name)
	assert.Equal(t, "", args[0].Value)
	assert.True(t, args[0].SetString)
	assert.Equal(t, grafanaStorageValueName, args[1].Name)
	assert.Equal(t, "", args[1].Value)
	assert.True(t, args[1].SetString)
	assert.Equal(t, prometheusStorageValueName, args[2].Name)
	assert.Equal(t, "", args[2].Value)
	assert.True(t, args[2].SetString)
}

// TestGetVerrazzanoInstallArgsUnspportedVolumeSource Test the getVerrazzanoInstallArgs  function
// GIVEN a call to getVerrazzanoInstallArgs with a PersistentVolumeClaimVolumeSource
// WHEN the ClaimName does not match the list of VolumeClaimSpecTemplates
// THEN the args list is empty and an error is returned
func TestGetVerrazzanoInstallArgsInvalidPVCVolumeSource(t *testing.T) {
	vzSpec := installv1alpha1.VerrazzanoSpec{
		DefaultVolumeSource: &corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: "foo",
			},
		},
		VolumeClaimSpecTemplates: []installv1alpha1.VolumeClaimSpecTemplate{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "default"},
				Spec:       corev1.PersistentVolumeClaimSpec{},
			},
		},
	}
	args, err := getVerrazzanoInstallArgs(&vzSpec)
	assert.Len(t, args, 0)
	assert.NotNil(t, err)
}

// TestGetVerrazzanoInstallArgsEmptydirDefaultVolumeSource Test the getVerrazzanoInstallArgs  function
// GIVEN a call to getVerrazzanoInstallArgs
// WHEN with an PersistentVolumeClaimVolumeSource
// THEN the args list specifies helm args the specified storage size for the ES/Grafana/Prometheus storage settings
func TestGetVerrazzanoInstallArgsPVCVolumeSource(t *testing.T) {
	resourceList := make(corev1.ResourceList, 1)
	q, err := resource.ParseQuantity("50Gi")
	assert.NoError(t, err)

	resourceList["storage"] = q
	storageClass := "mystorageclass"
	vzSpec := installv1alpha1.VerrazzanoSpec{
		DefaultVolumeSource: &corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: "default",
			},
		},
		VolumeClaimSpecTemplates: []installv1alpha1.VolumeClaimSpecTemplate{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "default"},
				Spec: corev1.PersistentVolumeClaimSpec{
					StorageClassName: &storageClass,
					Resources: corev1.ResourceRequirements{
						Requests: resourceList,
					},
				},
			},
		},
	}
	args, err := getVerrazzanoInstallArgs(&vzSpec)
	assert.Len(t, args, 3)
	assert.Nil(t, err)
	assert.Equal(t, esStorageValueName, args[0].Name)
	assert.Equal(t, "50Gi", args[0].Value)
	assert.True(t, args[0].SetString)
	assert.Equal(t, grafanaStorageValueName, args[1].Name)
	assert.Equal(t, "50Gi", args[1].Value)
	assert.True(t, args[1].SetString)
	assert.Equal(t, prometheusStorageValueName, args[2].Name)
	assert.Equal(t, "50Gi", args[2].Value)
	assert.True(t, args[2].SetString)
}

// TestGetVerrazzanoInstallArgsWithSecurity Test the getVerrazzanoInstallArgs function
// GIVEN a call to getVerrazzanoInstallArgs with a SecuritySpec
// WHEN the AdminBindng and MonitorBidnging are valid
// THEN the args list is length 4 and err is nil
func TestGetVerrazzanoInstallArgsWithSecurity(t *testing.T) {
	vzSpec := installv1alpha1.VerrazzanoSpec{
		Security: installv1alpha1.SecuritySpec{
			AdminSubjects: []rbacv1.Subject{{
				Kind: "User",
				Name: "kilgore-trout",
			}},
			MonitorSubjects: []rbacv1.Subject{{
				Kind: "Group",
				Name: "group-of-monitors",
			}},
		},
	}
	args, err := getVerrazzanoInstallArgs(&vzSpec)
	assert.Len(t, args, 4)
	assert.Nil(t, err)
	assert.Equal(t, "security.adminSubjects.subject-0.name", args[0].Name)
	assert.Equal(t, "kilgore-trout", args[0].Value)
	assert.True(t, args[0].SetString)
	assert.Equal(t, "security.adminSubjects.subject-0.kind", args[1].Name)
	assert.Equal(t, "User", args[1].Value)
	assert.True(t, args[1].SetString)
	assert.Equal(t, "security.monitorSubjects.subject-0.name", args[2].Name)
	assert.Equal(t, "group-of-monitors", args[2].Value)
	assert.True(t, args[2].SetString)
	assert.Equal(t, "security.monitorSubjects.subject-0.kind", args[3].Name)
	assert.Equal(t, "Group", args[3].Value)
	assert.True(t, args[3].SetString)
}

// TestGetVerrazzanoInstallArgsWithSecurityInvalidSubjectKind Test the getVerrazzanoInstallArgs function
// GIVEN a call to getVerrazzanoInstallArgs with a SecuritySpec
// WHEN the AdminBindng and MonitorBidnging are valid
// THEN the args list is length 4 and err is nil
func TestGetVerrazzanoInstallArgsWithSecurityInvalidSubjectKind(t *testing.T) {
	vzSpec := installv1alpha1.VerrazzanoSpec{
		Security: installv1alpha1.SecuritySpec{
			MonitorSubjects: []rbacv1.Subject{{
				Kind: "BadKind",
				Name: "bad-kind-name",
			}},
		},
	}
	args, err := getVerrazzanoInstallArgs(&vzSpec)
	assert.Len(t, args, 0)
	assert.NotNil(t, err)
}

// TestGetVerrazzanoInstallArgsWithSecurityAndPVCVolumeSource Test the getVerrazzanoInstallArgs function
// GIVEN a call to getVerrazzanoInstallArgs with a SecuritySpec and a PersistentVolumeClaimVolumeSource
// WHEN the SecuritySpec and PVCVolumeSource are both valid
// THEN the args list is specifies both security and helm args the specified storage size for the ES/Grafana/Prometheus storage settings
func TestGetVerrazzanoInstallArgsWithSecurityAndPVCVolumeSource(t *testing.T) {
	resourceList := make(corev1.ResourceList, 1)
	q, err := resource.ParseQuantity("50Gi")
	assert.NoError(t, err)

	resourceList["storage"] = q
	storageClass := "mystorageclass"
	vzSpec := installv1alpha1.VerrazzanoSpec{
		Security: installv1alpha1.SecuritySpec{
			AdminSubjects: []rbacv1.Subject{{
				Kind: "User",
				Name: "kilgore-trout",
			}},
		},
		DefaultVolumeSource: &corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: "default",
			},
		},
		VolumeClaimSpecTemplates: []installv1alpha1.VolumeClaimSpecTemplate{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "default"},
				Spec: corev1.PersistentVolumeClaimSpec{
					StorageClassName: &storageClass,
					Resources: corev1.ResourceRequirements{
						Requests: resourceList,
					},
				},
			},
		},
	}
	args, err := getVerrazzanoInstallArgs(&vzSpec)
	assert.Len(t, args, 5)
	assert.Nil(t, err)
	assert.Equal(t, esStorageValueName, args[0].Name)
	assert.Equal(t, "50Gi", args[0].Value)
	assert.True(t, args[0].SetString)
	assert.Equal(t, grafanaStorageValueName, args[1].Name)
	assert.Equal(t, "50Gi", args[1].Value)
	assert.True(t, args[1].SetString)
	assert.Equal(t, prometheusStorageValueName, args[2].Name)
	assert.Equal(t, "50Gi", args[2].Value)
	assert.True(t, args[2].SetString)
	assert.Equal(t, "security.adminSubjects.subject-0.name", args[3].Name)
	assert.Equal(t, "kilgore-trout", args[3].Value)
	assert.True(t, args[3].SetString)
	assert.Equal(t, "security.adminSubjects.subject-0.kind", args[4].Name)
	assert.Equal(t, "User", args[4].Value)
	assert.True(t, args[4].SetString)
}

// TestGetVMIInstallArgsAllEnabled Test the getVMIInstallArgs function
// GIVEN a call to getVMIInstallArgs
// WHEN all VMI components are enabled in the vzSpec
// THEN the correct set of InstallArg objects are returned and set to "true"
func TestGetVMIInstallArgsAllEnabled(t *testing.T) {

	enabled := true
	vzSpec := installv1alpha1.VerrazzanoSpec{
		Components: installv1alpha1.ComponentSpec{
			Elasticsearch: &installv1alpha1.ElasticsearchComponent{MonitoringComponent: installv1alpha1.MonitoringComponent{Enabled: &enabled}},
			Kibana:        &installv1alpha1.KibanaComponent{MonitoringComponent: installv1alpha1.MonitoringComponent{Enabled: &enabled}},
			Prometheus:    &installv1alpha1.PrometheusComponent{MonitoringComponent: installv1alpha1.MonitoringComponent{Enabled: &enabled}},
			Grafana:       &installv1alpha1.GrafanaComponent{MonitoringComponent: installv1alpha1.MonitoringComponent{Enabled: &enabled}},
		},
	}
	vmiInstallArgs := getVMIInstallArgs(&vzSpec)
	assert.NotNil(t, vmiInstallArgs)
	assert.Equal(t, 4, len(vmiInstallArgs))

	const (
		esIndex      = 0
		promIndex    = 1
		kibanaIndex  = 2
		grafanaIndex = 3
	)

	assert.Equal(t, esEnabledValueName, vmiInstallArgs[esIndex].Name)
	assert.Equal(t, "true", vmiInstallArgs[esIndex].Value)
	assert.False(t, vmiInstallArgs[esIndex].SetString)
	assert.Equal(t, grafanaEnabledValueName, vmiInstallArgs[grafanaIndex].Name)
	assert.Equal(t, "true", vmiInstallArgs[grafanaIndex].Value)
	assert.False(t, vmiInstallArgs[grafanaIndex].SetString)
	assert.Equal(t, promEnabledValueName, vmiInstallArgs[promIndex].Name)
	assert.Equal(t, "true", vmiInstallArgs[promIndex].Value)
	assert.False(t, vmiInstallArgs[promIndex].SetString)
	assert.Equal(t, kibanaEnabledValueName, vmiInstallArgs[kibanaIndex].Name)
	assert.Equal(t, "true", vmiInstallArgs[kibanaIndex].Value)
	assert.False(t, vmiInstallArgs[kibanaIndex].SetString)
}

// TestGetVMIInstallArgsNoneEnabled Test the getVMIInstallArgs function
// GIVEN a call to getVMIInstallArgs
// WHEN all VMI components are disabled in the vzSpec
// THEN the correct set of InstallArg objects are returned and set to "false"
func TestGetVMIInstallArgsNoneEnabled(t *testing.T) {

	enabled := false
	vzSpec := installv1alpha1.VerrazzanoSpec{
		Components: installv1alpha1.ComponentSpec{
			Elasticsearch: &installv1alpha1.ElasticsearchComponent{MonitoringComponent: installv1alpha1.MonitoringComponent{Enabled: &enabled}},
			Kibana:        &installv1alpha1.KibanaComponent{MonitoringComponent: installv1alpha1.MonitoringComponent{Enabled: &enabled}},
			Prometheus:    &installv1alpha1.PrometheusComponent{MonitoringComponent: installv1alpha1.MonitoringComponent{Enabled: &enabled}},
			Grafana:       &installv1alpha1.GrafanaComponent{MonitoringComponent: installv1alpha1.MonitoringComponent{Enabled: &enabled}},
		},
	}
	vmiInstallArgs := getVMIInstallArgs(&vzSpec)
	assert.NotNil(t, vmiInstallArgs)
	assert.Equal(t, 4, len(vmiInstallArgs))

	const (
		esIndex      = 0
		promIndex    = 1
		kibanaIndex  = 2
		grafanaIndex = 3
	)

	assert.Equal(t, esEnabledValueName, vmiInstallArgs[esIndex].Name)
	assert.Equal(t, "false", vmiInstallArgs[esIndex].Value)
	assert.False(t, vmiInstallArgs[esIndex].SetString)
	assert.Equal(t, grafanaEnabledValueName, vmiInstallArgs[grafanaIndex].Name)
	assert.Equal(t, "false", vmiInstallArgs[grafanaIndex].Value)
	assert.False(t, vmiInstallArgs[grafanaIndex].SetString)
	assert.Equal(t, promEnabledValueName, vmiInstallArgs[promIndex].Name)
	assert.Equal(t, "false", vmiInstallArgs[promIndex].Value)
	assert.False(t, vmiInstallArgs[promIndex].SetString)
	assert.Equal(t, kibanaEnabledValueName, vmiInstallArgs[kibanaIndex].Name)
	assert.Equal(t, "false", vmiInstallArgs[kibanaIndex].Value)
	assert.False(t, vmiInstallArgs[kibanaIndex].SetString)
}

// TestGetVMIInstallManagedClusterSettings Test the getVMIInstallArgs function
// GIVEN a call to getVMIInstallArgs
// WHEN all VMI components are disabled in the vzSpec except Prometheus
// THEN the correct set of InstallArg objects are returned and set to "false", minus the Prometheus setting
func TestGetVMIInstallArgsSomeDisabled(t *testing.T) {
	// Simulates the managed-cluster settings
	enabled := false
	vzSpec := installv1alpha1.VerrazzanoSpec{
		Components: installv1alpha1.ComponentSpec{
			Elasticsearch: &installv1alpha1.ElasticsearchComponent{MonitoringComponent: installv1alpha1.MonitoringComponent{Enabled: &enabled}},
			Kibana:        &installv1alpha1.KibanaComponent{MonitoringComponent: installv1alpha1.MonitoringComponent{Enabled: &enabled}},
			Grafana:       &installv1alpha1.GrafanaComponent{MonitoringComponent: installv1alpha1.MonitoringComponent{Enabled: &enabled}},
		},
	}
	vmiInstallArgs := getVMIInstallArgs(&vzSpec)
	assert.NotNil(t, vmiInstallArgs)
	assert.Equal(t, 3, len(vmiInstallArgs))

	const (
		esIndex      = 0
		kibanaIndex  = 1
		grafanaIndex = 2
	)

	assert.Equal(t, esEnabledValueName, vmiInstallArgs[esIndex].Name)
	assert.Equal(t, "false", vmiInstallArgs[esIndex].Value)
	assert.False(t, vmiInstallArgs[esIndex].SetString)
	assert.Equal(t, grafanaEnabledValueName, vmiInstallArgs[grafanaIndex].Name)
	assert.Equal(t, "false", vmiInstallArgs[grafanaIndex].Value)
	assert.False(t, vmiInstallArgs[grafanaIndex].SetString)
	assert.Equal(t, kibanaEnabledValueName, vmiInstallArgs[kibanaIndex].Name)
	assert.Equal(t, "false", vmiInstallArgs[kibanaIndex].Value)
	assert.False(t, vmiInstallArgs[kibanaIndex].SetString)
}

// TestGetVMIInstallArgsNonePresent Test the getVMIInstallArgs function
// GIVEN a call to getVMIInstallArgs
// WHEN no VMI components are present in the vzSpec
// THEN an empty list is returned
func TestGetVMIInstallArgsNonePresent(t *testing.T) {
	vzSpec := installv1alpha1.VerrazzanoSpec{
		Components: installv1alpha1.ComponentSpec{},
	}
	vmiInstallArgs := getVMIInstallArgs(&vzSpec)
	assert.NotNil(t, vmiInstallArgs)
	assert.Equal(t, 0, len(vmiInstallArgs))
}

// TestGetVerrazzanoInstallArgsNoArgs Test the getVerrazzanoInstallArgs function
// GIVEN a call to getVMIInstallArgs
// WHEN no VMI components are present in the vzSpec
// THEN an empty list is returned
func TestGetVerrazzanoInstallArgsNoArgs(t *testing.T) {
	vzSpec := installv1alpha1.VerrazzanoSpec{
		Components: installv1alpha1.ComponentSpec{},
	}
	installArgs, err := getVerrazzanoInstallArgs(&vzSpec)
	assert.NoError(t, err)
	assert.NotNil(t, installArgs)
	assert.Equal(t, 0, len(installArgs))
}

// TestGetVerrazzanoInstallArgsSomeEnabled Test the getVerrazzanoInstallArgs function
// GIVEN a call to getVerrazzanoInstallArgs
// WHEN some components are enabled
// THEN the correct set of InstallArg objects are returned and set to "true", minus the Prometheus setting
func TestGetVerrazzanoInstallArgsSomeEnabled(t *testing.T) {
	// Simulates the managed-cluster settings
	enabled := true
	vzSpec := installv1alpha1.VerrazzanoSpec{
		Components: installv1alpha1.ComponentSpec{
			Elasticsearch: &installv1alpha1.ElasticsearchComponent{MonitoringComponent: installv1alpha1.MonitoringComponent{Enabled: &enabled}},
			Kibana:        &installv1alpha1.KibanaComponent{MonitoringComponent: installv1alpha1.MonitoringComponent{Enabled: &enabled}},
			Grafana:       &installv1alpha1.GrafanaComponent{MonitoringComponent: installv1alpha1.MonitoringComponent{Enabled: &enabled}},
			Console:       &installv1alpha1.ConsoleComponent{MonitoringComponent: installv1alpha1.MonitoringComponent{Enabled: &enabled}},
		},
	}
	installArgs, err := getVerrazzanoInstallArgs(&vzSpec)
	assert.NoError(t, err)
	assert.NotNil(t, installArgs)
	assert.Equal(t, 4, len(installArgs))

	const (
		esIndex      = 0
		kibanaIndex  = 1
		grafanaIndex = 2
		consoleIndex = 3
	)

	assert.Equal(t, esEnabledValueName, installArgs[esIndex].Name)
	assert.Equal(t, "true", installArgs[esIndex].Value)
	assert.False(t, installArgs[esIndex].SetString)
	assert.Equal(t, grafanaEnabledValueName, installArgs[grafanaIndex].Name)
	assert.Equal(t, "true", installArgs[grafanaIndex].Value)
	assert.False(t, installArgs[grafanaIndex].SetString)
	assert.Equal(t, kibanaEnabledValueName, installArgs[kibanaIndex].Name)
	assert.Equal(t, "true", installArgs[kibanaIndex].Value)
	assert.False(t, installArgs[kibanaIndex].SetString)
	assert.Equal(t, consoleEnabledValueName, installArgs[consoleIndex].Name)
	assert.Equal(t, "true", installArgs[consoleIndex].Value)
	assert.False(t, installArgs[consoleIndex].SetString)
}

// TestGetVerrazzanoInstallArgsSomeVMIDisabled Test the getVerrazzanoInstallArgs function
// GIVEN a call to getVerrazzanoInstallArgs
// WHEN all VMI components are disabled in the vzSpec except Prometheus
// THEN the correct set of InstallArg objects are returned and set to "false", minus the Prometheus setting
func TestGetVerrazzanoInstallArgsSomeVMIDisabled(t *testing.T) {
	// Simulates the managed-cluster settings
	enabled := false
	vzSpec := installv1alpha1.VerrazzanoSpec{
		Components: installv1alpha1.ComponentSpec{
			Elasticsearch: &installv1alpha1.ElasticsearchComponent{MonitoringComponent: installv1alpha1.MonitoringComponent{Enabled: &enabled}},
			Kibana:        &installv1alpha1.KibanaComponent{MonitoringComponent: installv1alpha1.MonitoringComponent{Enabled: &enabled}},
			Grafana:       &installv1alpha1.GrafanaComponent{MonitoringComponent: installv1alpha1.MonitoringComponent{Enabled: &enabled}},
			Console:       &installv1alpha1.ConsoleComponent{MonitoringComponent: installv1alpha1.MonitoringComponent{Enabled: &enabled}},
		},
	}
	installArgs, err := getVerrazzanoInstallArgs(&vzSpec)
	assert.NoError(t, err)
	assert.NotNil(t, installArgs)
	assert.Equal(t, 4, len(installArgs))

	const (
		esIndex      = 0
		kibanaIndex  = 1
		grafanaIndex = 2
		consoleIndex = 3
	)

	assert.Equal(t, esEnabledValueName, installArgs[esIndex].Name)
	assert.Equal(t, "false", installArgs[esIndex].Value)
	assert.False(t, installArgs[esIndex].SetString)
	assert.Equal(t, grafanaEnabledValueName, installArgs[grafanaIndex].Name)
	assert.Equal(t, "false", installArgs[grafanaIndex].Value)
	assert.False(t, installArgs[grafanaIndex].SetString)
	assert.Equal(t, kibanaEnabledValueName, installArgs[kibanaIndex].Name)
	assert.Equal(t, "false", installArgs[kibanaIndex].Value)
	assert.False(t, installArgs[kibanaIndex].SetString)
	assert.Equal(t, consoleEnabledValueName, installArgs[consoleIndex].Name)
	assert.Equal(t, "false", installArgs[consoleIndex].Value)
	assert.False(t, installArgs[consoleIndex].SetString)
}

// TestGetKeycloakEmptyDirVolumeSourceNoDefaultVolumeSource Test the getKeycloak  function
// GIVEN a call to getKeycloak
// WHEN with an EmptyDirVolumeSource in the MySQL VolumeSource configuration
// THEN the args list specifies helm args the specified storage size for the ES/Grafana/Prometheus storage settings
func TestGetKeycloakEmptyDirVolumeSourceNoDefaultVolumeSource(t *testing.T) {
	vzSpec := installv1alpha1.VerrazzanoSpec{
		Components: installv1alpha1.ComponentSpec{
			Keycloak: &installv1alpha1.KeycloakComponent{
				MySQL: installv1alpha1.MySQLComponent{
					MySQLInstallArgs: nil,
					VolumeSource: &corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}
	keycloak, err := getKeycloak(vzSpec.Components.Keycloak, []installv1alpha1.VolumeClaimSpecTemplate{}, nil)
	assert.Nil(t, err)
	assert.NotNil(t, keycloak)
	args := keycloak.MySQL.MySQLInstallArgs
	assert.Len(t, args, 1)

	assert.Equal(t, "persistence.enabled", args[0].Name)
	assert.Equal(t, "false", args[0].Value)
	assert.False(t, args[0].SetString)
}

// TestGetKeycloakEmptyDirVolumeSourceNoDefaultVolumeSource Test the getKeycloak  function
// GIVEN a call to getKeycloak
// WHEN with a PVCVolumeSource in the MySQL VolumeSource configuration and an EmptyDirVolumeSource DefaultVolumeSource
// THEN The MySQL configuration overrides the default EmptyDir configuration
func TestGetKeycloakPVCVolumeSourceOverrideDefaultVolumeSource(t *testing.T) {
	resourceList := make(corev1.ResourceList, 1)
	q, err := resource.ParseQuantity("50Gi")
	assert.NoError(t, err)

	resourceList["storage"] = q
	storageClass := "mystorageclass"
	vzSpec := installv1alpha1.VerrazzanoSpec{
		DefaultVolumeSource: &corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
		Components: installv1alpha1.ComponentSpec{
			Keycloak: &installv1alpha1.KeycloakComponent{
				MySQL: installv1alpha1.MySQLComponent{
					MySQLInstallArgs: nil,
					VolumeSource: &corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "default",
						},
					},
				},
			},
		},
		VolumeClaimSpecTemplates: []installv1alpha1.VolumeClaimSpecTemplate{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "default"},
				Spec: corev1.PersistentVolumeClaimSpec{
					StorageClassName: &storageClass,
					Resources: corev1.ResourceRequirements{
						Requests: resourceList,
					},
					AccessModes: []corev1.PersistentVolumeAccessMode{
						"ReadWriteOnce",
						"ReadWriteMany",
					},
				},
			},
		},
	}
	keycloak, err := getKeycloak(vzSpec.Components.Keycloak, vzSpec.VolumeClaimSpecTemplates, vzSpec.DefaultVolumeSource)
	assert.Nil(t, err)
	assert.NotNil(t, keycloak)
	args := keycloak.MySQL.MySQLInstallArgs
	assert.Len(t, args, 4)

	assert.Equal(t, "persistence.storageClass", args[0].Name)
	assert.Equal(t, storageClass, args[0].Value)
	assert.True(t, args[0].SetString)
	assert.Equal(t, "persistence.size", args[1].Name)
	assert.Equal(t, "50Gi", args[1].Value)
	assert.True(t, args[1].SetString)
	assert.Equal(t, "persistence.accessMode", args[2].Name)
	assert.Equal(t, "ReadWriteOnce", args[2].Value)
	assert.True(t, args[2].SetString)
}

// TestGetKeycloakPVCVolumeSourceNoAccessModes Test the getKeycloak  function
// GIVEN a call to getKeycloak
// WHEN with a PVCVolumeSource in the MySQL VolumeSource configuration with no AccessModes specified
// THEN The MySQL args do not contain the "persistence.accessMode" helm arg override
func TestGetKeycloakPVCVolumeSourceNoAccessModes(t *testing.T) {
	resourceList := make(corev1.ResourceList, 1)
	q, err := resource.ParseQuantity("50Gi")
	assert.NoError(t, err)

	resourceList["storage"] = q
	storageClass := "mystorageclass"
	vzSpec := installv1alpha1.VerrazzanoSpec{
		Components: installv1alpha1.ComponentSpec{
			Keycloak: &installv1alpha1.KeycloakComponent{
				MySQL: installv1alpha1.MySQLComponent{
					MySQLInstallArgs: nil,
					VolumeSource: &corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "default",
						},
					},
				},
			},
		},
		VolumeClaimSpecTemplates: []installv1alpha1.VolumeClaimSpecTemplate{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "default"},
				Spec: corev1.PersistentVolumeClaimSpec{
					StorageClassName: &storageClass,
					Resources: corev1.ResourceRequirements{
						Requests: resourceList,
					},
				},
			},
		},
	}
	keycloak, err := getKeycloak(vzSpec.Components.Keycloak, vzSpec.VolumeClaimSpecTemplates, vzSpec.DefaultVolumeSource)
	assert.Nil(t, err)
	assert.NotNil(t, keycloak)
	args := keycloak.MySQL.MySQLInstallArgs
	assert.Len(t, args, 3)

	assert.Equal(t, "persistence.storageClass", args[0].Name)
	assert.Equal(t, storageClass, args[0].Value)
	assert.True(t, args[0].SetString)
	assert.Equal(t, "persistence.size", args[1].Name)
	assert.Equal(t, "50Gi", args[1].Value)
	assert.True(t, args[1].SetString)
}

// TestGetKeycloakPVCVolumeSourceNoTemplates Test the getKeycloak  function
// GIVEN a call to getKeycloak
// WHEN with a PVCVolumeSource in the MySQL VolumeSource configuration with no templates specified
// THEN The MySQL args are empty and an error is returned
func TestGetKeycloakPVCVolumeSourceNoTemplates(t *testing.T) {
	vzSpec := installv1alpha1.VerrazzanoSpec{
		Components: installv1alpha1.ComponentSpec{
			Keycloak: &installv1alpha1.KeycloakComponent{
				MySQL: installv1alpha1.MySQLComponent{
					MySQLInstallArgs: nil,
					VolumeSource: &corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "default",
						},
					},
				},
			},
		},
	}
	keycloak, err := getKeycloak(vzSpec.Components.Keycloak, vzSpec.VolumeClaimSpecTemplates, vzSpec.DefaultVolumeSource)
	assert.NotNil(t, err)
	assert.NotNil(t, keycloak)
	args := keycloak.MySQL.MySQLInstallArgs
	assert.Len(t, args, 0)
}

// TestGetKeycloakPVCVolumeSourceStorageSizeOnly Test the getKeycloak  function
// GIVEN a call to getKeycloak
// WHEN with a PVCVolumeSource in the MySQL VolumeSource configuration with no AccessModes or StorageClass specified
// THEN The MySQL args do not contain the "persistence.accessMode" or "persistence.storageClass" helm arg override
func TestGetKeycloakPVCVolumeSourceStorageSizeOnly(t *testing.T) {
	resourceList := make(corev1.ResourceList, 1)
	q, err := resource.ParseQuantity("50Gi")
	assert.NoError(t, err)

	resourceList["storage"] = q
	vzSpec := installv1alpha1.VerrazzanoSpec{
		Components: installv1alpha1.ComponentSpec{
			Keycloak: &installv1alpha1.KeycloakComponent{
				MySQL: installv1alpha1.MySQLComponent{
					MySQLInstallArgs: nil,
					VolumeSource: &corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "default",
						},
					},
				},
			},
		},
		VolumeClaimSpecTemplates: []installv1alpha1.VolumeClaimSpecTemplate{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "default"},
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.ResourceRequirements{
						Requests: resourceList,
					},
				},
			},
		},
	}
	keycloak, err := getKeycloak(vzSpec.Components.Keycloak, vzSpec.VolumeClaimSpecTemplates, vzSpec.DefaultVolumeSource)
	assert.Nil(t, err)
	assert.NotNil(t, keycloak)
	args := keycloak.MySQL.MySQLInstallArgs
	assert.Len(t, args, 2)

	assert.Equal(t, "persistence.size", args[0].Name)
	assert.Equal(t, "50Gi", args[0].Value)
	assert.True(t, args[0].SetString)
}

// TestGetKeycloakPVCVolumeSourceZeroStorageSize Test the getKeycloak  function
// GIVEN a call to getKeycloak
// WHEN with a PVCVolumeSource in the MySQL VolumeSource configuration with a Zero size string
// THEN The MySQL args do not contain the "persistence.size" helm arg override
func TestGetKeycloakPVCVolumeSourceZeroStorageSize(t *testing.T) {
	resourceList := make(corev1.ResourceList, 1)
	q, err := resource.ParseQuantity("0")
	assert.NoError(t, err)

	resourceList["storage"] = q
	vzSpec := installv1alpha1.VerrazzanoSpec{
		Components: installv1alpha1.ComponentSpec{
			Keycloak: &installv1alpha1.KeycloakComponent{
				MySQL: installv1alpha1.MySQLComponent{
					MySQLInstallArgs: nil,
					VolumeSource: &corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "default",
						},
					},
				},
			},
		},
		VolumeClaimSpecTemplates: []installv1alpha1.VolumeClaimSpecTemplate{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "default"},
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.ResourceRequirements{
						Requests: resourceList,
					},
				},
			},
		},
	}
	keycloak, err := getKeycloak(vzSpec.Components.Keycloak, vzSpec.VolumeClaimSpecTemplates, vzSpec.DefaultVolumeSource)
	assert.Nil(t, err)
	assert.NotNil(t, keycloak)
	args := keycloak.MySQL.MySQLInstallArgs
	assert.Len(t, args, 1)
}

// TestGetKeycloakPVCVolumeSourceEmptyPVCConfiguration Test the getKeycloak  function
// GIVEN a call to getKeycloak
// WHEN with a PVCVolumeSource in the MySQL VolumeSource configuration with an empty struct
// THEN The MySQL args do not contain any helm overrides
func TestGetKeycloakPVCVolumeSourceEmptyPVCConfiguration(t *testing.T) {
	vzSpec := installv1alpha1.VerrazzanoSpec{
		Components: installv1alpha1.ComponentSpec{
			Keycloak: &installv1alpha1.KeycloakComponent{
				MySQL: installv1alpha1.MySQLComponent{
					MySQLInstallArgs: nil,
					VolumeSource: &corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "default",
						},
					},
				},
			},
		},
		VolumeClaimSpecTemplates: []installv1alpha1.VolumeClaimSpecTemplate{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "default"},
				Spec:       corev1.PersistentVolumeClaimSpec{},
			},
		},
	}
	keycloak, err := getKeycloak(vzSpec.Components.Keycloak, vzSpec.VolumeClaimSpecTemplates, vzSpec.DefaultVolumeSource)
	assert.Nil(t, err)
	assert.NotNil(t, keycloak)
	args := keycloak.MySQL.MySQLInstallArgs
	assert.Len(t, args, 1)
}

// TestNewExternalDNSInstallConfigInvalidVZInstallArgs Test the getVerrazzanoInstallArgs  function
// GIVEN a call to newExternalDNSInstallConfig
// WHEN the VerrazzanoSpec contains an invalid storage config
// THEN the returned config is nil and an error is returned
func TestNewExternalDNSInstallConfigInvalidVZInstallArgs(t *testing.T) {
	vzSpec := installv1alpha1.Verrazzano{
		Spec: installv1alpha1.VerrazzanoSpec{
			DefaultVolumeSource: &corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "foo",
				},
			},
			VolumeClaimSpecTemplates: []installv1alpha1.VolumeClaimSpecTemplate{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "default"},
					Spec:       corev1.PersistentVolumeClaimSpec{},
				},
			},
		},
	}
	config, err := GetInstallConfig(&vzSpec)
	assert.Nil(t, config)
	assert.NotNil(t, err)
}

// TestNewExternalDNSInstallConfigInvalidKeyCloakConfig Test the getKeycloak  function
// GIVEN a call to newExternalDNSInstallConfig
// WHEN with a PVCVolumeSource in the MySQL VolumeSource configuration with no templates specified
// THEN the returned config is nil and an error is returned
func TestNewExternalDNSInstallConfigInvalidKeyCloakConfig(t *testing.T) {
	vzSpec := installv1alpha1.Verrazzano{
		Spec: installv1alpha1.VerrazzanoSpec{
			Components: installv1alpha1.ComponentSpec{
				Keycloak: &installv1alpha1.KeycloakComponent{
					MySQL: installv1alpha1.MySQLComponent{
						MySQLInstallArgs: nil,
						VolumeSource: &corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "default",
							},
						},
					},
				},
			},
		},
	}
	config, err := GetInstallConfig(&vzSpec)
	assert.Nil(t, config)
	assert.NotNil(t, err)
}

// TestNewWildcardInstallConfigInvalidVZInstallArgs Test the getVerrazzanoInstallArgs function
// GIVEN a call to newWildcardInstallConfig
// WHEN the VerrazzanoSpec contains an invalid storage config
// THEN the returned config is nil and an error is returned
func TestNewWildcardInstallConfigInvalidVZInstallArgs(t *testing.T) {
	vzSpec := installv1alpha1.Verrazzano{
		Spec: installv1alpha1.VerrazzanoSpec{
			DefaultVolumeSource: &corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "foo",
				},
			},
			VolumeClaimSpecTemplates: []installv1alpha1.VolumeClaimSpecTemplate{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "default"},
					Spec:       corev1.PersistentVolumeClaimSpec{},
				},
			},
		},
	}
	config, err := GetInstallConfig(&vzSpec)
	assert.Nil(t, config)
	assert.NotNil(t, err)
}

// TestNewWildcardInstallConfigInvalidKeyCloakConfig Test the getKeycloak  function
// GIVEN a call to newWildcardInstallConfig
// WHEN with a PVCVolumeSource in the MySQL VolumeSource configuration with no templates specified
// THEN the returned config is nil and an error is returned
func TestNewWildcardInstallConfigInvalidKeyCloakConfig(t *testing.T) {
	vzSpec := installv1alpha1.Verrazzano{
		Spec: installv1alpha1.VerrazzanoSpec{
			Components: installv1alpha1.ComponentSpec{
				Keycloak: &installv1alpha1.KeycloakComponent{
					MySQL: installv1alpha1.MySQLComponent{
						MySQLInstallArgs: nil,
						VolumeSource: &corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "default",
							},
						},
					},
				},
			},
		},
	}
	config, err := GetInstallConfig(&vzSpec)
	assert.Nil(t, config)
	assert.NotNil(t, err)
}

// TestNewOCIDNSInstallConfigInvalidVZInstallArgs Test the getVerrazzanoInstallArgs  function
// GIVEN a call to newOCIDNSInstallConfig
// WHEN the VerrazzanoSpec contains an invalid storage config
// THEN the returned config is nil and an error is returned
func TestNewOCIDNSInstallConfigInvalidVZInstallArgs(t *testing.T) {
	vzSpec := installv1alpha1.Verrazzano{
		Spec: installv1alpha1.VerrazzanoSpec{
			DefaultVolumeSource: &corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "foo",
				},
			},
			VolumeClaimSpecTemplates: []installv1alpha1.VolumeClaimSpecTemplate{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "default"},
					Spec:       corev1.PersistentVolumeClaimSpec{},
				},
			},
		},
	}
	config, err := GetInstallConfig(&vzSpec)
	assert.Nil(t, config)
	assert.NotNil(t, err)
}

// TestNewOCIDNSInstallConfigInvalidKeyCloakConfig Test the getKeycloak  function
// GIVEN a call to newOCIDNSInstallConfig
// WHEN with a PVCVolumeSource in the MySQL VolumeSource configuration with no templates specified
// THEN the returned config is nil and an error is returned
func TestNewOCIDNSInstallConfigInvalidKeyCloakConfig(t *testing.T) {
	vzSpec := installv1alpha1.Verrazzano{
		Spec: installv1alpha1.VerrazzanoSpec{
			Components: installv1alpha1.ComponentSpec{
				Keycloak: &installv1alpha1.KeycloakComponent{
					MySQL: installv1alpha1.MySQLComponent{
						MySQLInstallArgs: nil,
						VolumeSource: &corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "default",
							},
						},
					},
				},
			},
		},
	}
	config, err := GetInstallConfig(&vzSpec)
	assert.Nil(t, config)
	assert.NotNil(t, err)
}

// TestOKEInstallWithFluentdExtraVolumeMounts tests the creation of an Verrazzano with FluentdExtraVolumeMounts
// GIVEN a verrazzano.install.verrazzano.io custom resource
//  WHEN I call GetInstallConfig
//  THEN the Verrazzano install configuration is created and verified
func TestOKEInstallWithFluentdExtraVolumeMounts(t *testing.T) {
	boolTrue := true
	boolFalse := false
	vz := installv1alpha1.Verrazzano{
		Spec: installv1alpha1.VerrazzanoSpec{
			Profile:         "prod",
			EnvironmentName: "oke",
			Components: installv1alpha1.ComponentSpec{
				Fluentd: &installv1alpha1.FluentdComponent{
					ExtraVolumeMounts: []installv1alpha1.VolumeMount{
						{Source: "/u01/data"},
						{Source: "/var/lib", Destination: "/var-lib", ReadOnly: &boolTrue},
						{Source: "/test/data/foo", ReadOnly: &boolFalse},
					},
				},
			},
		},
	}
	config, err := GetInstallConfig(&vz)
	assert.NoError(t, err)
	assert.Equalf(t, "logging.extraVolumeMounts[0].source",
		config.Fluentd.FluentdInstallArgs[0].Name, "Expected name did not match")
	assert.Equalf(t, "/u01/data", config.Fluentd.FluentdInstallArgs[0].Value, "Expected Source did not match")
	assert.Equalf(t, "logging.extraVolumeMounts[0].destination",
		config.Fluentd.FluentdInstallArgs[1].Name, "Expected name did not match")
	assert.Equalf(t, "/u01/data", config.Fluentd.FluentdInstallArgs[1].Value, "Expected Destination did not match")
	assert.Equalf(t, "logging.extraVolumeMounts[0].readOnly",
		config.Fluentd.FluentdInstallArgs[2].Name, "Expected name did not match")
	assert.Equalf(t, "true", config.Fluentd.FluentdInstallArgs[2].Value, "Expected ReadOnly did not match")

	assert.Equalf(t, "logging.extraVolumeMounts[1].source",
		config.Fluentd.FluentdInstallArgs[3].Name, "Expected name did not match")
	assert.Equalf(t, "/var/lib", config.Fluentd.FluentdInstallArgs[3].Value, "Expected Source did not match")
	assert.Equalf(t, "logging.extraVolumeMounts[1].destination",
		config.Fluentd.FluentdInstallArgs[4].Name, "Expected name did not match")
	assert.Equalf(t, "/var-lib", config.Fluentd.FluentdInstallArgs[4].Value, "Expected Destination did not match")
	assert.Equalf(t, "logging.extraVolumeMounts[1].readOnly",
		config.Fluentd.FluentdInstallArgs[5].Name, "Expected name did not match")
	assert.Equalf(t, "true", config.Fluentd.FluentdInstallArgs[5].Value, "Expected ReadOnly did not match")

	assert.Equalf(t, "logging.extraVolumeMounts[2].source",
		config.Fluentd.FluentdInstallArgs[6].Name, "Expected name did not match")
	assert.Equalf(t, "/test/data/foo", config.Fluentd.FluentdInstallArgs[6].Value, "Expected Source did not match")
	assert.Equalf(t, "logging.extraVolumeMounts[2].destination",
		config.Fluentd.FluentdInstallArgs[7].Name, "Expected name did not match")
	assert.Equalf(t, "/test/data/foo", config.Fluentd.FluentdInstallArgs[7].Value, "Expected Destination did not match")
	assert.Equalf(t, "logging.extraVolumeMounts[2].readOnly",
		config.Fluentd.FluentdInstallArgs[8].Name, "Expected name did not match")
	assert.Equalf(t, "false", config.Fluentd.FluentdInstallArgs[8].Value, "Expected ReadOnly did not match")
}

// TestFluentdInstallDefaults tests the creation of fluentd install configuration
// GIVEN a verrazzano.install.verrazzano.io custom resource
//  WHEN I call GetInstallConfig
//  THEN the fluentd install configuration is created and verified
func TestFluentdInstallDefaults(t *testing.T) {
	vz := installv1alpha1.Verrazzano{}
	config, err := GetInstallConfig(&vz)
	assert.NoError(t, err)
	assert.Equalf(t, "true", config.Rancher.Enabled, "Expected Fluentd enabled did not match")
	assert.Equalf(t, defaultElasticsearchSecret, config.Fluentd.ElasticsearchSecret, "Expected ElasticsearchSecret did not match")
	assert.Equalf(t, defaultElasticsearchURL, config.Fluentd.ElasticsearchURL, "Expected ElasticsearchURL did not match")
}

// TestFluentdInstallCustom tests the creation of fluentd install configuration
// GIVEN a verrazzano.install.verrazzano.io custom resource
//  WHEN I call GetInstallConfig
//  THEN the fluentd install configuration is created and verified
func TestFluentdInstallCustom(t *testing.T) {
	vz := installv1alpha1.Verrazzano{
		Spec: installv1alpha1.VerrazzanoSpec{
			Components: installv1alpha1.ComponentSpec{
				Fluentd: &installv1alpha1.FluentdComponent{
					ElasticsearchSecret: "es-secret",
					ElasticsearchURL:    "es-url",
				},
			},
		},
	}
	config, err := GetInstallConfig(&vz)
	assert.NoError(t, err)
	assert.Equalf(t, "true", config.Rancher.Enabled, "Expected Fluentd enabled did not match")
	assert.Equalf(t, "es-secret", config.Fluentd.ElasticsearchSecret, "Expected ElasticsearchSecret did not match")
	assert.Equalf(t, "es-url", config.Fluentd.ElasticsearchURL, "Expected ElasticsearchURL did not match")
}
