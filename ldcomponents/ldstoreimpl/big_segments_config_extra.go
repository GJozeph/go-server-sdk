package ldstoreimpl

import (
	"time"

	"gopkg.in/launchdarkly/go-server-sdk.v5/interfaces"
)

// BigSegmentsConfigurationProperties encapsulates the SDK's configuration with regard to Big Segments.
//
// This struct implements the BigSegmentsConfiguration interface, but allows for addition of new
// properties. In a future version, BigSegmentsConfigurationBuilder and other configuration builders
// may be changed to use concrete types instead of interfaces.
type BigSegmentsConfigurationProperties struct {
	// Store the data store instance that is used for Big Segments data. If nil, Big Segments are disabled.
	Store interfaces.BigSegmentStore

	// UserCacheSize is the maximum number of users whose Big Segment state will be cached by the SDK
	// at any given time.
	UserCacheSize int

	// UserCacheTime is the maximum length of time that the Big Segment state for a user will be cached
	// by the SDK.
	UserCacheTime time.Duration

	// StatusPollInterval is the interval at which the SDK will poll the Big Segment store to make sure
	// it is available and to determine how long ago it was updated
	StatusPollInterval time.Duration

	// StaleAfter is the maximum length of time between updates of the Big Segments data before the data
	// is considered out of date.
	StaleAfter time.Duration

	// StartPolling is true if the polling task should be started immediately. Otherwise, it will only
	// start after calling BigSegmentsStoreWrapper.SetPollingActive(true). This property is always true
	// in regular use of the SDK; the Relay Proxy may set it to false.
	StartPolling bool
}

func (p BigSegmentsConfigurationProperties) GetStore() interfaces.BigSegmentStore { //nolint:revive
	return p.Store
}

func (p BigSegmentsConfigurationProperties) GetUserCacheSize() int { //nolint:revive
	return p.UserCacheSize
}

func (p BigSegmentsConfigurationProperties) GetUserCacheTime() time.Duration { //nolint:revive
	return p.UserCacheTime
}

func (p BigSegmentsConfigurationProperties) GetStatusPollInterval() time.Duration { //nolint:revive
	return p.StatusPollInterval
}

func (p BigSegmentsConfigurationProperties) GetStaleAfter() time.Duration { //nolint:revive
	return p.StaleAfter
}
