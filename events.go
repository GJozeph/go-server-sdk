package ldclient

import (
	"time"

	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"
)

// An Event represents an analytics event generated by the client, which will be passed to
// the EventProcessor.  The event data that the EventProcessor actually sends to LaunchDarkly
// may be slightly different.
type Event interface {
	GetBase() BaseEvent
}

// BaseEvent provides properties common to all events.
type BaseEvent struct {
	CreationDate uint64
	User         User
}

// FeatureRequestEvent is generated by evaluating a feature flag or one of a flag's prerequisites.
type FeatureRequestEvent struct {
	BaseEvent
	Key       string
	Variation *int
	Value     ldvalue.Value
	Default   ldvalue.Value
	Version   *int
	PrereqOf  *string
	Reason    EvaluationReasonContainer
	// Note, we need to use EvaluationReasonContainer here because FeatureRequestEvent can be
	// deserialized by ld-relay.
	TrackEvents          bool
	Debug                bool
	DebugEventsUntilDate *uint64
}

// CustomEvent is generated by calling the client's Track method.
type CustomEvent struct {
	BaseEvent
	Key         string
	Data        ldvalue.Value
	MetricValue *float64
}

// IdentifyEvent is generated by calling the client's Identify method.
type IdentifyEvent struct {
	BaseEvent
}

// IndexEvent is generated internally to capture user details from other events.
type IndexEvent struct {
	BaseEvent
}

// NewFeatureRequestEvent creates a feature request event. Normally, you don't need to call this;
// the event is created and queued automatically during feature flag evaluation.
//
// Deprecated: This function will be removed in a later release.
func NewFeatureRequestEvent(key string, flag *FeatureFlag, user User, variation *int, value, defaultVal ldvalue.Value, prereqOf *string) FeatureRequestEvent {
	fre := FeatureRequestEvent{
		BaseEvent: BaseEvent{
			CreationDate: now(),
			User:         user,
		},
		Key:       key,
		Variation: variation,
		Value:     value,
		Default:   defaultVal,
		PrereqOf:  prereqOf,
	}
	if flag != nil {
		fre.Version = &flag.Version
		fre.TrackEvents = flag.TrackEvents
		fre.DebugEventsUntilDate = flag.DebugEventsUntilDate
	}
	return fre
}

func newUnknownFlagEvent(key string, user User, defaultVal ldvalue.Value, reason EvaluationReason,
	includeReason bool) FeatureRequestEvent {
	fre := FeatureRequestEvent{
		BaseEvent: BaseEvent{
			CreationDate: now(),
			User:         user,
		},
		Key:     key,
		Value:   defaultVal,
		Default: defaultVal,
	}
	if includeReason {
		fre.Reason.Reason = reason
	}
	return fre
}

func newSuccessfulEvalEvent(flag *FeatureFlag, user User, variation *int, value, defaultVal ldvalue.Value,
	reason EvaluationReason, includeReason bool, prereqOf *string) FeatureRequestEvent {
	requireExperimentData := isExperiment(flag, reason)
	fre := FeatureRequestEvent{
		BaseEvent: BaseEvent{
			CreationDate: now(),
			User:         user,
		},
		Key:                  flag.Key,
		Version:              &flag.Version,
		Variation:            variation,
		Value:                value,
		Default:              defaultVal,
		PrereqOf:             prereqOf,
		TrackEvents:          requireExperimentData || flag.TrackEvents,
		DebugEventsUntilDate: flag.DebugEventsUntilDate,
	}
	if includeReason || requireExperimentData {
		fre.Reason.Reason = reason
	}
	return fre
}

func isExperiment(flag *FeatureFlag, reason EvaluationReason) bool {
	if reason == nil {
		return false
	}
	switch reason.GetKind() {
	case EvalReasonFallthrough:
		return flag.TrackEventsFallthrough
	case EvalReasonRuleMatch:
		i := reason.GetRuleIndex()
		if i >= 0 && i < len(flag.Rules) {
			return flag.Rules[i].TrackEvents
		}
	}
	return false
}

// GetBase returns the BaseEvent
func (evt FeatureRequestEvent) GetBase() BaseEvent {
	return evt.BaseEvent
}

func newCustomEvent(key string, user User, data ldvalue.Value, withMetric bool, metricValue float64) CustomEvent {
	ce := CustomEvent{
		BaseEvent: BaseEvent{
			CreationDate: now(),
			User:         user,
		},
		Key:  key,
		Data: data,
	}
	if withMetric {
		ce.MetricValue = &metricValue
	}
	return ce
}

// GetBase returns the BaseEvent
func (evt CustomEvent) GetBase() BaseEvent {
	return evt.BaseEvent
}

// NewIdentifyEvent constructs a new identify event, but does not send it. Typically, Identify should be used to both create the
// event and send it to LaunchDarkly.
func NewIdentifyEvent(user User) IdentifyEvent {
	return IdentifyEvent{
		BaseEvent: BaseEvent{
			CreationDate: now(),
			User:         user,
		},
	}
}

// GetBase returns the BaseEvent
func (evt IdentifyEvent) GetBase() BaseEvent {
	return evt.BaseEvent
}

// GetBase returns the BaseEvent
func (evt IndexEvent) GetBase() BaseEvent {
	return evt.BaseEvent
}

func now() uint64 {
	return toUnixMillis(time.Now())
}

func toUnixMillis(t time.Time) uint64 {
	ms := time.Duration(t.UnixNano()) / time.Millisecond

	return uint64(ms)
}
