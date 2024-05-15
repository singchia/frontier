package controller

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/singchia/frontier/operator/api/v1alpha1"
	"github.com/singchia/frontier/operator/pkg/kube/container"
	"github.com/singchia/frontier/operator/pkg/kube/deployment"
	"github.com/singchia/frontier/operator/pkg/kube/podtemplatespec"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// image
	FrontierImageEnv = "FRONTIER_IMAGE"
	FrontlasImageEnv = "FRONTLAS_IMAGE"

	// node for frontlas
	NodeNameEnv = "NODE_NAME"

	// port for frontier and frontlas
	FrontierServiceboundPortEnv = "FRONTIER_SERVICEBOUND_PORT"
	FrontierEdgeboundPortEnv    = "FRONTIER_EDGEBOUND_PORT"
	FrontlasControlPlanePortEnv = "FRONTLAS_CONTROLPLANE_PORT"

	// tls for frontier
	FrontierEdgeboundTLSCAMountPath      = "/app/conf/edgebound/tls/ca"
	FrontierEdgebountTLSCertKeyMountPath = "/app/conf/edgebound/tls/secret"

	// redis for frontlas
	FrontlasRedisAddrsEnv    = "REDIS_ADDRS"
	FrontlasRedisDBEnv       = "REDIS_DB"
	FrontlasRedisUserEnv     = "REDIS_USER"
	FrontlasRedisPasswordEnv = "REDIS_PASSWORD"
	FrontlasRedisTypeEnv     = "REDIS_TYPE"
	FrontlasRedisMasterName  = "MASTER_NAME"

	// inner addr
	FrontlasAddrEnv = "FRONTLAS_ADDR" // service + frontierport
)

func (r *FrontierClusterReconciler) ensureDeployment(ctx context.Context, fc v1alpha1.FrontierCluster) (bool, error) {
	log := log.FromContext(ctx)

	log.Info("Create/Updating Frontlas Deployment")
	if err := r.ensureFrontlasDeployment(ctx, fc); err != nil {
		return false, fmt.Errorf("error creating/updating frontlas Deployment: %s", err)
	}

	log.Info("Creating/Updating Frontier Deployment")
	if err := r.ensureFrontierDeployment(ctx, fc); err != nil {
		return false, fmt.Errorf("error creating/updating frontier Deployment: %s", err)
	}

	currentFrontierDeployment, err := r.client.GetDeployment(ctx, fc.FrontierDeploymentNamespacedName())
	if err != nil {
		return false, fmt.Errorf("error getting Deployment: %s", err)
	}
	frontierIsReady := deployment.IsReady(currentFrontierDeployment, fc.FrontierReplicas())

	currentFrontlasDeployment, err := r.client.GetDeployment(ctx, fc.FrontlasDeploymentNamespacedName())
	if err != nil {
		return false, fmt.Errorf("error getting Deployment: %s", err)
	}
	frontlasIsReady := deployment.IsReady(currentFrontlasDeployment, fc.FrontlasReplicas())

	return frontierIsReady && frontlasIsReady, nil
}

func (r *FrontierClusterReconciler) ensureFrontierDeployment(ctx context.Context, fc v1alpha1.FrontierCluster) error {
	app := fc.Name + "-frontier"
	labels := map[string]string{
		"app": app,
	}

	volumeMounts := []corev1.VolumeMount{}
	volumes := []corev1.Volume{}

	if fc.Spec.Frontier.Edgebound.TLS.Enabled {
		// volumes
		permission := int32(416)
		volumeCertKey := corev1.Volume{
			Name: "tls-secret",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  fc.EBTLSOperatorCASecretNamespacedName().Name,
					DefaultMode: &permission,
				},
			},
		}
		volumeCertKeyMount := corev1.VolumeMount{
			Name:      volumeCertKey.Name,
			MountPath: FrontierEdgebountTLSCertKeyMountPath,
		}
		volumes = append(volumes, volumeCertKey)
		volumeMounts = append(volumeMounts, volumeCertKeyMount)

		if fc.Spec.Frontier.Edgebound.TLS.MTLS {
			volumeCA := corev1.Volume{
				Name: "tls-ca",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  fc.EBTLSOperatorCASecretNamespacedName().Name,
						DefaultMode: &permission,
					},
				},
			}
			volumeCAMount := corev1.VolumeMount{
				Name:      volumeCA.Name,
				MountPath: FrontierEdgeboundTLSCAMountPath,
			}
			volumes = append(volumes, volumeCA)
			volumeMounts = append(volumeMounts, volumeCAMount)
		}

	}

	sbservice, _, sbport := fc.FrontierServiceboundServicePort()
	_, _, ebport := fc.FrontierEdgeboundServicePort()
	frontierservice, _, _, fpport := fc.FrontlasServicePort()

	// container
	container := container.Builder().
		SetName("frontier").
		SetImage("singchia/frontier:1.0.0-dev").
		SetImagePullPolicy(corev1.PullAlways).
		SetEnvs([]corev1.EnvVar{{
			Name:  FrontierServiceboundPortEnv,
			Value: strconv.Itoa(int(sbport.Port)),
		}, {
			Name:  FrontierEdgeboundPortEnv,
			Value: strconv.Itoa(int(ebport.Port)),
		}, {
			Name: NodeNameEnv,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		}, {
			Name:  FrontlasAddrEnv,
			Value: net.JoinHostPort(frontierservice, strconv.Itoa(int(fpport.Port))),
		}}).
		SetCommand(nil).
		SetArgs(nil).
		SetVolumeMounts(volumeMounts).
		Build()

	// pod
	podTemplateSpec := podtemplatespec.Builder().
		SetName("frontier").
		SetNamespace(fc.Namespace).
		AddVolumes(volumes).
		SetLabels(labels).
		SetMatchLabels(labels).
		SetNodeAffinity(&fc.Spec.Frontier.NodeAffinity).
		SetOwnerReference(fc.OwnerReferences).
		AddContainer(container).
		SetPodAntiAffinity(&corev1.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
				{
					TopologyKey: "kubernetes.io/hostname",
					LabelSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "app",
								Operator: metav1.LabelSelectorOpIn,
								Values:   []string{app},
							},
						},
					},
				},
			},
		}).
		Build()

	deploy, err := deployment.Builder().
		SetName(fc.FrontierDeploymentNamespacedName().Name).
		SetNamespace(fc.Namespace).
		SetServiceName(sbservice).
		SetLabels(labels).
		SetMatchLabels(labels).
		SetReplicas(fc.FrontierReplicas()).
		SetPodTemplateSpec(podTemplateSpec).
		SetOwnerReference(fc.GetOwnerReferences()).
		Build()
	if err != nil {
		return fmt.Errorf("error build deployment: %s", err)
	}

	_, err = deployment.CreateOrUpdate(ctx, r.client, deploy)
	return err
}

func (r *FrontierClusterReconciler) ensureFrontlasDeployment(ctx context.Context, fc v1alpha1.FrontierCluster) error {
	app := fc.Name + "-frontlas"
	labels := map[string]string{
		"app": app,
	}

	service, _, cpport, _ := fc.FrontlasServicePort()

	// container
	container := container.Builder().
		SetName("frontlas").
		SetImage("singchia/frontlas:1.0.0-dev").
		SetImagePullPolicy(corev1.PullAlways).
		SetEnvs([]corev1.EnvVar{{
			Name:  FrontlasControlPlanePortEnv,
			Value: strconv.Itoa(int(cpport.Port)),
		}, {
			Name:  FrontlasRedisAddrsEnv,
			Value: strings.Join(fc.Spec.Frontlas.Redis.Addrs, ","),
		}, {
			Name:  FrontlasRedisUserEnv,
			Value: fc.Spec.Frontlas.Redis.User,
		}, {
			Name:  FrontlasRedisPasswordEnv,
			Value: fc.Spec.Frontlas.Redis.Password,
		}, {
			Name:  FrontlasRedisTypeEnv,
			Value: string(fc.Spec.Frontlas.Redis.RedisType),
		}, {
			Name:  FrontlasRedisDBEnv,
			Value: strconv.Itoa(fc.Spec.Frontlas.Redis.DB),
		}, {
			Name:  FrontlasRedisMasterName,
			Value: fc.Spec.Frontlas.Redis.MasterName,
		}}).
		SetReadinessProbe(&corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				/* 1.24+
				GRPC: &corev1.GRPCAction{
					Port:    cpport.TargetPort.IntVal,
					Service: &service,
				},
				*/
				HTTPGet: &corev1.HTTPGetAction{
					Port: cpport.TargetPort,
					Path: "/cluster/v1/health",
				},
			},
			PeriodSeconds: 5,
		}).
		SetCommand(nil).
		SetArgs(nil).
		Build()

	// pod
	podTemplateSpec := podtemplatespec.Builder().
		SetName("frontlas").
		SetNamespace(fc.Namespace).
		SetLabels(labels).
		SetMatchLabels(labels).
		SetNodeAffinity(&fc.Spec.Frontlas.NodeAffinity).
		SetOwnerReference(fc.OwnerReferences).
		AddContainer(container).
		Build()

	deploy, err := deployment.Builder().
		SetName(fc.FrontlasDeploymentNamespacedName().Name).
		SetNamespace(fc.Namespace).
		SetServiceName(service).
		SetLabels(labels).
		SetMatchLabels(labels).
		SetReplicas(fc.FrontierReplicas()).
		SetPodTemplateSpec(podTemplateSpec).
		SetOwnerReference(fc.GetOwnerReferences()).
		Build()
	if err != nil {
		return fmt.Errorf("error build deployment: %s", err)
	}

	_, err = deployment.CreateOrUpdate(ctx, r.client, deploy)
	return err
}
