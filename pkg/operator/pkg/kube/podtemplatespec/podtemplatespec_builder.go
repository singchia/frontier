package podtemplatespec

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type builder struct {
	name      string
	namespace string

	// these fields need to be initialised
	labels          map[string]string
	matchLabels     map[string]string
	ownerReferences []metav1.OwnerReference
	annotations     map[string]string

	nodeSelector map[string]string

	// containers
	containers []corev1.Container
	// volumes
	volumes      []corev1.Volume
	volumeMounts map[string][]corev1.VolumeMount // containerName volumeMount

	terminationGracePeriodSeconds int64
	tolerations                   []corev1.Toleration
	topologySpreadConstraints     []corev1.TopologySpreadConstraint
	// affinity
	affinity        *corev1.Affinity // 全量 Affinity，优先级高于下面三段
	podAntiAffinity *corev1.PodAntiAffinity
	nodeAffinity    *corev1.NodeAffinity
	podAffinity     *corev1.PodAffinity

	priorityClassName  string
	serviceAccountName string
	imagePullSecrets   []corev1.LocalObjectReference
	podSecurityContext *corev1.PodSecurityContext
}

func (b *builder) SetLabels(labels map[string]string) *builder {
	b.labels = labels
	return b
}

func (b *builder) MergeLabels(labels map[string]string) *builder {
	for k, v := range labels {
		b.labels[k] = v
	}
	return b
}

func (b *builder) SetAnnotations(annotations map[string]string) *builder {
	b.annotations = annotations
	return b
}

func (b *builder) MergeAnnotations(annotations map[string]string) *builder {
	for k, v := range annotations {
		b.annotations[k] = v
	}
	return b
}

func (b *builder) SetName(name string) *builder {
	b.name = name
	return b
}

func (b *builder) SetNamespace(namespace string) *builder {
	b.namespace = namespace
	return b
}

func (b *builder) SetOwnerReference(ownerReference []metav1.OwnerReference) *builder {
	b.ownerReferences = ownerReference
	return b
}

func (b *builder) SetMatchLabels(matchLabels map[string]string) *builder {
	b.matchLabels = matchLabels
	return b
}

func (b *builder) AddContainer(container corev1.Container) *builder {
	b.containers = append(b.containers, container)
	return b
}

func (b *builder) AddVolume(volume corev1.Volume) *builder {
	b.volumes = append(b.volumes, volume)
	return b
}

func (b *builder) AddVolumes(volumes []corev1.Volume) *builder {
	for _, v := range volumes {
		b.AddVolume(v)
	}
	return b
}

func (b *builder) SetTerminationGracePeriodSeconds(seconds int64) *builder {
	b.terminationGracePeriodSeconds = seconds
	return b
}

func (b *builder) SetNodeSelector(nodeSelector map[string]string) *builder {
	b.nodeSelector = nodeSelector
	return b
}

func (b *builder) SetAffinity(affinity *corev1.Affinity) *builder {
	b.affinity = affinity
	return b
}

func (b *builder) SetPodAntiAffinity(podAntiAffinity *corev1.PodAntiAffinity) *builder {
	b.podAntiAffinity = podAntiAffinity
	return b
}

func (b *builder) SetPodAffinity(podAffinity *corev1.PodAffinity) *builder {
	b.podAffinity = podAffinity
	return b
}

func (b *builder) SetNodeAffinity(nodeAffinity *corev1.NodeAffinity) *builder {
	b.nodeAffinity = nodeAffinity
	return b
}

func (b *builder) SetTolerations(tolerations []corev1.Toleration) *builder {
	b.tolerations = tolerations
	return b
}

func (b *builder) SetTopologySpreadConstraints(c []corev1.TopologySpreadConstraint) *builder {
	b.topologySpreadConstraints = c
	return b
}

func (b *builder) SetPriorityClassName(name string) *builder {
	b.priorityClassName = name
	return b
}

func (b *builder) SetServiceAccountName(name string) *builder {
	b.serviceAccountName = name
	return b
}

func (b *builder) SetImagePullSecrets(s []corev1.LocalObjectReference) *builder {
	b.imagePullSecrets = s
	return b
}

func (b *builder) SetPodSecurityContext(s *corev1.PodSecurityContext) *builder {
	b.podSecurityContext = s
	return b
}

// resolveAffinity 决定最终的 Affinity：用户传整体 affinity 走它（fully wins）；
// 否则用 builder 累计的三段子结构组装。
func (b *builder) resolveAffinity() *corev1.Affinity {
	if b.affinity != nil {
		return b.affinity
	}
	if b.nodeAffinity == nil && b.podAffinity == nil && b.podAntiAffinity == nil {
		return nil
	}
	return &corev1.Affinity{
		NodeAffinity:    b.nodeAffinity,
		PodAffinity:     b.podAffinity,
		PodAntiAffinity: b.podAntiAffinity,
	}
}

func (b *builder) Build() corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name:            b.name,
			Namespace:       b.namespace,
			Labels:          b.labels,
			OwnerReferences: b.ownerReferences,
			Annotations:     b.annotations,
		},
		Spec: corev1.PodSpec{
			Volumes:                       b.volumes,
			Containers:                    b.containers,
			TerminationGracePeriodSeconds: &b.terminationGracePeriodSeconds,
			NodeSelector:                  b.nodeSelector,
			Affinity:                      b.resolveAffinity(),
			Tolerations:                   b.tolerations,
			TopologySpreadConstraints:     b.topologySpreadConstraints,
			PriorityClassName:             b.priorityClassName,
			ServiceAccountName:            b.serviceAccountName,
			ImagePullSecrets:              b.imagePullSecrets,
			SecurityContext:               b.podSecurityContext,
		},
	}
}

func Builder() *builder {
	return &builder{
		labels:          map[string]string{},
		matchLabels:     map[string]string{},
		annotations:     map[string]string{},
		ownerReferences: []metav1.OwnerReference{},
		containers:      []corev1.Container{},
		nodeSelector:    map[string]string{},
		volumes:         []corev1.Volume{},
		volumeMounts:    map[string][]corev1.VolumeMount{},
		tolerations:     []corev1.Toleration{},
	}
}
