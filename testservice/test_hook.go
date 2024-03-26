package main

import (
	"context"

	"github.com/launchdarkly/go-sdk-common/v3/ldreason"
	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
	"github.com/launchdarkly/go-server-sdk/v7/ldhooks"
	"github.com/launchdarkly/go-server-sdk/v7/testservice/servicedef"
)

type testHook struct {
	ldhooks.UnimplementedHook
	metadata        ldhooks.HookMetadata
	dataPayloads    map[servicedef.HookStage]servicedef.SDKConfigEvaluationHookData
	callbackService callbackService
}

func newTestHook(
	name string,
	endpoint string,
	data map[servicedef.HookStage]servicedef.SDKConfigEvaluationHookData,
) testHook {
	return testHook{
		metadata:        ldhooks.NewHookMetadata(name),
		dataPayloads:    data,
		callbackService: callbackService{baseURL: endpoint},
	}
}

func (t testHook) GetMetadata() ldhooks.HookMetadata {
	return t.metadata
}

func (t testHook) BeforeEvaluation(
	_ context.Context,
	seriesContext ldhooks.EvaluationSeriesContext,
	data ldhooks.EvaluationSeriesData,
) (ldhooks.EvaluationSeriesData, error) {
	err := t.callbackService.post("", servicedef.HookExecutionPayload{
		EvaluationSeriesContext: evaluationSeriesContextToService(seriesContext),
		EvaluationSeriesData:    evaluationSeriesDataToService(data),
		Stage:                   servicedef.BeforeEvaluation,
	}, nil)
	if err != nil {
		return ldhooks.EmptyEvaluationSeriesData(), err
	}
	dataBuilder := ldhooks.NewEvaluationSeriesBuilder(data)
	stageData := t.dataPayloads[servicedef.BeforeEvaluation]
	for key, value := range stageData {
		dataBuilder.Set(key, value)
	}
	return dataBuilder.Build(), nil
}

func (t testHook) AfterEvaluation(
	_ context.Context,
	seriesContext ldhooks.EvaluationSeriesContext,
	data ldhooks.EvaluationSeriesData,
	detail ldreason.EvaluationDetail,
) (ldhooks.EvaluationSeriesData, error) {
	err := t.callbackService.post("", servicedef.HookExecutionPayload{
		EvaluationSeriesContext: evaluationSeriesContextToService(seriesContext),
		EvaluationSeriesData:    evaluationSeriesDataToService(data),
		Stage:                   servicedef.AfterEvaluation,
		EvaluationDetail:        *detailToService(detail),
	}, nil)
	if err != nil {
		return ldhooks.EmptyEvaluationSeriesData(), err
	}
	dataBuilder := ldhooks.NewEvaluationSeriesBuilder(data)
	stageData := t.dataPayloads[servicedef.AfterEvaluation]
	for key, value := range stageData {
		dataBuilder.Set(key, value)
	}
	return data, nil
}

func evaluationSeriesContextToService(
	seriesContext ldhooks.EvaluationSeriesContext,
) servicedef.EvaluationSeriesContext {
	return servicedef.EvaluationSeriesContext{
		FlagKey:      seriesContext.FlagKey(),
		Context:      seriesContext.Context(),
		DefaultValue: seriesContext.DefaultValue(),
		Method:       seriesContext.Method(),
	}
}

func evaluationSeriesDataToService(seriesData ldhooks.EvaluationSeriesData) map[string]ldvalue.Value {
	ret := make(map[string]ldvalue.Value)
	for key, value := range seriesData.AsAnyMap() {
		ret[key] = ldvalue.CopyArbitraryValue(value)
	}
	return ret
}

func detailToService(detail ldreason.EvaluationDetail) *servicedef.EvaluateFlagResponse {
	rep := &servicedef.EvaluateFlagResponse{
		Value:          detail.Value,
		VariationIndex: detail.VariationIndex.AsPointer(),
	}
	if detail.Reason.IsDefined() {
		rep.Reason = &detail.Reason
	}
	return rep
}
