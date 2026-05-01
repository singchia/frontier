package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/singchia/frontier/operator/api/v1alpha1"
)

func TestPickPullPolicy(t *testing.T) {
	assert.Equal(t, corev1.PullIfNotPresent, pickPullPolicy("", corev1.PullIfNotPresent))
	assert.Equal(t, corev1.PullAlways, pickPullPolicy(corev1.PullAlways, corev1.PullIfNotPresent))
	assert.Equal(t, corev1.PullNever, pickPullPolicy(corev1.PullNever, corev1.PullIfNotPresent))
}

func TestPickResources_NilFallsBackToDefault(t *testing.T) {
	def := &corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU: resource.MustParse("100m"),
		},
	}
	assert.Same(t, def, pickResources(nil, def))

	custom := &corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU: resource.MustParse("2"),
		},
	}
	assert.Same(t, custom, pickResources(custom, def))
}

func TestPickGrace_DefaultWhenNil(t *testing.T) {
	assert.Equal(t, int64(60), pickGrace(nil, 60))
	v := int64(120)
	assert.Equal(t, int64(120), pickGrace(&v, 60))
}

func TestPickAffinity_UserAffinityFullyWins(t *testing.T) {
	defaultAnti := preferredAntiAffinityByHost("foo")
	userAff := &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{},
		},
	}
	got := pickAffinity(userAff, defaultAnti, nil)
	assert.Same(t, userAff, got)
	assert.Nil(t, got.PodAntiAffinity, "用户 Affinity 全量替换默认，不再叠加 default anti-affinity")
}

func TestPickAffinity_DefaultsKeepLegacyNodeAffinity(t *testing.T) {
	defaultAnti := preferredAntiAffinityByHost("foo")
	legacy := &corev1.NodeAffinity{
		PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
			{Weight: 1},
		},
	}
	got := pickAffinity(nil, defaultAnti, legacy)
	assert.NotNil(t, got)
	assert.Same(t, defaultAnti, got.PodAntiAffinity)
	assert.Same(t, legacy, got.NodeAffinity)
}

func TestPickAffinity_EmptyLegacyAffinityIsIgnored(t *testing.T) {
	defaultAnti := preferredAntiAffinityByHost("foo")
	emptyLegacy := &corev1.NodeAffinity{} // 用户没填 NodeAffinity 时会传一个零值结构体进来
	got := pickAffinity(nil, defaultAnti, emptyLegacy)
	assert.Nil(t, got.NodeAffinity, "空 NodeAffinity 不应进 PodSpec")
	assert.Same(t, defaultAnti, got.PodAntiAffinity)
}

func TestPreferredAntiAffinityByHost_IsPreferredNotRequired(t *testing.T) {
	a := preferredAntiAffinityByHost("frontier")
	assert.NotNil(t, a)
	assert.Empty(t, a.RequiredDuringSchedulingIgnoredDuringExecution, "必须是 preferred 而非 required；否则小集群副本超 node 数即调度失败")
	assert.Len(t, a.PreferredDuringSchedulingIgnoredDuringExecution, 1)
	assert.Equal(t, int32(100), a.PreferredDuringSchedulingIgnoredDuringExecution[0].Weight)
	assert.Equal(t, "kubernetes.io/hostname", a.PreferredDuringSchedulingIgnoredDuringExecution[0].PodAffinityTerm.TopologyKey)
}

func TestDefaultPodSecurityContext_NonRoot(t *testing.T) {
	psc := defaultPodSecurityContext()
	assert.NotNil(t, psc.RunAsNonRoot)
	assert.True(t, *psc.RunAsNonRoot, "默认必须 nonRoot——gospec 安全红线")
	assert.NotNil(t, psc.RunAsUser)
	assert.NotZero(t, *psc.RunAsUser)
	assert.NotNil(t, psc.SeccompProfile)
	assert.Equal(t, corev1.SeccompProfileTypeRuntimeDefault, psc.SeccompProfile.Type)
}

func TestDefaultContainerSecurityContext_DropAll(t *testing.T) {
	c := defaultContainerSecurityContext()
	assert.NotNil(t, c.AllowPrivilegeEscalation)
	assert.False(t, *c.AllowPrivilegeEscalation)
	assert.NotNil(t, c.Capabilities)
	assert.Contains(t, c.Capabilities.Drop, corev1.Capability("ALL"))
}

func TestFrontlasRedisEnvs_PasswordSecretWinsOverPlaintext(t *testing.T) {
	fc := v1alpha1.FrontierCluster{}
	fc.Spec.Frontlas.Redis = v1alpha1.Redis{
		Addrs:    []string{"r:6379"},
		Password: "should-not-leak",
		PasswordSecret: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: "redis-creds"},
			Key:                  "password",
		},
		RedisType: v1alpha1.RedisTypeStandalone,
	}
	envs := frontlasRedisEnvs(fc, 40011, 40012)

	var passwordEnv *corev1.EnvVar
	for i := range envs {
		if envs[i].Name == FrontlasRedisPasswordEnv {
			passwordEnv = &envs[i]
		}
	}
	assert.NotNil(t, passwordEnv)
	assert.Empty(t, passwordEnv.Value, "PasswordSecret 存在时禁止把明文 Password 写入 env")
	assert.NotNil(t, passwordEnv.ValueFrom)
	assert.NotNil(t, passwordEnv.ValueFrom.SecretKeyRef)
	assert.Equal(t, "redis-creds", passwordEnv.ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, "password", passwordEnv.ValueFrom.SecretKeyRef.Key)
}

func TestFrontlasRedisEnvs_FallbackToPlaintext(t *testing.T) {
	fc := v1alpha1.FrontierCluster{}
	fc.Spec.Frontlas.Redis = v1alpha1.Redis{
		Addrs:     []string{"r:6379"},
		Password:  "legacy-plaintext",
		RedisType: v1alpha1.RedisTypeStandalone,
	}
	envs := frontlasRedisEnvs(fc, 40011, 40012)

	var passwordEnv *corev1.EnvVar
	for i := range envs {
		if envs[i].Name == FrontlasRedisPasswordEnv {
			passwordEnv = &envs[i]
		}
	}
	assert.NotNil(t, passwordEnv)
	assert.Equal(t, "legacy-plaintext", passwordEnv.Value)
	assert.Nil(t, passwordEnv.ValueFrom)
}

func TestFrontierDefaults_HasGoodSafetyDefaults(t *testing.T) {
	d := frontierDefaults(30011, 30012)
	assert.NotNil(t, d.livenessProbe)
	assert.NotNil(t, d.readinessProbe)
	assert.NotNil(t, d.lifecycle, "preStop 必须有，避免长连接秒断")
	assert.NotNil(t, d.lifecycle.PreStop)
	assert.Greater(t, d.terminationGracePeriodSeconds, int64(0))
	assert.Equal(t, corev1.PullIfNotPresent, d.imagePullPolicy)
	assert.NotNil(t, d.podSecurityContext)
	assert.NotNil(t, d.containerSecurityContext)
}
