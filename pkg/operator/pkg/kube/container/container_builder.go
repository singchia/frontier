package container

import (
	corev1 "k8s.io/api/core/v1"
)

type builder struct {
	name string

	image           string
	imagePullPolicy corev1.PullPolicy
	workDir         string
	command         []string
	args            []string
	envs            []corev1.EnvVar

	volumeMounts []corev1.VolumeMount
	ports        []corev1.ContainerPort
}

func (b *builder) SetName(name string) *builder {
	b.name = name
	return b
}

func (b *builder) SetImage(image string) *builder {
	b.image = image
	return b
}

func (b *builder) SetImagePullPolicy(imagePullPolicy corev1.PullPolicy) *builder {
	b.imagePullPolicy = imagePullPolicy
	return b
}

func (b *builder) SetWorkDir(workDir string) *builder {
	b.workDir = workDir
	return b
}

func (b *builder) SetCommand(command []string) *builder {
	b.command = command
	return b
}

func (b *builder) SetArgs(args []string) *builder {
	b.args = args
	return b
}

func (b *builder) SetEnvs(envs []corev1.EnvVar) *builder {
	b.envs = envs
	return b
}

func (b *builder) SetVolumeMounts(volumeMount []corev1.VolumeMount) *builder {
	b.volumeMounts = volumeMount
	return b
}

func (b *builder) SetPorts(ports []corev1.ContainerPort) *builder {
	b.ports = ports
	return b
}

func (b *builder) Build() corev1.Container {
	return corev1.Container{
		Name:            b.name,
		Image:           b.image,
		ImagePullPolicy: b.imagePullPolicy,
		WorkingDir:      b.workDir,
		Command:         b.command,
		Args:            b.args,
		Env:             b.envs,
		VolumeMounts:    b.volumeMounts,
		Ports:           b.ports,
	}
}

func Builder() *builder {
	return &builder{
		envs:         []corev1.EnvVar{},
		volumeMounts: []corev1.VolumeMount{},
		ports:        []corev1.ContainerPort{},
	}
}
