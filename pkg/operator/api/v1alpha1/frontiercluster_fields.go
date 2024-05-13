package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (fc *FrontierCluster) FrontierServiceboundServicePort() (string, corev1.ServiceType, corev1.ServicePort) {
	// port
	port := corev1.ServicePort{
		Port: 30011,
		Name: fc.Name + "-servicebound",
	}
	if fc.Spec.Frontier.Servicebound.Port != 0 {
		port.Port = int32(fc.Spec.Frontier.Servicebound.Port)
	}
	// service type
	serviceType := corev1.ServiceTypeClusterIP
	if fc.Spec.Frontier.Servicebound.ServiceType != "" {
		serviceType = fc.Spec.Frontier.Servicebound.ServiceType
	}

	// service name
	serviceName := fc.Spec.Frontier.Servicebound.ServiceName
	if serviceName != "" {
		return serviceName, serviceType, port
	}
	return fc.Name + "-servicebound-svc", serviceType, port
}

func (fc *FrontierCluster) FrontierEdgeboundServicePort() (string, corev1.ServiceType, corev1.ServicePort) {
	// port
	port := corev1.ServicePort{
		Port: 30012,
		Name: fc.Name + "-edgebound",
	}
	if fc.Spec.Frontier.Edgebound.Port != 0 {
		port.Port = int32(fc.Spec.Frontier.Edgebound.Port)
	}
	// service type
	serviceType := corev1.ServiceTypeNodePort
	if fc.Spec.Frontier.Edgebound.ServiceType != "" {
		serviceType = fc.Spec.Frontier.Edgebound.ServiceType
	}
	// service name
	serviceName := fc.Spec.Frontier.Edgebound.ServiceName
	if serviceName != "" {
		return serviceName, serviceType, port
	}
	return fc.Name + "-edgebound-svc", serviceType, port
}

func (fc *FrontierCluster) FrontlasControlPlaneServicePort() (string, corev1.ServiceType, corev1.ServicePort) {
	// port
	port := corev1.ServicePort{
		Port: 30012,
		Name: fc.Name + "-controlplane",
	}
	if fc.Spec.Frontlas.ControlPlane.Port != 0 {
		port.Port = int32(fc.Spec.Frontlas.ControlPlane.Port)
	}
	// service type
	serviceType := corev1.ServiceTypeNodePort
	if fc.Spec.Frontlas.ControlPlane.ServiceType != "" {
		serviceType = fc.Spec.Frontlas.ControlPlane.ServiceType
	}
	// service name
	serviceName := fc.Spec.Frontlas.ControlPlane.ServiceName
	if serviceName != "" {
		return serviceName, serviceType, port
	}
	return fc.Name + "-controlplane-svc", serviceType, port
}

// EBTLSCASecretNamespacedName will get the namespaced name of the Secret containing the CA certificate
// As the Secret will be mounted to our pods, it has to be in the same namespace as the FrontierCluster resource
func (fc *FrontierCluster) EBTLSCASecretNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      fc.Spec.Frontier.Edgebound.TLS.CaCertificateSecret.Name,
		Namespace: fc.Namespace,
	}
}

// EBTLSCertKeySecretNamespacedName will get namespaced name of Secret containing the server certificate and key
func (fc *FrontierCluster) EBTLSCertKeySecretNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      fc.Spec.Frontier.Edgebound.TLS.CertificateKeySecret.Name,
		Namespace: fc.Namespace,
	}
}

// operator ca and secret
func (fc *FrontierCluster) EBTLSOperatorCASecretNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      fc.Name + "edgebound-ca-certificate",
		Namespace: fc.Namespace,
	}
}

func (fc *FrontierCluster) EBTLSOperatorCertKeyNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      fc.Name + "edgebound-certkey-certificate",
		Namespace: fc.Namespace,
	}
}

// deployment name
func (fc *FrontierCluster) FrontierDeploymentName() string {
	return fc.Name + "-frontier"
}

func (fc *FrontierCluster) FrontlasDeploymentName() string {
	return fc.Name + "-frontlas"
}

// replicas
func (fc *FrontierCluster) FrontierReplicas() int {
	return fc.Spec.Frontier.Replicas
}

func (fc *FrontierCluster) FrontlasReplicas() int {
	return fc.Spec.Frontlas.Replicas
}

// for lable
func (fc *FrontierCluster) AppName() string {
	return fc.Name
}
