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
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// image
	FrontierImageEnv = "FRONTIER_IMAGE"
	FrontlasImageEnv = "FRONTLAS_IMAGE"

	// node for frontlas
	NodeNameEnv = "NODE_NAME"

	// port for frontier and frontlas
	FrontierServiceboundPortEnv  = "FRONTIER_SERVICEBOUND_PORT"
	FrontierEdgeboundPortEnv     = "FRONTIER_EDGEBOUND_PORT"
	FrontlasControlPlanePortEnv  = "FRONTLAS_CONTROLPLANE_PORT"
	FrontlasFrontierPlanePortEnv = "FRONTLAS_FRONTIERPLANE_PORT"

	// graceful shutdown
	FrontierDrainSecondsEnv = "FRONTIER_DRAIN_SECONDS"

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

	currentFrontlasDeployment, err := r.client.GetDeployment(ctx, fc.FrontlasDeploymentNamespacedName())
	if err != nil {
		return false, fmt.Errorf("error getting Deployment: %s", err)
	}
	frontlasIsReady := deployment.IsReady(currentFrontlasDeployment, fc.FrontlasReplicas())
	if !frontlasIsReady {
		log.Info("frontlas deployment is not ready",
			"expectedReplicas", fc.FrontierReplicas(),
			"updatedReplicas", currentFrontlasDeployment.Status.UpdatedReplicas,
			"readyReplicas", currentFrontlasDeployment.Status.ReadyReplicas,
			"generation", currentFrontlasDeployment.Generation,
			"observedGeneration", currentFrontlasDeployment.Status.ObservedGeneration)
		return false, nil
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
	if !frontierIsReady {
		log.Info("frontier deployment is not ready",
			"expectedReplicas", fc.FrontierReplicas(),
			"updatedReplicas", currentFrontierDeployment.Status.UpdatedReplicas,
			"readyReplicas", currentFrontierDeployment.Status.ReadyReplicas,
			"generation", currentFrontierDeployment.Generation,
			"observedGeneration", currentFrontierDeployment.Status.ObservedGeneration)
		return false, nil
	}

	return frontierIsReady && frontlasIsReady, nil
}

func (r *FrontierClusterReconciler) ensureFrontierDeployment(ctx context.Context, fc v1alpha1.FrontierCluster) error {
	app := fc.Name + "-frontier"
	labels := map[string]string{
		"app": app,
	}
	image := fc.Spec.Frontier.Image
	if image == "" {
		image = "singchia/frontier:1.1.0"
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
					SecretName:  fc.EBTLSOperatorCertKeyNamespacedName().Name,
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

	defaults := frontierDefaults(sbport.Port, ebport.Port)
	pod := fc.Spec.Frontier.Pod
	gracePeriod := pickGrace(pod.TerminationGracePeriodSeconds, defaults.terminationGracePeriodSeconds)
	// drain 默认 = grace - 10s，给 Close() 自身留余量；下界 0，不会变负。
	drainSeconds := gracePeriod - 10
	if drainSeconds < 0 {
		drainSeconds = 0
	}

	// container
	container := container.Builder().
		SetName("frontier").
		SetImage(image).
		SetImagePullPolicy(pickPullPolicy(pod.ImagePullPolicy, defaults.imagePullPolicy)).
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
		}, {
			Name:  FrontierDrainSecondsEnv,
			Value: strconv.FormatInt(drainSeconds, 10),
		}}).
		SetCommand(nil).
		SetArgs(nil).
		SetVolumeMounts(volumeMounts).
		SetResources(pickResources(pod.Resources, defaults.resources)).
		SetLivenessProbe(pickProbe(pod.LivenessProbe, defaults.livenessProbe)).
		SetReadinessProbe(pickProbe(pod.ReadinessProbe, defaults.readinessProbe)).
		SetLifecycle(pickLifecycle(pod.Lifecycle, defaults.lifecycle)).
		SetSecurityContext(pickContainerSecurityContext(pod.ContainerSecurityContext, defaults.containerSecurityContext)).
		Build()

	specOver := extractPodSpecOverrides(pod)
	mergedLabels := map[string]string{}
	for k, v := range labels {
		mergedLabels[k] = v
	}

	// pod
	podBuilder := podtemplatespec.Builder().
		SetName("frontier").
		SetNamespace(fc.Namespace).
		AddVolumes(volumes).
		SetLabels(mergedLabels).
		MergeLabels(specOver.Labels).
		SetMatchLabels(labels).
		SetAnnotations(specOver.Annotations).
		SetOwnerReference(fc.OwnerReferences).
		AddContainer(container).
		SetTerminationGracePeriodSeconds(gracePeriod).
		SetTolerations(specOver.Tolerations).
		SetTopologySpreadConstraints(specOver.TopologySpreadConstraints).
		SetNodeSelector(specOver.NodeSelector).
		SetPriorityClassName(specOver.PriorityClassName).
		SetServiceAccountName(specOver.ServiceAccountName).
		SetImagePullSecrets(specOver.ImagePullSecrets).
		SetPodSecurityContext(pickPodSecurityContext(pod.PodSecurityContext, defaults.podSecurityContext)).
		SetAffinity(pickAffinity(pod.Affinity, preferredAntiAffinityByHost(app), &fc.Spec.Frontier.NodeAffinity))

	podTemplateSpec := podBuilder.Build()

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
	image := fc.Spec.Frontlas.Image
	if image == "" {
		image = "singchia/frontlas:1.1.0"
	}

	service, _, cpport, fpport := fc.FrontlasServicePort()

	defaults := frontlasDefaults(cpport.Port)
	pod := fc.Spec.Frontlas.Pod

	// container
	container := container.Builder().
		SetName("frontlas").
		SetImage(image).
		SetImagePullPolicy(pickPullPolicy(pod.ImagePullPolicy, defaults.imagePullPolicy)).
		SetEnvs(frontlasRedisEnvs(fc, cpport.Port, fpport.Port)).
		SetCommand(nil).
		SetArgs(nil).
		SetResources(pickResources(pod.Resources, defaults.resources)).
		SetLivenessProbe(pickProbe(pod.LivenessProbe, defaults.livenessProbe)).
		SetReadinessProbe(pickProbe(pod.ReadinessProbe, defaults.readinessProbe)).
		SetLifecycle(pickLifecycle(pod.Lifecycle, defaults.lifecycle)).
		SetSecurityContext(pickContainerSecurityContext(pod.ContainerSecurityContext, defaults.containerSecurityContext)).
		Build()

	specOver := extractPodSpecOverrides(pod)
	mergedLabels := map[string]string{}
	for k, v := range labels {
		mergedLabels[k] = v
	}

	// pod
	podTemplateSpec := podtemplatespec.Builder().
		SetName("frontlas").
		SetNamespace(fc.Namespace).
		SetLabels(mergedLabels).
		MergeLabels(specOver.Labels).
		SetMatchLabels(labels).
		SetAnnotations(specOver.Annotations).
		SetOwnerReference(fc.OwnerReferences).
		AddContainer(container).
		SetTerminationGracePeriodSeconds(pickGrace(pod.TerminationGracePeriodSeconds, defaults.terminationGracePeriodSeconds)).
		SetTolerations(specOver.Tolerations).
		SetTopologySpreadConstraints(specOver.TopologySpreadConstraints).
		SetNodeSelector(specOver.NodeSelector).
		SetPriorityClassName(specOver.PriorityClassName).
		SetServiceAccountName(specOver.ServiceAccountName).
		SetImagePullSecrets(specOver.ImagePullSecrets).
		SetPodSecurityContext(pickPodSecurityContext(pod.PodSecurityContext, defaults.podSecurityContext)).
		SetAffinity(pickAffinity(pod.Affinity, preferredAntiAffinityByHost(app), &fc.Spec.Frontlas.NodeAffinity)).
		Build()

	deploy, err := deployment.Builder().
		SetName(fc.FrontlasDeploymentNamespacedName().Name).
		SetNamespace(fc.Namespace).
		SetServiceName(service).
		SetLabels(labels).
		SetMatchLabels(labels).
		SetReplicas(fc.FrontlasReplicas()).
		SetPodTemplateSpec(podTemplateSpec).
		SetOwnerReference(fc.GetOwnerReferences()).
		Build()
	if err != nil {
		return fmt.Errorf("error build deployment: %s", err)
	}

	_, err = deployment.CreateOrUpdate(ctx, r.client, deploy)
	return err
}

// frontlasRedisEnvs 把端口 + Redis 配置组装成 EnvVar，
// Password 与 PasswordSecret 互斥（PasswordSecret 优先，进 valueFrom；否则走明文兼容路径）。
func frontlasRedisEnvs(fc v1alpha1.FrontierCluster, cpPort, fpPort int32) []corev1.EnvVar {
	r := fc.Spec.Frontlas.Redis
	envs := []corev1.EnvVar{
		{Name: FrontlasControlPlanePortEnv, Value: strconv.Itoa(int(cpPort))},
		{Name: FrontlasFrontierPlanePortEnv, Value: strconv.Itoa(int(fpPort))},
		{Name: FrontlasRedisAddrsEnv, Value: strings.Join(r.Addrs, ",")},
		{Name: FrontlasRedisUserEnv, Value: r.User},
		{Name: FrontlasRedisTypeEnv, Value: string(r.RedisType)},
		{Name: FrontlasRedisDBEnv, Value: strconv.Itoa(r.DB)},
		{Name: FrontlasRedisMasterName, Value: r.MasterName},
	}
	if r.PasswordSecret != nil {
		envs = append(envs, corev1.EnvVar{
			Name:      FrontlasRedisPasswordEnv,
			ValueFrom: &corev1.EnvVarSource{SecretKeyRef: r.PasswordSecret},
		})
	} else {
		envs = append(envs, corev1.EnvVar{
			Name:  FrontlasRedisPasswordEnv,
			Value: r.Password,
		})
	}
	return envs
}
