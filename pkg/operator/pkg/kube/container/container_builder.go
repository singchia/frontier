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
	resources       *corev1.ResourceRequirements
	livenessProbe   *corev1.Probe
	readinessProbe  *corev1.Probe
	lifecycle       *corev1.Lifecycle
	securityContext *corev1.SecurityContext

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

func (b *builder) AddEnvs(envs []corev1.EnvVar) *builder {
	b.envs = append(b.envs, envs...)
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

func (b *builder) SetResources(r *corev1.ResourceRequirements) *builder {
	b.resources = r
	return b
}

func (b *builder) SetLivenessProbe(p *corev1.Probe) *builder {
	b.livenessProbe = p
	return b
}

func (b *builder) SetReadinessProbe(p *corev1.Probe) *builder {
	b.readinessProbe = p
	return b
}

func (b *builder) SetLifecycle(l *corev1.Lifecycle) *builder {
	b.lifecycle = l
	return b
}

func (b *builder) SetSecurityContext(s *corev1.SecurityContext) *builder {
	b.securityContext = s
	return b
}

func (b *builder) Build() corev1.Container {
	c := corev1.Container{
		Name:            b.name,
		Image:           b.image,
		ImagePullPolicy: b.imagePullPolicy,
		WorkingDir:      b.workDir,
		Command:         b.command,
		Args:            b.args,
		Env:             b.envs,
		VolumeMounts:    b.volumeMounts,
		Ports:           b.ports,
		LivenessProbe:   b.livenessProbe,
		ReadinessProbe:  b.readinessProbe,
		Lifecycle:       b.lifecycle,
		SecurityContext: b.securityContext,
	}
	if b.resources != nil {
		c.Resources = *b.resources
	}
	return c
}

func Builder() *builder {
	return &builder{
		envs:         []corev1.EnvVar{},
		volumeMounts: []corev1.VolumeMount{},
		ports:        []corev1.ContainerPort{},
	}
}
