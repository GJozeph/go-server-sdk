package ldclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"
)

var flagUser = NewUser("x")
var emptyFeatureStore = NewInMemoryFeatureStore(nil)
var fallthroughValue = ldvalue.String("fall")
var offValue = ldvalue.String("off")
var onValue = ldvalue.String("on")

func intPtr(n int) *int {
	return &n
}

func TestFlagReturnsOffVariationIfFlagIsOff(t *testing.T) {
	f := FeatureFlag{
		Key:          "feature",
		On:           false,
		OffVariation: intPtr(1),
		Fallthrough:  VariationOrRollout{Variation: intPtr(0)},
		Variations:   []ldvalue.Value{fallthroughValue, offValue, onValue},
	}

	result, events := f.EvaluateDetail(flagUser, emptyFeatureStore, false)
	assert.Equal(t, offValue, result.JSONValue)
	assert.Equal(t, intPtr(1), result.VariationIndex)
	assert.Equal(t, evalReasonOffInstance, result.Reason)
	assert.Equal(t, 0, len(events))
}

func TestFlagReturnsNilIfFlagIsOffAndOffVariationIsUnspecified(t *testing.T) {
	f := FeatureFlag{
		Key:         "feature",
		On:          false,
		Fallthrough: VariationOrRollout{Variation: intPtr(0)},
		Variations:  []ldvalue.Value{fallthroughValue, offValue, onValue},
	}

	result, events := f.EvaluateDetail(flagUser, emptyFeatureStore, false)
	assert.Equal(t, ldvalue.Null(), result.JSONValue)
	assert.Nil(t, result.VariationIndex)
	assert.Equal(t, evalReasonOffInstance, result.Reason)
	assert.Equal(t, 0, len(events))
}

func TestFlagReturnsFallthroughIfFlagIsOnAndThereAreNoRules(t *testing.T) {
	f := FeatureFlag{
		Key:         "feature",
		On:          true,
		Rules:       []Rule{},
		Fallthrough: VariationOrRollout{Variation: intPtr(0)},
		Variations:  []ldvalue.Value{fallthroughValue, offValue, onValue},
	}

	result, events := f.EvaluateDetail(flagUser, emptyFeatureStore, false)
	assert.Equal(t, fallthroughValue, result.JSONValue)
	assert.Equal(t, intPtr(0), result.VariationIndex)
	assert.Equal(t, evalReasonFallthroughInstance, result.Reason)
	assert.Equal(t, 0, len(events))
}

func TestFlagReturnsErrorIfFallthroughHasTooHighVariation(t *testing.T) {
	f := FeatureFlag{
		Key:         "feature",
		On:          true,
		Rules:       []Rule{},
		Fallthrough: VariationOrRollout{Variation: intPtr(999)},
		Variations:  []ldvalue.Value{fallthroughValue, offValue, onValue},
	}

	result, events := f.EvaluateDetail(flagUser, emptyFeatureStore, false)
	assert.Equal(t, newEvalErrorResult(EvalErrorMalformedFlag), result)
	assert.Equal(t, 0, len(events))
}

func TestFlagReturnsErrorIfFallthroughHasNegativeVariation(t *testing.T) {
	f := FeatureFlag{
		Key:         "feature",
		On:          true,
		Rules:       []Rule{},
		Fallthrough: VariationOrRollout{Variation: intPtr(-1)},
		Variations:  []ldvalue.Value{fallthroughValue, offValue, onValue},
	}

	result, events := f.EvaluateDetail(flagUser, emptyFeatureStore, false)
	assert.Equal(t, newEvalErrorResult(EvalErrorMalformedFlag), result)
	assert.Equal(t, 0, len(events))
}

func TestFlagReturnsErrorIfFallthroughHasNeitherVariationNorRollout(t *testing.T) {
	f := FeatureFlag{
		Key:         "feature",
		On:          true,
		Rules:       []Rule{},
		Fallthrough: VariationOrRollout{},
		Variations:  []ldvalue.Value{fallthroughValue, offValue, onValue},
	}

	result, events := f.EvaluateDetail(flagUser, emptyFeatureStore, false)
	assert.Equal(t, newEvalErrorResult(EvalErrorMalformedFlag), result)
	assert.Equal(t, 0, len(events))
}

func TestFlagReturnsErrorIfFallthroughHasEmptyRolloutVariationList(t *testing.T) {
	f := FeatureFlag{
		Key:         "feature",
		On:          true,
		Rules:       []Rule{},
		Fallthrough: VariationOrRollout{Rollout: &Rollout{Variations: []WeightedVariation{}}},
		Variations:  []ldvalue.Value{fallthroughValue, offValue, onValue},
	}

	result, events := f.EvaluateDetail(flagUser, emptyFeatureStore, false)
	assert.Equal(t, newEvalErrorResult(EvalErrorMalformedFlag), result)
	assert.Equal(t, 0, len(events))
}

func TestFlagReturnsOffVariationIfPrerequisiteIsNotFound(t *testing.T) {
	f0 := FeatureFlag{
		Key:           "feature0",
		On:            true,
		OffVariation:  intPtr(1),
		Prerequisites: []Prerequisite{Prerequisite{"feature1", 1}},
		Fallthrough:   VariationOrRollout{Variation: intPtr(0)},
		Variations:    []ldvalue.Value{fallthroughValue, offValue, onValue},
	}

	result, events := f0.EvaluateDetail(flagUser, emptyFeatureStore, false)
	assert.Equal(t, offValue, result.JSONValue)
	assert.Equal(t, intPtr(1), result.VariationIndex)
	assert.Equal(t, newEvalReasonPrerequisiteFailed("feature1"), result.Reason)
	assert.Equal(t, 0, len(events))
}

func TestFlagReturnsOffVariationAndEventIfPrerequisiteIsOff(t *testing.T) {
	f0 := FeatureFlag{
		Key:           "feature0",
		On:            true,
		OffVariation:  intPtr(1),
		Prerequisites: []Prerequisite{Prerequisite{"feature1", 1}},
		Fallthrough:   VariationOrRollout{Variation: intPtr(0)},
		Variations:    []ldvalue.Value{fallthroughValue, offValue, onValue},
		Version:       1,
	}
	f1 := FeatureFlag{
		Key:          "feature1",
		On:           false,
		OffVariation: intPtr(1),
		// note that even though it returns the desired variation, it is still off and therefore not a match
		Fallthrough: VariationOrRollout{Variation: intPtr(0)},
		Variations:  []ldvalue.Value{ldvalue.String("nogo"), ldvalue.String("go")},
		Version:     2,
	}
	featureStore := NewInMemoryFeatureStore(nil)
	featureStore.Upsert(Features, &f1)

	result, events := f0.EvaluateDetail(flagUser, featureStore, false)
	assert.Equal(t, offValue, result.JSONValue)
	assert.Equal(t, intPtr(1), result.VariationIndex)
	assert.Equal(t, newEvalReasonPrerequisiteFailed("feature1"), result.Reason)

	assert.Equal(t, 1, len(events))
	e := events[0]
	assert.Equal(t, f1.Key, e.Key)
	assert.Equal(t, ldvalue.String("go"), e.Value)
	assert.Equal(t, intPtr(f1.Version), e.Version)
	assert.Equal(t, intPtr(1), e.Variation)
	assert.Equal(t, strPtr(f0.Key), e.PrereqOf)
}

func TestFlagReturnsOffVariationAndEventIfPrerequisiteIsNotMet(t *testing.T) {
	f0 := FeatureFlag{
		Key:           "feature0",
		On:            true,
		OffVariation:  intPtr(1),
		Prerequisites: []Prerequisite{Prerequisite{"feature1", 1}},
		Fallthrough:   VariationOrRollout{Variation: intPtr(0)},
		Variations:    []ldvalue.Value{fallthroughValue, offValue, onValue},
		Version:       1,
	}
	f1 := FeatureFlag{
		Key:          "feature1",
		On:           true,
		OffVariation: intPtr(1),
		Fallthrough:  VariationOrRollout{Variation: intPtr(0)},
		Variations:   []ldvalue.Value{ldvalue.String("nogo"), ldvalue.String("go")},
		Version:      2,
	}
	featureStore := NewInMemoryFeatureStore(nil)
	featureStore.Upsert(Features, &f1)

	result, events := f0.EvaluateDetail(flagUser, featureStore, false)
	assert.Equal(t, offValue, result.JSONValue)
	assert.Equal(t, intPtr(1), result.VariationIndex)
	assert.Equal(t, newEvalReasonPrerequisiteFailed("feature1"), result.Reason)

	assert.Equal(t, 1, len(events))
	e := events[0]
	assert.Equal(t, f1.Key, e.Key)
	assert.Equal(t, ldvalue.String("nogo"), e.Value)
	assert.Equal(t, intPtr(f1.Version), e.Version)
	assert.Equal(t, intPtr(0), e.Variation)
	assert.Equal(t, strPtr(f0.Key), e.PrereqOf)
}

func TestFlagReturnsFallthroughVariationAndEventIfPrerequisiteIsMetAndThereAreNoRules(t *testing.T) {
	f0 := FeatureFlag{
		Key:           "feature0",
		On:            true,
		OffVariation:  intPtr(1),
		Prerequisites: []Prerequisite{Prerequisite{"feature1", 1}},
		Fallthrough:   VariationOrRollout{Variation: intPtr(0)},
		Variations:    []ldvalue.Value{fallthroughValue, offValue, onValue},
		Version:       1,
	}
	f1 := FeatureFlag{
		Key:          "feature1",
		On:           true,
		OffVariation: intPtr(1),
		Fallthrough:  VariationOrRollout{Variation: intPtr(1)}, // this 1 matches the 1 in the prerequisites array
		Variations:   []ldvalue.Value{ldvalue.String("nogo"), ldvalue.String("go")},
		Version:      2,
	}
	featureStore := NewInMemoryFeatureStore(nil)
	featureStore.Upsert(Features, &f1)

	result, events := f0.EvaluateDetail(flagUser, featureStore, false)
	assert.Equal(t, fallthroughValue, result.JSONValue)
	assert.Equal(t, intPtr(0), result.VariationIndex)
	assert.Equal(t, evalReasonFallthroughInstance, result.Reason)

	assert.Equal(t, 1, len(events))
	e := events[0]
	assert.Equal(t, f1.Key, e.Key)
	assert.Equal(t, ldvalue.String("go"), e.Value)
	assert.Equal(t, intPtr(1), e.Variation)
	assert.Equal(t, intPtr(f1.Version), e.Version)
	assert.Equal(t, strPtr(f0.Key), e.PrereqOf)
}

func TestPrerequisiteCanMatchWithNonScalarValue(t *testing.T) {
	f0 := FeatureFlag{
		Key:           "feature0",
		On:            true,
		OffVariation:  intPtr(1),
		Prerequisites: []Prerequisite{Prerequisite{"feature1", 1}},
		Fallthrough:   VariationOrRollout{Variation: intPtr(0)},
		Variations:    []ldvalue.Value{fallthroughValue, offValue, onValue},
		Version:       1,
	}
	f1 := FeatureFlag{
		Key:          "feature1",
		On:           true,
		OffVariation: intPtr(1),
		Fallthrough:  VariationOrRollout{Variation: intPtr(1)}, // this 1 matches the 1 in the prerequisites array
		Variations:   []ldvalue.Value{ldvalue.ArrayOf(ldvalue.String("000")), ldvalue.ArrayOf(ldvalue.String("001"))},
		Version:      2,
	}
	featureStore := NewInMemoryFeatureStore(nil)
	featureStore.Upsert(Features, &f1)

	result, events := f0.EvaluateDetail(flagUser, featureStore, false)
	assert.Equal(t, fallthroughValue, result.JSONValue)
	assert.Equal(t, intPtr(0), result.VariationIndex)
	assert.Equal(t, evalReasonFallthroughInstance, result.Reason)

	assert.Equal(t, 1, len(events))
	e := events[0]
	assert.Equal(t, f1.Key, e.Key)
	assert.Equal(t, ldvalue.ArrayOf(ldvalue.String("001")), e.Value)
	assert.Equal(t, intPtr(1), e.Variation)
	assert.Equal(t, intPtr(f1.Version), e.Version)
	assert.Equal(t, strPtr(f0.Key), e.PrereqOf)
}

func TestMultipleLevelsOfPrerequisiteProduceMultipleEvents(t *testing.T) {
	f0 := FeatureFlag{
		Key:           "feature0",
		On:            true,
		OffVariation:  intPtr(1),
		Prerequisites: []Prerequisite{Prerequisite{"feature1", 1}},
		Fallthrough:   VariationOrRollout{Variation: intPtr(0)},
		Variations:    []ldvalue.Value{fallthroughValue, offValue, onValue},
		Version:       1,
	}
	f1 := FeatureFlag{
		Key:           "feature1",
		On:            true,
		OffVariation:  intPtr(1),
		Prerequisites: []Prerequisite{Prerequisite{"feature2", 1}},
		Fallthrough:   VariationOrRollout{Variation: intPtr(1)}, // this 1 matches the 1 in the prerequisites array
		Variations:    []ldvalue.Value{ldvalue.String("nogo"), ldvalue.String("go")},
		Version:       2,
	}
	f2 := FeatureFlag{
		Key:         "feature2",
		On:          true,
		Fallthrough: VariationOrRollout{Variation: intPtr(1)},
		Variations:  []ldvalue.Value{ldvalue.String("nogo"), ldvalue.String("go")},
		Version:     3,
	}
	featureStore := NewInMemoryFeatureStore(nil)
	featureStore.Upsert(Features, &f1)
	featureStore.Upsert(Features, &f2)

	result, events := f0.EvaluateDetail(flagUser, featureStore, false)
	assert.Equal(t, fallthroughValue, result.JSONValue)
	assert.Equal(t, intPtr(0), result.VariationIndex)
	assert.Equal(t, evalReasonFallthroughInstance, result.Reason)

	assert.Equal(t, 2, len(events))
	// events are generated recursively, so the deepest level of prerequisite appears first

	e0 := events[0]
	assert.Equal(t, f2.Key, e0.Key)
	assert.Equal(t, ldvalue.String("go"), e0.Value)
	assert.Equal(t, intPtr(1), e0.Variation)
	assert.Equal(t, intPtr(f2.Version), e0.Version)
	assert.Equal(t, strPtr(f1.Key), e0.PrereqOf)

	e1 := events[1]
	assert.Equal(t, f1.Key, e1.Key)
	assert.Equal(t, ldvalue.String("go"), e1.Value)
	assert.Equal(t, intPtr(1), e1.Variation)
	assert.Equal(t, intPtr(f1.Version), e1.Version)
	assert.Equal(t, strPtr(f0.Key), e1.PrereqOf)
}

func TestFlagMatchesUserFromTargets(t *testing.T) {
	f := FeatureFlag{
		Key:          "feature",
		On:           true,
		OffVariation: intPtr(1),
		Targets:      []Target{Target{[]string{"whoever", "userkey"}, 2}},
		Fallthrough:  VariationOrRollout{Variation: intPtr(0)},
		Variations:   []ldvalue.Value{fallthroughValue, offValue, onValue},
	}
	user := NewUser("userkey")

	result, events := f.EvaluateDetail(user, emptyFeatureStore, false)
	assert.Equal(t, onValue, result.JSONValue)
	assert.Equal(t, intPtr(2), result.VariationIndex)
	assert.Equal(t, evalReasonTargetMatchInstance, result.Reason)
	assert.Equal(t, 0, len(events))
}

func TestFlagMatchesUserFromRules(t *testing.T) {
	user := NewUser("userkey")
	f := makeFlagToMatchUser(user, VariationOrRollout{Variation: intPtr(2)})

	result, events := f.EvaluateDetail(user, emptyFeatureStore, false)
	assert.Equal(t, onValue, result.JSONValue)
	assert.Equal(t, intPtr(2), result.VariationIndex)
	assert.Equal(t, newEvalReasonRuleMatch(0, "rule-id"), result.Reason)
	assert.Equal(t, 0, len(events))
}

func TestRuleWithTooHighVariationIndexReturnsMalformedFlagError(t *testing.T) {
	user := NewUser("userkey")
	f := makeFlagToMatchUser(user, VariationOrRollout{Variation: intPtr(999)})

	result, events := f.EvaluateDetail(user, emptyFeatureStore, false)
	assert.Equal(t, newEvalErrorResult(EvalErrorMalformedFlag), result)
	assert.Equal(t, 0, len(events))
}

func TestRuleWithNegativeVariationIndexReturnsMalformedFlagError(t *testing.T) {
	user := NewUser("userkey")
	f := makeFlagToMatchUser(user, VariationOrRollout{Variation: intPtr(-1)})

	result, events := f.EvaluateDetail(user, emptyFeatureStore, false)
	assert.Equal(t, newEvalErrorResult(EvalErrorMalformedFlag), result)
	assert.Equal(t, 0, len(events))
}

func TestRuleWithNoVariationOrRolloutReturnsMalformedFlagError(t *testing.T) {
	user := NewUser("userkey")
	f := makeFlagToMatchUser(user, VariationOrRollout{})

	result, events := f.EvaluateDetail(user, emptyFeatureStore, false)
	assert.Equal(t, newEvalErrorResult(EvalErrorMalformedFlag), result)
	assert.Equal(t, 0, len(events))
}

func TestRuleWithRolloutWithEmptyVariationsListReturnsMalformedFlagError(t *testing.T) {
	user := NewUser("userkey")
	f := makeFlagToMatchUser(user, VariationOrRollout{Rollout: &Rollout{Variations: []WeightedVariation{}}})

	result, events := f.EvaluateDetail(user, emptyFeatureStore, false)
	assert.Equal(t, newEvalErrorResult(EvalErrorMalformedFlag), result)
	assert.Equal(t, 0, len(events))
}

func TestClauseCanMatchBuiltInAttribute(t *testing.T) {
	clause := Clause{
		Attribute: "name",
		Op:        "in",
		Values:    []ldvalue.Value{ldvalue.String("Bob")},
	}
	f := booleanFlagWithClause(clause)
	user := NewUserBuilder("key").Name("Bob").Build()

	result, _ := f.EvaluateDetail(user, emptyFeatureStore, false)
	assert.Equal(t, ldvalue.Bool(true), result.JSONValue)
}

func TestClauseCanMatchCustomAttribute(t *testing.T) {
	clause := Clause{
		Attribute: "legs",
		Op:        "in",
		Values:    []ldvalue.Value{ldvalue.Int(4)},
	}
	f := booleanFlagWithClause(clause)
	user := NewUserBuilder("key").Custom("legs", ldvalue.Int(4)).Build()

	result, _ := f.EvaluateDetail(user, emptyFeatureStore, false)
	assert.Equal(t, ldvalue.Bool(true), result.JSONValue)
}

func TestClauseReturnsFalseForMissingAttribute(t *testing.T) {
	clause := Clause{
		Attribute: "legs",
		Op:        "in",
		Values:    []ldvalue.Value{ldvalue.Int(4)},
	}
	f := booleanFlagWithClause(clause)
	user := NewUserBuilder("key").Name("Bob").Build()

	result, _ := f.EvaluateDetail(user, emptyFeatureStore, false)
	assert.Equal(t, ldvalue.Bool(false), result.JSONValue)
}

func TestClauseCanBeNegated(t *testing.T) {
	clause := Clause{
		Attribute: "name",
		Op:        "in",
		Values:    []ldvalue.Value{ldvalue.String("Bob")},
		Negate:    true,
	}
	f := booleanFlagWithClause(clause)
	user := NewUserBuilder("key").Name("Bob").Build()

	result, _ := f.EvaluateDetail(user, emptyFeatureStore, false)
	assert.Equal(t, ldvalue.Bool(false), result.JSONValue)
}

func TestClauseForMissingAttributeIsFalseEvenIfNegated(t *testing.T) {
	clause := Clause{
		Attribute: "legs",
		Op:        "in",
		Values:    []ldvalue.Value{ldvalue.Int(4)},
		Negate:    true,
	}
	f := booleanFlagWithClause(clause)
	user := NewUserBuilder("key").Name("Bob").Build()

	result, _ := f.EvaluateDetail(user, emptyFeatureStore, false)
	assert.Equal(t, ldvalue.Bool(false), result.JSONValue)
}

func TestClauseWithUnknownOperatorDoesNotMatch(t *testing.T) {
	clause := Clause{
		Attribute: "name",
		Op:        "doesSomethingUnsupported",
		Values:    []ldvalue.Value{ldvalue.String("Bob")},
	}
	f := booleanFlagWithClause(clause)
	user := NewUserBuilder("key").Name("Bob").Build()

	result, _ := f.EvaluateDetail(user, emptyFeatureStore, false)
	assert.Equal(t, ldvalue.Bool(false), result.JSONValue)
}

func TestClauseWithUnknownOperatorDoesNotStopSubsequentRuleFromMatching(t *testing.T) {
	badClause := Clause{
		Attribute: "name",
		Op:        "doesSomethingUnsupported",
		Values:    []ldvalue.Value{ldvalue.String("Bob")},
	}
	badRule := Rule{ID: "bad", Clauses: []Clause{badClause}, VariationOrRollout: VariationOrRollout{Variation: intPtr(1)}}
	goodClause := Clause{
		Attribute: "name",
		Op:        "in",
		Values:    []ldvalue.Value{ldvalue.String("Bob")},
	}
	goodRule := Rule{ID: "good", Clauses: []Clause{goodClause}, VariationOrRollout: VariationOrRollout{Variation: intPtr(1)}}
	f := FeatureFlag{
		Key:         "feature",
		On:          true,
		Rules:       []Rule{badRule, goodRule},
		Fallthrough: VariationOrRollout{Variation: intPtr(0)},
		Variations:  []ldvalue.Value{ldvalue.Bool(false), ldvalue.Bool(true)},
	}
	user := NewUserBuilder("key").Name("Bob").Build()

	result, _ := f.EvaluateDetail(user, emptyFeatureStore, false)
	assert.Equal(t, ldvalue.Bool(true), result.JSONValue)
	assert.Equal(t, newEvalReasonRuleMatch(1, "good"), result.Reason)
}

func TestSegmentMatchClauseRetrievesSegmentFromStore(t *testing.T) {
	segment := Segment{
		Key:      "segkey",
		Included: []string{"foo"},
	}
	clause := Clause{Attribute: "", Op: "segmentMatch", Values: []ldvalue.Value{ldvalue.String("segkey")}}
	f := booleanFlagWithClause(clause)
	featureStore := NewInMemoryFeatureStore(nil)
	featureStore.Upsert(Segments, &segment)
	user := NewUser("foo")

	result, _ := f.EvaluateDetail(user, featureStore, false)
	assert.Equal(t, ldvalue.Bool(true), result.JSONValue)
}

func TestSegmentMatchClauseFallsThroughIfSegmentNotFound(t *testing.T) {
	clause := Clause{Attribute: "", Op: "segmentMatch", Values: []ldvalue.Value{ldvalue.String("segkey")}}
	f := booleanFlagWithClause(clause)
	user := NewUser("foo")

	result, _ := f.EvaluateDetail(user, emptyFeatureStore, false)
	assert.Equal(t, ldvalue.Bool(false), result.JSONValue)
}

func TestCanMatchJustOneSegmentFromList(t *testing.T) {
	segment := Segment{
		Key:      "segkey",
		Included: []string{"foo"},
	}
	clause := Clause{
		Attribute: "",
		Op:        "segmentMatch",
		Values:    []ldvalue.Value{ldvalue.String("unknownsegkey"), ldvalue.String("segkey")},
	}
	f := booleanFlagWithClause(clause)
	featureStore := NewInMemoryFeatureStore(nil)
	featureStore.Upsert(Segments, &segment)
	user := NewUser("foo")

	result, _ := f.EvaluateDetail(user, featureStore, false)
	assert.Equal(t, ldvalue.Bool(true), result.JSONValue)
}

func TestVariationIndexForUser(t *testing.T) {
	wv1 := WeightedVariation{Variation: 0, Weight: 60000.0}
	wv2 := WeightedVariation{Variation: 1, Weight: 40000.0}
	rollout := Rollout{Variations: []WeightedVariation{wv1, wv2}}
	rule := Rule{VariationOrRollout: VariationOrRollout{Rollout: &rollout}}

	variationIndex := rule.variationIndexForUser(NewUser("userKeyA"), "hashKey", "saltyA")
	assert.NotNil(t, variationIndex)
	assert.Equal(t, 0, *variationIndex)

	variationIndex = rule.variationIndexForUser(NewUser("userKeyB"), "hashKey", "saltyA")
	assert.NotNil(t, variationIndex)
	assert.Equal(t, 1, *variationIndex)

	variationIndex = rule.variationIndexForUser(NewUser("userKeyC"), "hashKey", "saltyA")
	assert.NotNil(t, variationIndex)
	assert.Equal(t, 0, *variationIndex)
}

func TestBucketUserByKey(t *testing.T) {
	user := NewUser("userKeyA")
	bucket := bucketUser(user, "hashKey", "key", "saltyA")
	assert.InEpsilon(t, 0.42157587, bucket, 0.0000001)

	user = NewUser("userKeyB")
	bucket = bucketUser(user, "hashKey", "key", "saltyA")
	assert.InEpsilon(t, 0.6708485, bucket, 0.0000001)

	user = NewUser("userKeyC")
	bucket = bucketUser(user, "hashKey", "key", "saltyA")
	assert.InEpsilon(t, 0.10343106, bucket, 0.0000001)
}

func TestBucketUserByIntAttr(t *testing.T) {
	user := NewUserBuilder("userKeyD").Custom("intAttr", ldvalue.Int(33333)).Build()
	bucket := bucketUser(user, "hashKey", "intAttr", "saltyA")
	assert.InEpsilon(t, 0.54771423, bucket, 0.0000001)

	user = NewUserBuilder("userKeyD").Custom("stringAttr", ldvalue.String("33333")).Build()
	bucket2 := bucketUser(user, "hashKey", "stringAttr", "saltyA")
	assert.InEpsilon(t, bucket, bucket2, 0.0000001)
}

func TestBucketUserByFloatAttrNotAllowed(t *testing.T) {
	user := NewUserBuilder("userKeyE").Custom("floatAttr", ldvalue.Float64(999.999)).Build()
	bucket := bucketUser(user, "hashKey", "floatAttr", "saltyA")
	assert.InDelta(t, 0.0, bucket, 0.0000001)
}

func TestBucketUserByFloatAttrThatIsReallyAnIntIsAllowed(t *testing.T) {
	user := NewUserBuilder("userKeyE").Custom("floatAttr", ldvalue.Float64(33333)).Build()
	bucket := bucketUser(user, "hashKey", "floatAttr", "saltyA")
	assert.InEpsilon(t, 0.54771423, bucket, 0.0000001)
}

func booleanFlagWithClause(clause Clause) FeatureFlag {
	return FeatureFlag{
		Key: "feature",
		On:  true,
		Rules: []Rule{
			Rule{Clauses: []Clause{clause}, VariationOrRollout: VariationOrRollout{Variation: intPtr(1)}},
		},
		Fallthrough: VariationOrRollout{Variation: intPtr(0)},
		Variations:  []ldvalue.Value{ldvalue.Bool(false), ldvalue.Bool(true)},
	}
}

func newEvalErrorResult(kind EvalErrorKind) EvaluationDetail {
	return EvaluationDetail{Reason: newEvalReasonError(kind)}
}

func makeClauseToMatchUser(user User) Clause {
	return Clause{
		Attribute: "key",
		Op:        "in",
		Values:    []ldvalue.Value{ldvalue.String(user.GetKey())},
	}
}

func makeClauseToNotMatchUser(user User) Clause {
	return Clause{
		Attribute: "key",
		Op:        "in",
		Values:    []ldvalue.Value{ldvalue.String("not-" + user.GetKey())},
	}
}

func makeFlagToMatchUser(user User, variationOrRollout VariationOrRollout) FeatureFlag {
	return FeatureFlag{
		Key:          "feature",
		On:           true,
		OffVariation: intPtr(1),
		Rules: []Rule{
			Rule{
				ID:                 "rule-id",
				Clauses:            []Clause{makeClauseToMatchUser(user)},
				VariationOrRollout: variationOrRollout,
			},
		},
		Fallthrough: VariationOrRollout{Variation: intPtr(0)},
		Variations:  []ldvalue.Value{fallthroughValue, offValue, onValue},
	}
}
