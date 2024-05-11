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
	tlsSBCAMountPath      = "" // tls servicebound CA mount path
	tlsSBCertKeyMountPath = "" // tls servicebound Cert and Key mount path
	tlsEBCAMountPath      = "" // tls edgebound CA mount path
	tlsEBCertKeyMountPath = "" // tls edgebound Cert and Key mount path

	tlsSecretCertName = "tls.crt"
	tlsSecretKeyName  = "tls.key"
	tlsCACertName     = "ca.crt"

	tlsEBSecretCertName = "tls.crt"
	tlsEBSecretKeyName  = "tls.key"
	tlsEBCACertName     = "ca.crt"
)

var (
	ErrCANotFoundInSecret = errors.New("CA certificate resource not found in secret")
)

func (r *FrontierClusterReconciler) ensureTLS(ctx context.Context, fc v1alpha1.FrontierCluster) error {
	log := log.FromContext(ctx)
	// servicebound tls
	if fc.Spec.Security.ServiceboundTLS.Enabled {
		log.Info("Servicebound TLS is enable, creating/updating TLS certificate and key")
		if err := r.ensureSBCertKeySecret(ctx, r.client, fc); err != nil {
			return fmt.Errorf("count not ensure certkey secret: %s", err)
		}

		if fc.Spec.Security.ServiceboundTLS.MTLS {
			log.Info("Servicebound TLS is enabled, creating/updating CA secret")
			if err := r.ensureSBCASecret(ctx, r.client, fc); err != nil {
				return fmt.Errorf("cound not ensure CA secret: %s", err)
			}
		}
	}
	// edgebound tls
	if fc.Spec.Security.EdgeboundTLS.Enabled {
		log.Info("Edgebound TLS is enable, creating/updating TLS certificate and key")
		if err := r.ensureEBCertKeySecret(ctx, r.client, fc); err != nil {
			return fmt.Errorf("count not ensure certkey secret: %s", err)
		}

		if fc.Spec.Security.EdgeboundTLS.MTLS {
			log.Info("Edgebound TLS is enabled, creating/updating CA secret")
			if err := r.ensureEBCASecret(ctx, r.client, fc); err != nil {
				return fmt.Errorf("cound not ensure CA secret: %s", err)
			}
		}
	}
	return nil
}

// ensure servicebound CA
func (r *FrontierClusterReconciler) ensureSBCASecret(ctx context.Context, getUpdateCreator secret.GetUpdateCreator, fc v1alpha1.FrontierCluster) error {
	ca, err := getSBCAFromSecret(ctx, getUpdateCreator, fc.SBTLSCASecretNamespacedName())
	if err != nil {
		return err
	}

	operatorSBCASecret := secret.Builder().
		SetName(fc.SBTLSCASecretNamespacedName().Name).
		SetNamespace(fc.SBTLSCASecretNamespacedName().Namespace).
		SetField("ca.crt", ca).
		SetOwnerReferences(fc.GetOwnerReferences()).
		Build()

	return secret.CreateOrUpdate(ctx, getUpdateCreator, operatorSBCASecret)
}

// ensure servicebound cert and key
func (r *FrontierClusterReconciler) ensureSBCertKeySecret(ctx context.Context, getUpdateCreator secret.GetUpdateCreator, fc v1alpha1.FrontierCluster) error {
	cert, key, err := getSBCertAndKeyFromSecret(ctx, getUpdateCreator, fc.SBTLSCertKeySecretNamespacedName())
	if err != nil {
		return err
	}

	operatorSBCertKeySecret := secret.Builder().
		SetName(fc.SBTLSCASecretNamespacedName().Name).
		SetNamespace(fc.SBTLSCASecretNamespacedName().Namespace).
		SetField("tls.crt", cert).
		SetField("tls.key", key).
		SetDataType(corev1.SecretTypeTLS).
		SetOwnerReferences(fc.GetOwnerReferences()).
		Build()

	return secret.CreateOrUpdate(ctx, getUpdateCreator, operatorSBCertKeySecret)
}

// ensure servicebound CA
func (r *FrontierClusterReconciler) ensureEBCASecret(ctx context.Context, getUpdateCreator secret.GetUpdateCreator, fc v1alpha1.FrontierCluster) error {
	ca, err := getEBCAFromSecret(ctx, getUpdateCreator, fc.EBTLSCASecretNamespacedName())
	if err != nil {
		return err
	}

	operatorEBCASecret := secret.Builder().
		SetName(fc.EBTLSCASecretNamespacedName().Name).
		SetNamespace(fc.EBTLSCASecretNamespacedName().Namespace).
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
		SetName(fc.EBTLSCASecretNamespacedName().Name).
		SetNamespace(fc.EBTLSCASecretNamespacedName().Namespace).
		SetField("tls.crt", cert).
		SetField("tls.key", key).
		SetDataType(corev1.SecretTypeTLS).
		SetOwnerReferences(fc.GetOwnerReferences()).
		Build()

	return secret.CreateOrUpdate(ctx, getUpdateCreator, operatorEBCertKeySecret)
}

// helper functions
func getSBCertAndKeyFromSecret(ctx context.Context, getter secret.Getter, secretName types.NamespacedName) (string, string, error) {
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

func getSBCAFromSecret(ctx context.Context, getter secret.Getter, secretName types.NamespacedName) (string, error) {
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
