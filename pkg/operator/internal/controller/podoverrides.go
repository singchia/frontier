package controller

import (
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/singchia/frontier/operator/api/v1alpha1"
)

func intToStr(i int) string {
	return strconv.Itoa(i)
}

func labelSelectorForApp(app string) *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      "app",
				Operator: metav1.LabelSelectorOpIn,
				Values:   []string{app},
			},
		},
	}
}

// componentDefaults 是 operator 给 Frontier / Frontlas 各自的硬编码默认值。
// 用户在 spec 里没填的项走这里。
type componentDefaults struct {
	livenessProbe                 *corev1.Probe
	readinessProbe                *corev1.Probe
	lifecycle                     *corev1.Lifecycle
	terminationGracePeriodSeconds int64
	resources                     *corev1.ResourceRequirements
	podAntiAffinity               *corev1.PodAntiAffinity
	podSecurityContext            *corev1.PodSecurityContext
	containerSecurityContext      *corev1.SecurityContext
	imagePullPolicy               corev1.PullPolicy
}

// 通用默认值：非 root + 删掉所有 capability + 关掉特权升级 + RuntimeDefault seccomp。
// 这些是 gospec "容器以非 root 用户运行" 红线 + Pod Security Standards "restricted" profile 的最小集。
func defaultPodSecurityContext() *corev1.PodSecurityContext {
	uid := int64(65532)
	runAsNonRoot := true
	return &corev1.PodSecurityContext{
		RunAsNonRoot: &runAsNonRoot,
		RunAsUser:    &uid,
		RunAsGroup:   &uid,
		FSGroup:      &uid,
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}
}

func defaultContainerSecurityContext() *corev1.SecurityContext {
	allowPrivEsc := false
	runAsNonRoot := true
	return &corev1.SecurityContext{
		AllowPrivilegeEscalation: &allowPrivEsc,
		RunAsNonRoot:             &runAsNonRoot,
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
	}
}

// preStop 缓冲：让 K8s 把 Pod 从 Service Endpoints 摘除一段时间，再发 SIGTERM。
// 长连接型 frontier 必须有，避免在 endpoint 还没传播到 kube-proxy 时连接被秒断。
func defaultLifecycle(preStopSleepSeconds int) *corev1.Lifecycle {
	return &corev1.Lifecycle{
		PreStop: &corev1.LifecycleHandler{
			Exec: &corev1.ExecAction{
				Command: []string{"/bin/sh", "-c", "sleep " + intToStr(preStopSleepSeconds)},
			},
		},
	}
}

// 跨 host 反亲和（preferred 而非 required，避免在小集群里调度不出去）。
func preferredAntiAffinityByHost(appLabel string) *corev1.PodAntiAffinity {
	return &corev1.PodAntiAffinity{
		PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
			{
				Weight: 100,
				PodAffinityTerm: corev1.PodAffinityTerm{
					TopologyKey: "kubernetes.io/hostname",
					LabelSelector: labelSelectorForApp(appLabel),
				},
			},
		},
	}
}

// frontierDefaults / frontlasDefaults 由 ensureXxxDeployment 调用，参数化各自端口。
func frontierDefaults(servicePort, edgePort int32) componentDefaults {
	return componentDefaults{
		livenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt32(edgePort)},
			},
			InitialDelaySeconds: 10,
			PeriodSeconds:       20,
			TimeoutSeconds:      3,
			FailureThreshold:    3,
		},
		readinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt32(servicePort)},
			},
			InitialDelaySeconds: 5,
			PeriodSeconds:       10,
			TimeoutSeconds:      3,
			FailureThreshold:    3,
		},
		lifecycle:                     defaultLifecycle(10),
		terminationGracePeriodSeconds: 60,
		podSecurityContext:            defaultPodSecurityContext(),
		containerSecurityContext:      defaultContainerSecurityContext(),
		imagePullPolicy:               corev1.PullIfNotPresent,
	}
}

func frontlasDefaults(controlPort int32) componentDefaults {
	return componentDefaults{
		livenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt32(controlPort)},
			},
			InitialDelaySeconds: 10,
			PeriodSeconds:       20,
			TimeoutSeconds:      3,
			FailureThreshold:    3,
		},
		readinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Port: intstr.FromInt32(controlPort),
					Path: "/cluster/v1/health",
				},
			},
			PeriodSeconds: 5,
		},
		lifecycle:                     defaultLifecycle(5),
		terminationGracePeriodSeconds: 30,
		podSecurityContext:            defaultPodSecurityContext(),
		containerSecurityContext:      defaultContainerSecurityContext(),
		imagePullPolicy:               corev1.PullIfNotPresent,
	}
}

// pickXxx 是用户覆盖优先 / 默认兜底的小工具。
// 任一字段被显式置 nil/零值都视为 "用默认"——这与 K8s
// 资源 spec "未设置走默认" 的语义一致。

func pickResources(o, def *corev1.ResourceRequirements) *corev1.ResourceRequirements {
	if o != nil {
		return o
	}
	return def
}

func pickProbe(o, def *corev1.Probe) *corev1.Probe {
	if o != nil {
		return o
	}
	return def
}

func pickLifecycle(o, def *corev1.Lifecycle) *corev1.Lifecycle {
	if o != nil {
		return o
	}
	return def
}

func pickContainerSecurityContext(o, def *corev1.SecurityContext) *corev1.SecurityContext {
	if o != nil {
		return o
	}
	return def
}

func pickPodSecurityContext(o, def *corev1.PodSecurityContext) *corev1.PodSecurityContext {
	if o != nil {
		return o
	}
	return def
}

func pickPullPolicy(o, def corev1.PullPolicy) corev1.PullPolicy {
	if o != "" {
		return o
	}
	return def
}

func pickGrace(o *int64, def int64) int64 {
	if o != nil {
		return *o
	}
	return def
}

// pickAffinity：用户给 affinity 优先；否则用 operator 默认 PodAntiAffinity 打底，
// 同时保留 spec.<component>.NodeAffinity（v1alpha1 老字段）走 NodeAffinity 槽位。
// 当用户传完整 PodOverrides.Affinity 时，老的 NodeAffinity 字段被忽略。
func pickAffinity(o *corev1.Affinity, defaultAnti *corev1.PodAntiAffinity, legacyNodeAff *corev1.NodeAffinity) *corev1.Affinity {
	if o != nil {
		return o
	}
	out := &corev1.Affinity{
		PodAntiAffinity: defaultAnti,
	}
	if legacyNodeAff != nil && (len(legacyNodeAff.PreferredDuringSchedulingIgnoredDuringExecution) > 0 || legacyNodeAff.RequiredDuringSchedulingIgnoredDuringExecution != nil) {
		out.NodeAffinity = legacyNodeAff
	}
	return out
}

// 把 PodOverrides 里和 PodSpec 直接对应的"列表/标量"取出来，方便 deployment.go
// 一次塞进 podtemplatespec builder。
type podSpecOverrides struct {
	NodeSelector              map[string]string
	Tolerations               []corev1.Toleration
	TopologySpreadConstraints []corev1.TopologySpreadConstraint
	PriorityClassName         string
	ServiceAccountName        string
	ImagePullSecrets          []corev1.LocalObjectReference
	Annotations               map[string]string
	Labels                    map[string]string
}

func extractPodSpecOverrides(o v1alpha1.PodOverrides) podSpecOverrides {
	return podSpecOverrides{
		NodeSelector:              o.NodeSelector,
		Tolerations:               o.Tolerations,
		TopologySpreadConstraints: o.TopologySpreadConstraints,
		PriorityClassName:         o.PriorityClassName,
		ServiceAccountName:        o.ServiceAccountName,
		ImagePullSecrets:          o.ImagePullSecrets,
		Annotations:               o.Annotations,
		Labels:                    o.Labels,
	}
}
