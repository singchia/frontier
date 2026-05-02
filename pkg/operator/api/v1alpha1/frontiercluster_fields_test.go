package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newCluster(name, ns string) *FrontierCluster {
	return &FrontierCluster{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
	}
}

// 回归 M1 fix #3：Operator 自管 Secret 名应包含连字符，避免与其它资源命名风格冲突。
func TestEBTLSOperatorSecretNames_HaveDashSeparator(t *testing.T) {
	fc := newCluster("foo", "default")

	caName := fc.EBTLSOperatorCASecretNamespacedName().Name
	ckName := fc.EBTLSOperatorCertKeyNamespacedName().Name

	assert.Equal(t, "foo-edgebound-ca-certificate", caName)
	assert.Equal(t, "foo-edgebound-certkey-certificate", ckName)
	assert.NotEqual(t, caName, ckName, "CA secret 名不应等于 CertKey secret 名")
}

// 回归 M1 fix #4：fpport 不再硬编码 40012，应可由 ControlPlane.FrontierPlanePort 配置。
func TestFrontlasServicePort_FrontierPlanePort(t *testing.T) {
	t.Run("默认 40012", func(t *testing.T) {
		fc := newCluster("foo", "default")
		_, _, _, fp := fc.FrontlasServicePort()
		assert.Equal(t, int32(40012), fp.Port)
		assert.Equal(t, int32(40012), fp.TargetPort.IntVal)
	})

	t.Run("可由 spec 覆盖", func(t *testing.T) {
		fc := newCluster("foo", "default")
		fc.Spec.Frontlas.ControlPlane.FrontierPlanePort = 50012
		_, _, _, fp := fc.FrontlasServicePort()
		assert.Equal(t, int32(50012), fp.Port)
		assert.Equal(t, int32(50012), fp.TargetPort.IntVal)
	})

	t.Run("NodePort 模式下 fpport 跟随自定义端口", func(t *testing.T) {
		fc := newCluster("foo", "default")
		fc.Spec.Frontlas.ControlPlane.FrontierPlanePort = 30412
		fc.Spec.Frontlas.ControlPlane.ServiceType = corev1.ServiceTypeNodePort
		_, _, _, fp := fc.FrontlasServicePort()
		assert.Equal(t, int32(30412), fp.Port)
		assert.Equal(t, int32(30412), fp.NodePort)
	})

	t.Run("cpport 与 fpport 解耦", func(t *testing.T) {
		fc := newCluster("foo", "default")
		fc.Spec.Frontlas.ControlPlane.Port = 40021
		fc.Spec.Frontlas.ControlPlane.FrontierPlanePort = 40022
		_, _, cp, fp := fc.FrontlasServicePort()
		assert.Equal(t, int32(40021), cp.Port)
		assert.Equal(t, int32(40022), fp.Port)
	})
}
