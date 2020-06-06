package interfaces

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v1/ldbuilders"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v1/ldmodel"
)

func TestDataKinds(t *testing.T) {
	assert.Equal(t, []StoreDataKind{DataKindFeatures(), DataKindSegments()}, StoreDataKinds())
}

func TestDataKindFeatures(t *testing.T) {
	kind := DataKindFeatures()

	t.Run("name", func(t *testing.T) {
		assert.Equal(t, "features", kind.GetName())
		assert.Equal(t, "features", fmt.Sprintf("%s", kind))
	})

	t.Run("serialize", func(t *testing.T) {
		flag := ldbuilders.NewFlagBuilder("flagkey").Version(2).Build()
		bytes := kind.Serialize(StoreItemDescriptor{Version: flag.Version, Item: &flag})
		assert.Contains(t, string(bytes), `"key":"flagkey"`)
		assert.Contains(t, string(bytes), `"version":2`)
	})

	t.Run("deserialize", func(t *testing.T) {
		json := `{"key":"flagkey","version":2}`
		item, err := kind.Deserialize([]byte(json))
		assert.NoError(t, err)
		assert.Equal(t, 2, item.Version)
		require.NotNil(t, item.Item)
		flag := item.Item.(*ldmodel.FeatureFlag)
		assert.Equal(t, "flagkey", flag.Key)
		assert.Equal(t, 2, flag.Version)
	})

	t.Run("deserialize deleted item", func(t *testing.T) {
		json := `{"key":"flagkey","version":2,"deleted":true}`
		item, err := kind.Deserialize([]byte(json))
		assert.NoError(t, err)
		assert.Equal(t, 2, item.Version)
		require.Nil(t, item.Item)
	})

	t.Run("will not serialize wrong type", func(t *testing.T) {
		bytes := kind.Serialize(StoreItemDescriptor{Version: 1, Item: "not a flag"})
		assert.Nil(t, bytes)
	})

	t.Run("deserialization error", func(t *testing.T) {
		json := `{"key":"flagkey"`
		item, err := kind.Deserialize([]byte(json))
		assert.Error(t, err)
		require.Nil(t, item.Item)
	})
}

func TestDataKindSegments(t *testing.T) {
	kind := DataKindSegments()

	t.Run("name", func(t *testing.T) {
		assert.Equal(t, "segments", kind.GetName())
		assert.Equal(t, "segments", fmt.Sprintf("%s", kind))
	})

	t.Run("serialize", func(t *testing.T) {
		segment := ldbuilders.NewSegmentBuilder("segmentkey").Version(2).Build()
		bytes := kind.Serialize(StoreItemDescriptor{Version: segment.Version, Item: &segment})
		assert.Contains(t, string(bytes), `"key":"segmentkey"`)
		assert.Contains(t, string(bytes), `"version":2`)
	})

	t.Run("deserialize", func(t *testing.T) {
		json := `{"key":"segmentkey","version":2}`
		item, err := kind.Deserialize([]byte(json))
		assert.NoError(t, err)
		require.NotNil(t, item.Item)
		segment := item.Item.(*ldmodel.Segment)
		assert.Equal(t, "segmentkey", segment.Key)
		assert.Equal(t, 2, segment.Version)
	})

	t.Run("deserialize deleted item", func(t *testing.T) {
		json := `{"key":"segmentkey","version":2,"deleted":true}`
		item, err := kind.Deserialize([]byte(json))
		assert.NoError(t, err)
		assert.Equal(t, 2, item.Version)
		require.Nil(t, item.Item)
	})

	t.Run("will not serialize wrong type", func(t *testing.T) {
		bytes := kind.Serialize(StoreItemDescriptor{Version: 1, Item: "not a flag"})
		assert.Nil(t, bytes)
	})

	t.Run("deserialization error", func(t *testing.T) {
		json := `{"key":"segmentkey"`
		item, err := kind.Deserialize([]byte(json))
		assert.Error(t, err)
		require.Nil(t, item.Item)
	})
}
