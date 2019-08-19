package ldclient

// The types in this file are for analytics event data structures that we send to
// LaunchDarkly.

// Serializable form of a feature request event. This differs from the event that was
// passed in to us in that it usually has a user key instead of a user object.
type featureRequestEventOutput struct {
	Kind         string            `json:"kind"`
	CreationDate uint64            `json:"creationDate"`
	Key          string            `json:"key"`
	UserKey      *string           `json:"userKey,omitempty"`
	User         *serializableUser `json:"user,omitempty"`
	Variation    *int              `json:"variation,omitempty"`
	Value        interface{}       `json:"value"`
	Default      interface{}       `json:"default"`
	Version      *int              `json:"version,omitempty"`
	PrereqOf     *string           `json:"prereqOf,omitempty"`
	Reason       EvaluationReason  `json:"reason,omitempty"`
}

// Serializable form of an identify event.
type identifyEventOutput struct {
	Kind         string            `json:"kind"`
	CreationDate uint64            `json:"creationDate"`
	Key          *string           `json:"key"`
	User         *serializableUser `json:"user"`
}

// Serializable form of a custom event. It has a user key instead of a user object.
type customEventOutput struct {
	Kind         string            `json:"kind"`
	CreationDate uint64            `json:"creationDate"`
	Key          string            `json:"key"`
	UserKey      *string           `json:"userKey,omitempty"`
	User         *serializableUser `json:"user,omitempty"`
	Data         interface{}       `json:"data,omitempty"`
	MetricValue  *float64          `json:"metricValue,omitempty"`
}

// Serializable form of an index event. This is not generated by an explicit client call,
// but is created automatically whenever we see a user we haven't seen before in a feature
// request event or custom event.
type indexEventOutput struct {
	Kind         string            `json:"kind"`
	CreationDate uint64            `json:"creationDate"`
	User         *serializableUser `json:"user"`
}

// Serializable form of a summary event, containing data generated by EventSummarizer.
type summaryEventOutput struct {
	Kind      string                     `json:"kind"`
	StartDate uint64                     `json:"startDate"`
	EndDate   uint64                     `json:"endDate"`
	Features  map[string]flagSummaryData `json:"features"`
}

type flagSummaryData struct {
	Default  interface{}       `json:"default"`
	Counters []flagCounterData `json:"counters"`
}

type flagCounterData struct {
	Value     interface{} `json:"value"`
	Variation *int        `json:"variation,omitempty"`
	Version   *int        `json:"version,omitempty"`
	Count     int         `json:"count"`
	Unknown   *bool       `json:"unknown,omitempty"`
}

// Event types
const (
	FeatureRequestEventKind = "feature"
	FeatureDebugEventKind   = "debug"
	CustomEventKind         = "custom"
	IdentifyEventKind       = "identify"
	IndexEventKind          = "index"
	SummaryEventKind        = "summary"
)

type eventOutputFormatter struct {
	userFilter  userFilter
	inlineUsers bool
	config      Config
}

func (ef eventOutputFormatter) makeOutputEvents(events []Event, summary eventSummary) []interface{} {
	out := make([]interface{}, 0, len(events)+1) // leave room for summary, if any
	for _, e := range events {
		oe := ef.makeOutputEvent(e)
		if oe != nil {
			out = append(out, oe)
		}
	}
	if len(summary.counters) > 0 {
		out = append(out, ef.makeSummaryEvent(summary))
	}
	return out
}

func (ef eventOutputFormatter) makeOutputEvent(evt interface{}) interface{} {
	switch evt := evt.(type) {
	case FeatureRequestEvent:
		fe := featureRequestEventOutput{
			CreationDate: evt.BaseEvent.CreationDate,
			Key:          evt.Key,
			Variation:    evt.Variation,
			Value:        evt.Value,
			Default:      evt.Default,
			Version:      evt.Version,
			PrereqOf:     evt.PrereqOf,
			Reason:       evt.Reason.Reason,
		}
		if ef.inlineUsers || evt.Debug {
			fe.User = ef.userFilter.scrubUser(evt.User)
		} else {
			fe.UserKey = evt.User.Key
		}
		if evt.Debug {
			fe.Kind = FeatureDebugEventKind
		} else {
			fe.Kind = FeatureRequestEventKind
		}
		return fe
	case CustomEvent:
		ce := customEventOutput{
			Kind:         CustomEventKind,
			CreationDate: evt.BaseEvent.CreationDate,
			Key:          evt.Key,
			Data:         evt.Data,
			MetricValue:  evt.MetricValue,
		}
		if ef.inlineUsers {
			ce.User = ef.userFilter.scrubUser(evt.User)
		} else {
			ce.UserKey = evt.User.Key
		}
		return ce
	case IdentifyEvent:
		return identifyEventOutput{
			Kind:         IdentifyEventKind,
			CreationDate: evt.BaseEvent.CreationDate,
			Key:          evt.User.Key,
			User:         ef.userFilter.scrubUser(evt.User),
		}
	case IndexEvent:
		return indexEventOutput{
			Kind:         IndexEventKind,
			CreationDate: evt.BaseEvent.CreationDate,
			User:         ef.userFilter.scrubUser(evt.User),
		}
	default:
		return nil
	}
}

// Transforms the summary data into the format used for event sending.
func (ef eventOutputFormatter) makeSummaryEvent(snapshot eventSummary) summaryEventOutput {
	features := make(map[string]flagSummaryData, len(snapshot.counters))
	for key, value := range snapshot.counters {
		var flagData flagSummaryData
		var known bool
		if flagData, known = features[key.key]; !known {
			flagData = flagSummaryData{
				Default:  value.flagDefault,
				Counters: make([]flagCounterData, 0, 2),
			}
		}
		data := flagCounterData{
			Value: value.flagValue,
			Count: value.count,
		}
		if key.variation != nilVariation {
			v := key.variation
			data.Variation = &v
		}
		if key.version == 0 {
			unknown := true
			data.Unknown = &unknown
		} else {
			version := key.version
			data.Version = &version
		}
		flagData.Counters = append(flagData.Counters, data)
		features[key.key] = flagData
	}

	return summaryEventOutput{
		Kind:      SummaryEventKind,
		StartDate: snapshot.startDate,
		EndDate:   snapshot.endDate,
		Features:  features,
	}
}
