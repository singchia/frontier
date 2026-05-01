package controller

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// errSecretGetter 模拟 Get 调用失败（非 NotFound），用于回归 fix #2：
// getEBCAFromSecret 必须把读 Secret 的错误向上传播，禁止吞成 (空字符串, nil)。
type errSecretGetter struct {
	err error
}

func (e errSecretGetter) GetSecret(ctx context.Context, key client.ObjectKey) (corev1.Secret, error) {
	return corev1.Secret{}, e.err
}

type fakeSecretGetter struct {
	secret corev1.Secret
}

func (f fakeSecretGetter) GetSecret(ctx context.Context, key client.ObjectKey) (corev1.Secret, error) {
	if f.secret.Name == key.Name && f.secret.Namespace == key.Namespace {
		return f.secret, nil
	}
	return corev1.Secret{}, &apierrors.StatusError{ErrStatus: metav1.Status{Reason: metav1.StatusReasonNotFound}}
}

func TestGetEBCAFromSecret_PropagatesReadError(t *testing.T) {
	ctx := context.Background()
	wantErr := errors.New("boom: api server unreachable")
	getter := errSecretGetter{err: wantErr}

	ca, err := getEBCAFromSecret(ctx, getter, types.NamespacedName{Namespace: "ns", Name: "ca"})

	assert.Empty(t, ca)
	assert.Error(t, err, "读 Secret 失败必须向上传播，不能被静默成 nil")
	assert.ErrorIs(t, err, wantErr)
}

func TestGetEBCAFromSecret_MissingCAKey(t *testing.T) {
	ctx := context.Background()
	getter := fakeSecretGetter{
		secret: corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "ca", Namespace: "ns"},
			Data:       map[string][]byte{"other.crt": []byte("x")},
		},
	}

	ca, err := getEBCAFromSecret(ctx, getter, types.NamespacedName{Namespace: "ns", Name: "ca"})

	assert.Empty(t, ca)
	assert.ErrorIs(t, err, ErrCANotFoundInSecret)
}

func TestGetEBCAFromSecret_Happy(t *testing.T) {
	ctx := context.Background()
	getter := fakeSecretGetter{
		secret: corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "ca", Namespace: "ns"},
			Data:       map[string][]byte{"ca.crt": []byte("PEM")},
		},
	}

	ca, err := getEBCAFromSecret(ctx, getter, types.NamespacedName{Namespace: "ns", Name: "ca"})

	assert.NoError(t, err)
	assert.Equal(t, "PEM", ca)
}
