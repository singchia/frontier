package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
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
	port.TargetPort = intstr.FromInt32(port.Port)
	// service type
	serviceType := corev1.ServiceTypeClusterIP
	if fc.Spec.Frontier.Servicebound.ServiceType != "" {
		serviceType = fc.Spec.Frontier.Servicebound.ServiceType
		if serviceType == corev1.ServiceTypeNodePort {
			port.NodePort = port.Port
		}
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
	port.TargetPort = intstr.FromInt32(port.Port)
	// service type
	serviceType := corev1.ServiceTypeNodePort
	if fc.Spec.Frontier.Edgebound.ServiceType != "" {
		serviceType = fc.Spec.Frontier.Edgebound.ServiceType
	}
	if serviceType == corev1.ServiceTypeNodePort {
		port.NodePort = port.Port
	}
	// service name
	serviceName := fc.Spec.Frontier.Edgebound.ServiceName
	if serviceName != "" {
		return serviceName, serviceType, port
	}
	return fc.Name + "-edgebound-svc", serviceType, port
}

func (fc *FrontierCluster) FrontlasServicePort() (string, corev1.ServiceType, corev1.ServicePort, corev1.ServicePort) {
	// port
	cpport := corev1.ServicePort{
		Port: 40011,
		Name: fc.Name + "-controlplane",
	}
	if fc.Spec.Frontlas.ControlPlane.Port != 0 {
		cpport.Port = int32(fc.Spec.Frontlas.ControlPlane.Port)
	}
	cpport.TargetPort = intstr.FromInt32(cpport.Port)

	fpport := corev1.ServicePort{
		Port:       40012,
		TargetPort: intstr.FromInt32(40012),
		Name:       fc.Name + "-frontierplane",
	}
	// service type
	serviceType := corev1.ServiceTypeClusterIP
	if fc.Spec.Frontlas.ControlPlane.ServiceType != "" {
		serviceType = fc.Spec.Frontlas.ControlPlane.ServiceType
		if serviceType == corev1.ServiceTypeNodePort {
			cpport.NodePort = cpport.Port
			fpport.NodePort = fpport.Port
		}
	}
	// service name
	serviceName := fc.Spec.Frontlas.ControlPlane.ServiceName
	if serviceName != "" {
		return serviceName, serviceType, cpport, fpport
	}
	return fc.Name + "-frontlas-svc", serviceType, cpport, fpport
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
func (fc *FrontierCluster) FrontierDeploymentNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      fc.Name + "-frontier",
		Namespace: fc.Namespace,
	}
}

func (fc *FrontierCluster) FrontlasDeploymentNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      fc.Name + "-frontlas",
		Namespace: fc.Namespace,
	}
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
