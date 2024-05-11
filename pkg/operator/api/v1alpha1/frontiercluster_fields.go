package v1alpha1

import "k8s.io/apimachinery/pkg/types"

// SBTLSCASecretNamespacedName will get the namespaced name of the Secret containing the CA certificate
// As the Secret will be mounted to our pods, it has to be in the same namespace as the FrontierCluster resource
func (fc *FrontierCluster) SBTLSCASecretNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      fc.Spec.Security.ServiceboundTLS.CaCertificateSecret.Name,
		Namespace: fc.Namespace,
	}
}

// SBTLSCertKeySecretNamespacedName will get namespaced name of Secret containing the server certificate and key
func (fc *FrontierCluster) SBTLSCertKeySecretNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      fc.Spec.Frontier.Servicebound.CertificateKeySecret.Name,
		Namespace: fc.Namespace,
	}
}

// EBTLSCASecretNamespacedName will get the namespaced name of the Secret containing the CA certificate
// As the Secret will be mounted to our pods, it has to be in the same namespace as the FrontierCluster resource
func (fc *FrontierCluster) EBTLSCASecretNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      fc.Spec.Security.EdgeboundTLS.CaCertificateSecret.Name,
		Namespace: fc.Namespace,
	}
}

// EBTLSCertKeySecretNamespacedName will get namespaced name of Secret containing the server certificate and key
func (fc *FrontierCluster) EBTLSCertKeySecretNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      fc.Spec.Security.EdgeboundTLS.CertificateKeySecret.Name,
		Namespace: fc.Namespace,
	}
}

// operator ca and secret
func (fc *FrontierCluster) SBTLSOperatorCASecretNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      fc.Name + "servicebound-ca-certificate",
		Namespace: fc.Namespace,
	}
}

func (fc *FrontierCluster) SBTLSOperatorCertKeyNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      fc.Name + "servicebound-certkey-certificate",
		Namespace: fc.Namespace,
	}
}

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
