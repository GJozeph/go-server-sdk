package ldclient

import (
	"encoding/json"
	"fmt"

	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"
)

// FeatureFlagsState is a snapshot of the state of all feature flags with regard to a
// specific user, generated by calling LDClient.AllFlagsState(). Serializing this object
// to JSON using json.Marshal will produce the appropriate data structure for
// bootstrapping the LaunchDarkly JavaScript client.
type FeatureFlagsState struct {
	flagValues   map[string]ldvalue.Value
	flagMetadata map[string]flagMetadata
	valid        bool
}

type flagMetadata struct {
	Variation            *int             `json:"variation,omitempty"`
	Version              *int             `json:"version,omitempty"`
	Reason               EvaluationReason `json:"reason,omitempty"`
	TrackEvents          *bool            `json:"trackEvents,omitempty"`
	DebugEventsUntilDate *uint64          `json:"debugEventsUntilDate,omitempty"`
}

// FlagsStateOption is the type of optional parameters that can be passed to LDClient.AllFlagsState.
type FlagsStateOption interface {
	fmt.Stringer
}

// ClientSideOnly - when passed to LDClient.AllFlagsState() - specifies that only flags marked
// for use with the client-side SDK should be included in the state object. By default, all
// flags are included.
var ClientSideOnly FlagsStateOption = clientSideOnlyFlagsStateOption{}

type clientSideOnlyFlagsStateOption struct{}

func (o clientSideOnlyFlagsStateOption) String() string {
	return "ClientSideOnly"
}

// WithReasons - when passed to LDClient.AllFlagsState() - specifies that evaluation reasons
// should be included in the state object. By default, they are not.
var WithReasons FlagsStateOption = withReasonsFlagsStateOption{}

type withReasonsFlagsStateOption struct{}

func (o withReasonsFlagsStateOption) String() string {
	return "WithReasons"
}

// DetailsOnlyForTrackedFlags - when passed to LDClient.AllFlagsState() - specifies that
// any flag metadata that is normally only used for event generation - such as flag versions
// and evaluation reasons - should be omitted for any flag that does not have event tracking
// or debugging turned on. This reduces the size of the JSON data if you are passing the flag
// state to the front end.
var DetailsOnlyForTrackedFlags FlagsStateOption = detailsOnlyForTrackedFlagsOption{}

type detailsOnlyForTrackedFlagsOption struct{}

func (o detailsOnlyForTrackedFlagsOption) String() string {
	return "DetailsOnlyForTrackedFlags"
}

func newFeatureFlagsState() FeatureFlagsState {
	return FeatureFlagsState{
		flagValues:   make(map[string]ldvalue.Value),
		flagMetadata: make(map[string]flagMetadata),
		valid:        true,
	}
}

func hasFlagsStateOption(options []FlagsStateOption, value FlagsStateOption) bool {
	for _, o := range options {
		if o == value {
			return true
		}
	}
	return false
}

func (s *FeatureFlagsState) addFlag(flag *FeatureFlag, value ldvalue.Value, variation *int, reason EvaluationReason, detailsOnlyIfTracked bool) {
	meta := flagMetadata{
		Variation:            variation,
		DebugEventsUntilDate: flag.DebugEventsUntilDate,
	}
	includeDetail := !detailsOnlyIfTracked || flag.TrackEvents
	if !includeDetail && flag.DebugEventsUntilDate != nil {
		includeDetail = *flag.DebugEventsUntilDate > now()
	}
	if includeDetail {
		meta.Version = &flag.Version
		meta.Reason = reason
	}
	if flag.TrackEvents { // omit this field if it's false, for brevity
		meta.TrackEvents = &flag.TrackEvents
	}
	s.flagValues[flag.Key] = value
	s.flagMetadata[flag.Key] = meta
}

// IsValid returns true if this object contains a valid snapshot of feature flag state, or false if the
// state could not be computed (for instance, because the client was offline or there was no user).
func (s FeatureFlagsState) IsValid() bool {
	return s.valid
}

// GetFlagValue returns the value of an individual feature flag at the time the state was recorded. The
// return value will be ldvalue.Null() if the flag returned the default value, or if there was no such flag.
func (s FeatureFlagsState) GetFlagValue(key string) ldvalue.Value {
	return s.flagValues[key]
}

// GetFlagReason returns the evaluation reason for an individual feature flag at the time the state was
// recorded. The return value will be nil if reasons were not recorded, or if there was no such flag.
func (s FeatureFlagsState) GetFlagReason(key string) EvaluationReason {
	if m, ok := s.flagMetadata[key]; ok {
		return m.Reason
	}
	return nil
}

// ToValuesMap returns a map of flag keys to flag values. If a flag would have evaluated to the default
// value, its value will be nil.
//
// Do not use this method if you are passing data to the front end to "bootstrap" the JavaScript client.
// Instead, convert the state object to JSON using json.Marshal.
func (s FeatureFlagsState) ToValuesMap() map[string]ldvalue.Value {
	return s.flagValues
}

// MarshalJSON implements a custom JSON serialization for FeatureFlagsState, to produce the correct
// data structure for "bootstrapping" the LaunchDarkly JavaScript client.
func (s FeatureFlagsState) MarshalJSON() ([]byte, error) {
	var outerMap = make(map[string]interface{}, len(s.flagValues)+2)
	for k, v := range s.flagValues {
		outerMap[k] = v
	}
	outerMap["$flagsState"] = s.flagMetadata
	outerMap["$valid"] = s.valid
	return json.Marshal(outerMap)
}
