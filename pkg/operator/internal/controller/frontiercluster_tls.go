package controller

import (
	"context"
	"errors"
	"fmt"

	"github.com/singchia/frontier/operator/api/v1alpha1"
	"github.com/singchia/frontier/operator/pkg/kube/secret"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	tlsEBCAMountPath      = "" // tls edgebound CA mount path
	tlsEBCertKeyMountPath = "" // tls edgebound Cert and Key mount path

	tlsSecretCertName = "tls.crt"
	tlsSecretKeyName  = "tls.key"
	tlsCACertName     = "ca.crt"
)

var (
	ErrCANotFoundInSecret = errors.New("CA certificate resource not found in secret")
)

func (r *FrontierClusterReconciler) ensureTLS(ctx context.Context, fc v1alpha1.FrontierCluster) error {
	log := log.FromContext(ctx)
	// edgebound tls
	if fc.Spec.Frontier.Edgebound.TLS.Enabled {
		log.Info("Edgebound TLS is enable, creating/updating TLS certificate and key")
		if err := r.ensureEBCertKeySecret(ctx, r.client, fc); err != nil {
			return fmt.Errorf("Could not ensure certkey secret: %s", err)
		}

		if fc.Spec.Frontier.Edgebound.TLS.MTLS {
			log.Info("Edgebound TLS is enabled, creating/updating CA secret")
			if err := r.ensureEBCASecret(ctx, r.client, fc); err != nil {
				return fmt.Errorf("Could not ensure CA secret: %s", err)
			}
		}
	}
	return nil
}

// ensure servicebound CA
func (r *FrontierClusterReconciler) ensureEBCASecret(ctx context.Context, getUpdateCreator secret.GetUpdateCreator, fc v1alpha1.FrontierCluster) error {
	ca, err := getEBCAFromSecret(ctx, getUpdateCreator, fc.EBTLSCASecretNamespacedName())
	if err != nil {
		return err
	}

	operatorEBCASecret := secret.Builder().
		SetName(fc.EBTLSOperatorCASecretNamespacedName().Name).
		SetNamespace(fc.EBTLSOperatorCASecretNamespacedName().Namespace).
		SetField("ca.crt", ca).
		SetOwnerReferences(fc.GetOwnerReferences()).
		Build()

	return secret.CreateOrUpdate(ctx, getUpdateCreator, operatorEBCASecret)
}

// ensure servicebound cert and key
func (r *FrontierClusterReconciler) ensureEBCertKeySecret(ctx context.Context, getUpdateCreator secret.GetUpdateCreator, fc v1alpha1.FrontierCluster) error {
	cert, key, err := getEBCertAndKeyFromSecret(ctx, getUpdateCreator, fc.EBTLSCertKeySecretNamespacedName())
	if err != nil {
		return err
	}

	operatorEBCertKeySecret := secret.Builder().
		SetName(fc.EBTLSOperatorCertKeyNamespacedName().Name).
		SetNamespace(fc.EBTLSOperatorCertKeyNamespacedName().Namespace).
		SetField("tls.crt", cert).
		SetField("tls.key", key).
		SetDataType(corev1.SecretTypeTLS).
		SetOwnerReferences(fc.GetOwnerReferences()).
		Build()

	return secret.CreateOrUpdate(ctx, getUpdateCreator, operatorEBCertKeySecret)
}

// helper functions
func getEBCertAndKeyFromSecret(ctx context.Context, getter secret.Getter, secretName types.NamespacedName) (string, string, error) {
	cert, err := secret.ReadKey(ctx, getter, tlsSecretCertName, secretName)
	if err != nil {
		return "", "", err
	}
	key, err := secret.ReadKey(ctx, getter, tlsSecretKeyName, secretName)
	if err != nil {
		return "", "", err
	}
	return cert, key, nil
}

func getEBCAFromSecret(ctx context.Context, getter secret.Getter, secretName types.NamespacedName) (string, error) {
	data, err := secret.ReadStringData(ctx, getter, secretName)
	if err != nil {
		return "", nil
	}
	if ca, ok := data[tlsCACertName]; !ok || ca == "" {
		return "", ErrCANotFoundInSecret
	} else {
		return ca, nil
	}
}
