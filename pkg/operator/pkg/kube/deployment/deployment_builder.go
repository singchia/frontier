package deployment

import (
	"fmt"
	"sort"

	"github.com/hashicorp/go-multierror"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type builder struct {
	name        string
	namespace   string
	replicas    int
	serviceName string

	// these fields need to be initialised
	labels                   map[string]string
	matchLabels              map[string]string
	ownerReference           []metav1.OwnerReference
	podTemplateSpec          corev1.PodTemplateSpec
	volumeMountsPerContainer map[string][]corev1.VolumeMount
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
	b.ownerReference = ownerReference
	return b
}

func (b *builder) SetServiceName(serviceName string) *builder {
	b.serviceName = serviceName
	return b
}

func (b *builder) SetReplicas(replicas int) *builder {
	b.replicas = replicas
	return b
}

func (b *builder) SetMatchLabels(matchLabels map[string]string) *builder {
	b.matchLabels = matchLabels
	return b
}

func (b *builder) SetPodTemplateSpec(podTemplateSpec corev1.PodTemplateSpec) *builder {
	b.podTemplateSpec = podTemplateSpec
	return b
}

func (b *builder) AddVolumeMount(containerName string, mount corev1.VolumeMount) *builder {
	b.volumeMountsPerContainer[containerName] = append(b.volumeMountsPerContainer[containerName], mount)
	return b
}

func (b *builder) AddVolumeMounts(containerName string, mounts []corev1.VolumeMount) *builder {
	for _, m := range mounts {
		b.AddVolumeMount(containerName, m)
	}
	return b
}

func (b *builder) AddVolume(volume corev1.Volume) *builder {
	b.podTemplateSpec.Spec.Volumes = append(b.podTemplateSpec.Spec.Volumes, volume)
	return b
}

func (b *builder) AddVolumes(volumes []corev1.Volume) *builder {
	for _, v := range volumes {
		b.AddVolume(v)
	}
	return b
}

// GetContainerIndexByName returns the index of the container with containerName.
func (b builder) GetContainerIndexByName(containerName string) (int, error) {
	for i, c := range b.podTemplateSpec.Spec.Containers {
		if c.Name == containerName {
			return i, nil
		}
	}
	return -1, fmt.Errorf("no container with name [%s] found", containerName)
}

func (b builder) buildPodTemplateSpec() (corev1.PodTemplateSpec, error) {
	podTemplateSpec := b.podTemplateSpec.DeepCopy()
	var errs error
	for containerName, volumeMounts := range b.volumeMountsPerContainer {
		idx, err := b.GetContainerIndexByName(containerName)
		if err != nil {
			errs = multierror.Append(errs, err)
			// other containers may have valid mounts
			continue
		}
		existingVolumeMounts := map[string]bool{}
		for _, volumeMount := range volumeMounts {
			if prevMount, seen := existingVolumeMounts[volumeMount.MountPath]; seen {
				// Volume with the same path already mounted
				errs = multierror.Append(errs, fmt.Errorf("Volume %v already mounted as %v", volumeMount, prevMount))
				continue
			}
			podTemplateSpec.Spec.Containers[idx].VolumeMounts = append(podTemplateSpec.Spec.Containers[idx].VolumeMounts, volumeMount)
			existingVolumeMounts[volumeMount.MountPath] = true
		}
	}

	// sorts environment variables for all containers
	for _, container := range podTemplateSpec.Spec.Containers {
		envVars := container.Env
		sort.SliceStable(envVars, func(i, j int) bool {
			return envVars[i].Name < envVars[j].Name
		})
	}
	return *podTemplateSpec, errs
}

func copyMap(originalMap map[string]string) map[string]string {
	newMap := map[string]string{}
	for k, v := range originalMap {
		newMap[k] = v
	}
	return newMap
}

func (b *builder) Build() (appsv1.Deployment, error) {
	podTemplateSpec, err := b.buildPodTemplateSpec()
	if err != nil {
		return appsv1.Deployment{}, err
	}

	replicas := int32(b.replicas)

	ownerReference := make([]metav1.OwnerReference, len(b.ownerReference))
	copy(ownerReference, b.ownerReference)

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            b.name,
			Namespace:       b.namespace,
			Labels:          copyMap(b.labels),
			OwnerReferences: ownerReference,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: copyMap(b.matchLabels),
			},
			Template: podTemplateSpec,
		},
	}
	return deployment, err
}

func Builder() *builder {
	return &builder{
		labels:                   map[string]string{},
		matchLabels:              map[string]string{},
		ownerReference:           []metav1.OwnerReference{},
		podTemplateSpec:          corev1.PodTemplateSpec{},
		volumeMountsPerContainer: map[string][]corev1.VolumeMount{},
		replicas:                 1,
	}
}
