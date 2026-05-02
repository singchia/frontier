package controller

import (
	"github.com/singchia/frontier/operator/api/v1alpha1"
	"github.com/singchia/frontier/operator/pkg/util/result"
	"github.com/singchia/frontier/operator/pkg/util/status"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// severity indicates the severity level
// at which the message should be logged
type severity string

const (
	Info  severity = "INFO"
	Debug severity = "DEBUG"
	Warn  severity = "WARN"
	Error severity = "ERROR"
	None  severity = "NONE"
)

// optionBuilder is in charge of constructing a slice of options that
// will be applied on top of the FrontierCluster resource that has been provided
type optionBuilder struct {
	options []status.Option
}

// GetOptions implements the OptionBuilder interface
func (o *optionBuilder) GetOptions() []status.Option {
	return o.options
}

// statusOptions returns an initialized optionBuilder
func statusOptions() *optionBuilder {
	return &optionBuilder{
		options: []status.Option{},
	}
}

func (o *optionBuilder) withPhase(phase v1alpha1.Phase, retryAfter int) *optionBuilder {
	o.options = append(o.options,
		phaseOption{
			phase:      phase,
			retryAfter: retryAfter,
		})
	return o
}

type message struct {
	messageString string
	severityLevel severity
}

type messageOption struct {
	message message
}

func (m messageOption) ApplyOption(fc *v1alpha1.FrontierCluster) {
	fc.Status.Message = m.message.messageString
	if m.message.severityLevel == Error {
		zap.S().Error(m.message.messageString)
	}
	if m.message.severityLevel == Warn {
		zap.S().Warn(m.message.messageString)
	}
	if m.message.severityLevel == Info {
		zap.S().Info(m.message.messageString)
	}
	if m.message.severityLevel == Debug {
		zap.S().Debug(m.message.messageString)
	}
}

func (m messageOption) GetResult() (reconcile.Result, error) {
	return result.OK()
}

func (o *optionBuilder) withMessage(severityLevel severity, msg string) *optionBuilder {
	o.options = append(o.options, messageOption{
		message: message{
			messageString: msg,
			severityLevel: severityLevel,
		},
	})
	return o
}

func (o *optionBuilder) withFailedPhase() *optionBuilder {
	return o.withPhase(v1alpha1.Failed, 0)
}

func (o *optionBuilder) withPendingPhase(retryAfter int) *optionBuilder {
	return o.withPhase(v1alpha1.Pending, retryAfter)
}

func (o *optionBuilder) withRunningPhase() *optionBuilder {
	return o.withPhase(v1alpha1.Running, -1)
}

func (o *optionBuilder) withReadyReplicas(frontier, frontlas int32) *optionBuilder {
	o.options = append(o.options, readyReplicasOption{
		frontierReady: frontier,
		frontlasReady: frontlas,
	})
	return o
}

type readyReplicasOption struct {
	frontierReady int32
	frontlasReady int32
}

func (r readyReplicasOption) ApplyOption(fc *v1alpha1.FrontierCluster) {
	fc.Status.FrontierReadyReplicas = r.frontierReady
	fc.Status.FrontlasReadyReplicas = r.frontlasReady
}

func (r readyReplicasOption) GetResult() (reconcile.Result, error) {
	return result.OK()
}

type phaseOption struct {
	phase      v1alpha1.Phase
	retryAfter int
}

func (p phaseOption) ApplyOption(fc *v1alpha1.FrontierCluster) {
	fc.Status.Phase = p.phase
	// 把 Phase 同步到 Conditions——Phase 是 deprecated 标记位但仍会被 printcolumn 用，
	// Conditions 是现代 K8s 状态查询入口，两者一起更新避免错位。
	fc.Status.ObservedGeneration = fc.Generation
	switch p.phase {
	case v1alpha1.Running:
		setCondition(&fc.Status.Conditions, fc.Generation, metav1.Condition{
			Type:    v1alpha1.ConditionAvailable,
			Status:  metav1.ConditionTrue,
			Reason:  "AllComponentsReady",
			Message: fc.Status.Message,
		})
		setCondition(&fc.Status.Conditions, fc.Generation, metav1.Condition{
			Type:    v1alpha1.ConditionProgressing,
			Status:  metav1.ConditionFalse,
			Reason:  "ReconcileSucceeded",
			Message: "Reconcile cycle completed successfully",
		})
		setCondition(&fc.Status.Conditions, fc.Generation, metav1.Condition{
			Type:    v1alpha1.ConditionDegraded,
			Status:  metav1.ConditionFalse,
			Reason:  "AllComponentsReady",
			Message: "",
		})
	case v1alpha1.Pending:
		setCondition(&fc.Status.Conditions, fc.Generation, metav1.Condition{
			Type:    v1alpha1.ConditionAvailable,
			Status:  metav1.ConditionFalse,
			Reason:  "ComponentsNotReady",
			Message: fc.Status.Message,
		})
		setCondition(&fc.Status.Conditions, fc.Generation, metav1.Condition{
			Type:    v1alpha1.ConditionProgressing,
			Status:  metav1.ConditionTrue,
			Reason:  "ReconcileInProgress",
			Message: fc.Status.Message,
		})
		setCondition(&fc.Status.Conditions, fc.Generation, metav1.Condition{
			Type:    v1alpha1.ConditionDegraded,
			Status:  metav1.ConditionFalse,
			Reason:  "ReconcileInProgress",
			Message: "",
		})
	case v1alpha1.Failed:
		setCondition(&fc.Status.Conditions, fc.Generation, metav1.Condition{
			Type:    v1alpha1.ConditionAvailable,
			Status:  metav1.ConditionFalse,
			Reason:  "ReconcileFailed",
			Message: fc.Status.Message,
		})
		setCondition(&fc.Status.Conditions, fc.Generation, metav1.Condition{
			Type:    v1alpha1.ConditionProgressing,
			Status:  metav1.ConditionFalse,
			Reason:  "ReconcileFailed",
			Message: fc.Status.Message,
		})
		setCondition(&fc.Status.Conditions, fc.Generation, metav1.Condition{
			Type:    v1alpha1.ConditionDegraded,
			Status:  metav1.ConditionTrue,
			Reason:  "ReconcileFailed",
			Message: fc.Status.Message,
		})
	}
}

func (p phaseOption) GetResult() (reconcile.Result, error) {
	if p.phase == v1alpha1.Running {
		return result.OK()
	}
	if p.phase == v1alpha1.Pending {
		return result.Retry(p.retryAfter)
	}
	if p.phase == v1alpha1.Failed {
		return result.Failed()
	}
	return result.OK()
}

// setCondition 是 K8s 风格的 condition upsert：按 Type 找到位置，
// 状态变化时刷新 LastTransitionTime；不变则只更新 Message/Reason/ObservedGeneration。
func setCondition(conds *[]metav1.Condition, gen int64, c metav1.Condition) {
	c.ObservedGeneration = gen
	if c.LastTransitionTime.IsZero() {
		c.LastTransitionTime = metav1.Now()
	}
	for i := range *conds {
		if (*conds)[i].Type != c.Type {
			continue
		}
		if (*conds)[i].Status == c.Status {
			// 状态未变，保留原 LastTransitionTime
			c.LastTransitionTime = (*conds)[i].LastTransitionTime
		}
		(*conds)[i] = c
		return
	}
	*conds = append(*conds, c)
}
