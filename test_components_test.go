package ldclient

import (
	"gopkg.in/launchdarkly/go-sdk-common.v2/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"
	ldevents "gopkg.in/launchdarkly/go-sdk-events.v1"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v1/ldbuilders"
	"gopkg.in/launchdarkly/go-server-sdk-evaluation.v1/ldmodel"
	"gopkg.in/launchdarkly/go-server-sdk.v5/interfaces"
	"gopkg.in/launchdarkly/go-server-sdk.v5/internal"
	"gopkg.in/launchdarkly/go-server-sdk.v5/sharedtest"
)

const testSdkKey = "test-sdk-key"

var testUser = lduser.NewUser("test-user-key")

var alwaysTrueFlag = ldbuilders.NewFlagBuilder("always-true-flag").SingleVariation(ldvalue.Bool(true)).Build()

func basicClientContext() interfaces.ClientContext {
	return newClientContextFromConfig(testSdkKey, Config{Loggers: sharedtest.NewTestLoggers()}, nil)
}

func makeInMemoryDataStore() interfaces.DataStore {
	return internal.NewInMemoryDataStore(sharedtest.NewTestLoggers())
}

func upsertFlag(store interfaces.DataStore, flag *ldmodel.FeatureFlag) {
	store.Upsert(interfaces.DataKindFeatures(), flag.Key, interfaces.StoreItemDescriptor{Version: flag.Version, Item: flag})
}

type singleDataStoreFactory struct {
	dataStore interfaces.DataStore
}

func (f singleDataStoreFactory) CreateDataStore(
	context interfaces.ClientContext,
	dataStoreUpdates interfaces.DataStoreUpdates,
) (interfaces.DataStore, error) {
	return f.dataStore, nil
}

type dataStoreFactoryThatExposesUpdater struct {
	underlyingFactory interfaces.DataStoreFactory
	dataStoreUpdates  interfaces.DataStoreUpdates
}

func (f *dataStoreFactoryThatExposesUpdater) CreateDataStore(
	context interfaces.ClientContext,
	dataStoreUpdates interfaces.DataStoreUpdates,
) (interfaces.DataStore, error) {
	f.dataStoreUpdates = dataStoreUpdates
	return f.underlyingFactory.CreateDataStore(context, dataStoreUpdates)
}

type singlePersistentDataStoreFactory struct {
	persistentDataStore interfaces.PersistentDataStore
}

func (f singlePersistentDataStoreFactory) CreatePersistentDataStore(
	context interfaces.ClientContext,
) (interfaces.PersistentDataStore, error) {
	return f.persistentDataStore, nil
}

type singleDataSourceFactory struct {
	dataSource interfaces.DataSource
}

func (f singleDataSourceFactory) CreateDataSource(
	context interfaces.ClientContext,
	dataSourceUpdates interfaces.DataSourceUpdates,
) (interfaces.DataSource, error) {
	return f.dataSource, nil
}

type dataSourceFactoryThatExposesUpdater struct {
	underlyingFactory interfaces.DataSourceFactory
	dataSourceUpdates interfaces.DataSourceUpdates
}

func (f *dataSourceFactoryThatExposesUpdater) CreateDataSource(
	context interfaces.ClientContext,
	dataSourceUpdates interfaces.DataSourceUpdates,
) (interfaces.DataSource, error) {
	f.dataSourceUpdates = dataSourceUpdates
	return f.underlyingFactory.CreateDataSource(context, dataSourceUpdates)
}

type singleEventProcessorFactory struct {
	eventProcessor ldevents.EventProcessor
}

func (f singleEventProcessorFactory) CreateEventProcessor(context interfaces.ClientContext) (ldevents.EventProcessor, error) {
	return f.eventProcessor, nil
}

type mockDataSource struct {
	Initialized bool
	CloseFn     func() error
	StartFn     func(chan<- struct{})
}

func (u mockDataSource) IsInitialized() bool {
	return u.Initialized
}

func (u mockDataSource) Close() error {
	if u.CloseFn == nil {
		return nil
	}
	return u.CloseFn()
}

func (u mockDataSource) Start(closeWhenReady chan<- struct{}) {
	if u.StartFn == nil {
		return
	}
	u.StartFn(closeWhenReady)
}

func startImmediately(closeWhenReady chan<- struct{}) {
	close(closeWhenReady)
}

type testEventProcessor struct {
	events []ldevents.Event
}

func (t *testEventProcessor) SendEvent(e ldevents.Event) {
	t.events = append(t.events, e)
}

func (t *testEventProcessor) Flush() {}

func (t *testEventProcessor) Close() error {
	return nil
}
