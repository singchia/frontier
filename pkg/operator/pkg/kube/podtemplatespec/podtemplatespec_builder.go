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
	// affinity
	podAntiAffinity *corev1.PodAntiAffinity
	nodeAffinity    *corev1.NodeAffinity
	podAffinity     *corev1.PodAffinity
}

func (b *builder) SetLabels(labels map[string]string) *builder {
	b.labels = labels
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

func (b *builder) AddContainer(name string, container corev1.Container) *builder {
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
			Affinity: &corev1.Affinity{
				NodeAffinity:    b.nodeAffinity,
				PodAffinity:     b.podAffinity,
				PodAntiAffinity: b.podAntiAffinity,
			},
			Tolerations: b.tolerations,
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
