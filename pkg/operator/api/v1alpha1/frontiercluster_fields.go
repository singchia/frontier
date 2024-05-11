package v1alpha1

import "k8s.io/apimachinery/pkg/types"

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
