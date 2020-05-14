package sharedtest

import (
	"testing"

	"gopkg.in/launchdarkly/go-server-sdk.v5/interfaces"
)

// This verifies that the PersistentDataStoreTestSuite tests behave as expected as long as the
// PersistentDataStore implementation behaves as expected, so we can distinguish between flaws in the
// implementations and flaws in the test logic.

type mockStoreFactory struct {
	db     *mockDatabaseInstance
	prefix string
}

func (f mockStoreFactory) CreatePersistentDataStore(context interfaces.ClientContext) (interfaces.PersistentDataStore, error) {
	store := newMockPersistentDataStoreWithPrefix(f.db, f.prefix)
	return store, nil
}

func TestPersistentDataStoreTestSuite(t *testing.T) {
	db := newMockDatabaseInstance()

	runTests := func(t *testing.T) {
		NewPersistentDataStoreTestSuite(
			func(prefix string) interfaces.PersistentDataStoreFactory {
				return mockStoreFactory{db, prefix}
			},
			func(prefix string) error {
				db.Clear(prefix)
				return nil
			},
		).ConcurrentModificationHook(
			func(store interfaces.PersistentDataStore, hook func()) {
				store.(*mockPersistentDataStore).testTxHook = hook
			},
		).AlwaysRun(true).Run(t)
	}

	runTests(t)
}
