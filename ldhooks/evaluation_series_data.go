package ldhooks

// EvaluationSeriesData is an immutable data type used for passing implementation-specific data between stages in the
// evaluation series.
type EvaluationSeriesData struct {
	data map[string]any
}

// EvaluationSeriesDataBuilder should be used by hook implementers to append data
type EvaluationSeriesDataBuilder struct {
	data map[string]any
}

// EmptyEvaluationSeriesData returns empty series data. This function is not intended for use by hook implementors.
// Hook implementations should always use NewEvaluationSeriesBuilder.
func EmptyEvaluationSeriesData() EvaluationSeriesData {
	return EvaluationSeriesData{
		data: make(map[string]any),
	}
}

// NewEvaluationSeriesBuilder creates an EvaluationSeriesDataBuilder based on the provided EvaluationSeriesData.
//
//	func(h MyHook) BeforeEvaluation(seriesContext EvaluationSeriesContext,
//		data EvaluationSeriesData) EvaluationSeriesData {
//		// Some hook functionality.
//		return NewEvaluationSeriesBuilder(data).Set("my-key", myValue).Build()
//	}
func NewEvaluationSeriesBuilder(data EvaluationSeriesData) EvaluationSeriesDataBuilder {
	newData := make(map[string]any, len(data.data))
	for k, v := range data.data {
		newData[k] = v
	}
	return EvaluationSeriesDataBuilder{
		data: newData,
	}
}

func (b EvaluationSeriesDataBuilder) Set(key string, value any) EvaluationSeriesDataBuilder {
	b.data[key] = value
	return b
}

func (b EvaluationSeriesDataBuilder) SetFromMap(newValues map[string]any) EvaluationSeriesDataBuilder {
	for k, v := range newValues {
		b.data[k] = v
	}
	return b
}

func (b EvaluationSeriesData) Get(key string) (any, bool) {
	val, ok := b.data[key]
	return val, ok
}

func (b EvaluationSeriesDataBuilder) Build() EvaluationSeriesData {
	newData := make(map[string]any, len(b.data))
	for k, v := range b.data {
		newData[k] = v
	}
	return EvaluationSeriesData{
		data: newData,
	}
}
