/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PodOverrides 集中暴露 Frontier / Frontlas Pod 的可配置项，
// 让用户在不破坏 v1alpha1 schema 的前提下，按生产需要覆盖资源、调度、
// 安全上下文、探针、生命周期等关键字段。所有字段都是 optional——
// 不填走 operator 内置默认值（见 internal/controller/podoverrides.go）。
type PodOverrides struct {
	Resources                     *corev1.ResourceRequirements      `json:"resources,omitempty"`
	NodeSelector                  map[string]string                 `json:"nodeSelector,omitempty"`
	Tolerations                   []corev1.Toleration               `json:"tolerations,omitempty"`
	TopologySpreadConstraints     []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
	Affinity                      *corev1.Affinity                  `json:"affinity,omitempty"`
	PriorityClassName             string                            `json:"priorityClassName,omitempty"`
	ServiceAccountName            string                            `json:"serviceAccountName,omitempty"`
	ImagePullSecrets              []corev1.LocalObjectReference     `json:"imagePullSecrets,omitempty"`
	ImagePullPolicy               corev1.PullPolicy                 `json:"imagePullPolicy,omitempty"`
	Annotations                   map[string]string                 `json:"annotations,omitempty"`
	Labels                        map[string]string                 `json:"labels,omitempty"`
	PodSecurityContext            *corev1.PodSecurityContext        `json:"podSecurityContext,omitempty"`
	ContainerSecurityContext      *corev1.SecurityContext           `json:"containerSecurityContext,omitempty"`
	TerminationGracePeriodSeconds *int64                            `json:"terminationGracePeriodSeconds,omitempty"`
	LivenessProbe                 *corev1.Probe                     `json:"livenessProbe,omitempty"`
	ReadinessProbe                *corev1.Probe                     `json:"readinessProbe,omitempty"`
	Lifecycle                     *corev1.Lifecycle                 `json:"lifecycle,omitempty"`
}

// TLS is the configuration used to set up TLS encryption
type TLS struct {
	Enabled bool `json:"enabled"`

	// Optional configures if TLS should be required or optional for connections
	// +optional
	Optional bool `json:"optional"`

	// CertificateKeySecret is a reference to a Secret containing a private key and certificate to use for TLS.
	// The key and cert are expected to be PEM encoded and available at "tls.key" and "tls.crt".
	// This is the same format used for the standard "kubernetes.io/tls" Secret type, but no specific type is required.
	// Alternatively, an entry tls.pem, containing the concatenation of cert and key, can be provided.
	// If all of tls.pem, tls.crt and tls.key are present, the tls.pem one needs to be equal to the concatenation of tls.crt and tls.key
	// +optional
	CertificateKeySecret corev1.LocalObjectReference `json:"certificateKeySecretRef"`

	// CaCertificate is needed when mtls is set
	MTLS bool `json:"mtls"`
	// CaCertificateSecret is a reference to a Secret containing the certificate for the CA which signed the server certificates
	// The certificate is expected to be available under the key "ca.crt"
	// +optional
	CaCertificateSecret *corev1.LocalObjectReference `json:"caCertificateSecretRef,omitempty"`
}

type Servicebound struct {
	Port        int                `json:"port,omitempty"`
	ServiceName string             `json:"service,omitempty"`
	ServiceType corev1.ServiceType `json:"serviceType,omitempty"` // typically edgebound should and default be ClusterIP
}

type Edgebound struct {
	Port        int                `json:"port,omitempty"`
	ServiceName string             `json:"serviceName,omitempty"`
	ServiceType corev1.ServiceType `json:"serviceType,omitempty"` // typically edgebound should and default be NodePort
	TLS         TLS                `json:"tls,omitempty"`
}

type Frontier struct {
	Replicas     int                 `json:"replicas,omitempty"` // frontier replicas, default 1
	Servicebound Servicebound        `json:"servicebound"`
	Edgebound    Edgebound           `json:"edgebound"`
	Image        string              `json:"image,omitempty"` // default singchia/frontier:1.1.0
	NodeAffinity corev1.NodeAffinity `json:"nodeAffinity,omitempty"`
	// Pod is the optional set of generic Pod-level overrides applied to the
	// frontier Deployment. Fields here win over operator defaults; fields not
	// provided fall back to defaults documented in PodOverrides.
	// When Pod.Affinity is set it fully replaces the legacy NodeAffinity above.
	Pod PodOverrides `json:"pod,omitempty"`
}

type ControlPlane struct {
	Port              int                `json:"port,omitempty"`              // control plane port exposed to service-side callers, default 40011
	FrontierPlanePort int                `json:"frontierPlanePort,omitempty"` // frontier-plane port exposed to frontier nodes, default 40012
	ServiceName       string             `json:"service,omitempty"`
	ServiceType       corev1.ServiceType `json:"serviceType,omitempty"` // typically should default to ClusterIP
}

type RedisType string

const (
	RedisTypeStandalone = "standalone"
	RedisTypeSentinel   = "sentinel"
	RedisTypeCluster    = "cluster"
)

type Redis struct {
	Addrs []string `json:"addrs"`
	DB    int      `json:"db,omitempty"`
	User  string   `json:"user,omitempty"`
	// Password 是密码明文，会被原样写进 Pod 环境变量。
	// Deprecated: 生产场景请改用 PasswordSecret，避免密钥进 spec / event / describe 输出。
	Password string `json:"password,omitempty"`
	// PasswordSecret 引用一个 Secret 中的字段作为 Redis 密码来源。
	// 设置后优先级高于 Password，env 通过 valueFrom.secretKeyRef 注入。
	PasswordSecret *corev1.SecretKeySelector `json:"passwordSecret,omitempty"`
	RedisType      RedisType                 `json:"redisType"`
	MasterName     string                    `json:"masterName,omitempty"`
}

type Frontlas struct {
	Replicas     int                 `json:"replicas,omitempty"` // frontlas replicas, default 1
	ControlPlane ControlPlane        `json:"controlplane,omitempty"`
	NodeAffinity corev1.NodeAffinity `json:"nodeAffinity,omitempty"`
	Image        string              `json:"image,omitempty"`
	Redis        Redis               `json:"redis"`
	// Pod is the optional set of generic Pod-level overrides applied to the
	// frontlas Deployment. See Frontier.Pod for semantics.
	Pod PodOverrides `json:"pod,omitempty"`
}

// FrontierClusterSpec defines the desired state of FrontierCluster
type FrontierClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Frontier Frontier `json:"frontier"`
	Frontlas Frontlas `json:"frontlas"`
}

type Phase string

const (
	Running            Phase = "Running"
	Failed             Phase = "Failed"
	Pending            Phase = "Pending"
	defaultPasswordKey       = "password"

	// Keep in sync with controllers/prometheus.go
	defaultPrometheusPort = 9216
)

// FrontierClusterStatus defines the observed state of FrontierCluster
type FrontierClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// TODO scale 1 a time
	// CurrentFrontierReplicas  int `json:"currentFrontierReplicas"`
	// CurrentFrontlasReplicass int `json:"currentFrontlasReplicas"`
	Phase   Phase  `json:"phase"`
	Message string `json:"message,omitemtpy"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// FrontierCluster is the Schema for the frontierclusters API
type FrontierCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FrontierClusterSpec   `json:"spec,omitempty"`
	Status FrontierClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FrontierClusterList contains a list of FrontierCluster
type FrontierClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FrontierCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FrontierCluster{}, &FrontierClusterList{})
}
