package ldcomponents

import (
	"github.com/launchdarkly/go-server-sdk/v6/internal"
	"github.com/launchdarkly/go-server-sdk/v6/internal/sharedtest"
	"github.com/launchdarkly/go-server-sdk/v6/subsystems"
)

const testSdkKey = "test-sdk-key"

func basicClientContext() subsystems.ClientContext {
	return sharedtest.NewSimpleTestContext(testSdkKey)
}

// Returns a basic context where all of the service endpoints point to the specified URI.
func makeTestContextWithBaseURIs(uri string) *internal.ClientContextImpl {
	return internal.NewClientContextImpl(
		subsystems.BasicConfiguration{SDKKey: testSdkKey, ServiceEndpoints: RelayProxyEndpoints(uri)},
		subsystems.HTTPConfiguration{},
		sharedtest.TestLoggingConfig(),
	)
}
