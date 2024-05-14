package controller

import (
	"github.com/singchia/frontier/operator/api/v1alpha1"
	"github.com/singchia/frontier/operator/pkg/util/result"
	"github.com/singchia/frontier/operator/pkg/util/status"
	"go.uber.org/zap"
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
// will be applied on top of the MongoDB resource that has been provided
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

type phaseOption struct {
	phase      v1alpha1.Phase
	retryAfter int
}

func (p phaseOption) ApplyOption(fc *v1alpha1.FrontierCluster) {
	fc.Status.Phase = p.phase
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
