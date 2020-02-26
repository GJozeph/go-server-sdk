package lddynamodb

import (
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldlog"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"
	"gopkg.in/launchdarkly/go-server-sdk.v5/interfaces"
	"gopkg.in/launchdarkly/go-server-sdk.v5/shared_test/ldtest"
	"gopkg.in/launchdarkly/go-server-sdk.v5/utils"
)

const (
	localDynamoEndpoint = "http://localhost:8000"
	testTableName       = "LD_DYNAMODB_TEST_TABLE"
)

func TestDynamoDBDataStoreUncached(t *testing.T) {
	err := createTableIfNecessary()
	require.NoError(t, err)

	ldtest.RunDataStoreTests(t, makeStoreWithCacheTTL(0), clearExistingData, false)
}

func TestDynamoDBDataStoreCached(t *testing.T) {
	err := createTableIfNecessary()
	require.NoError(t, err)

	ldtest.RunDataStoreTests(t, makeStoreWithCacheTTL(30*time.Second), clearExistingData, true)
}

func TestDynamoDBDataStorePrefixes(t *testing.T) {
	ldtest.RunDataStorePrefixIndependenceTests(t,
		func(prefix string) (interfaces.DataStoreFactory, error) {
			return NewDynamoDBDataStoreFactory(testTableName, SessionOptions(makeTestOptions()),
				Prefix(prefix), CacheTTL(0))
		}, clearExistingData)
}

func TestDynamoDBDataStoreConcurrentModification(t *testing.T) {
	var store1Internal *dynamoDBDataStore
	factory1 := func() (interfaces.DataStore, error) {
		opts, _ := validateOptions(testTableName, SessionOptions(makeTestOptions()))
		store1Internal, _ = newDynamoDBDataStoreInternal(opts, ldlog.NewDisabledLoggers())
		return utils.NewNonAtomicDataStoreWrapperWithConfig(store1Internal, ldlog.NewDisabledLoggers()), nil
	}
	factory2 := func() (interfaces.DataStore, error) {
		f, _ := NewDynamoDBDataStoreFactory(testTableName, SessionOptions(makeTestOptions()))
		return f.CreateDataStore(interfaces.NewClientContext("", nil, nil, ldlog.NewDisabledLoggers()))
	}
	ldtest.RunDataStoreConcurrentModificationTests(t, factory1, factory2, func(hook func()) {
		store1Internal.testUpdateHook = hook
	})
}

func TestDynamoDBStoreComponentTypeName(t *testing.T) {
	factory, _ := NewDynamoDBDataStoreFactory("table")
	assert.Equal(t, ldvalue.String("DynamoDB"), (factory.(dynamoDBDataStoreFactory)).DescribeConfiguration())
}

func makeStoreWithCacheTTL(ttl time.Duration) interfaces.DataStoreFactory {
	f, _ := NewDynamoDBDataStoreFactory(testTableName, SessionOptions(makeTestOptions()), CacheTTL(ttl))
	return f
}

func makeTestOptions() session.Options {
	return session.Options{
		Config: aws.Config{
			Credentials: credentials.NewStaticCredentials("dummy", "not", "used"),
			Endpoint:    aws.String(localDynamoEndpoint),
			Region:      aws.String("us-east-1"), // this is ignored for a local instance, but is still required
		},
	}
}

func createTestClient() (*dynamodb.DynamoDB, error) {
	sess, err := session.NewSessionWithOptions(makeTestOptions())
	if err != nil {
		return nil, err
	}
	return dynamodb.New(sess), nil
}

func createTableIfNecessary() error {
	client, err := createTestClient()
	if err != nil {
		return err
	}
	_, err = client.DescribeTable(&dynamodb.DescribeTableInput{TableName: aws.String(testTableName)})
	if err == nil {
		return nil
	}
	if e, ok := err.(awserr.Error); !ok || e.Code() != dynamodb.ErrCodeResourceNotFoundException {
		return err
	}
	createParams := dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			&dynamodb.AttributeDefinition{
				AttributeName: aws.String(tablePartitionKey),
				AttributeType: aws.String(dynamodb.ScalarAttributeTypeS),
			},
			&dynamodb.AttributeDefinition{
				AttributeName: aws.String(tableSortKey),
				AttributeType: aws.String(dynamodb.ScalarAttributeTypeS),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			&dynamodb.KeySchemaElement{
				AttributeName: aws.String(tablePartitionKey),
				KeyType:       aws.String(dynamodb.KeyTypeHash),
			},
			&dynamodb.KeySchemaElement{
				AttributeName: aws.String(tableSortKey),
				KeyType:       aws.String(dynamodb.KeyTypeRange),
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(1),
			WriteCapacityUnits: aws.Int64(1),
		},
		TableName: aws.String(testTableName),
	}
	_, err = client.CreateTable(&createParams)
	if err != nil {
		return err
	}
	// When DynamoDB creates a table, it may not be ready to use immediately
	deadline := time.After(10 * time.Second)
	retry := time.Tick(100 * time.Millisecond)
	for {
		select {
		case <-deadline:
			return fmt.Errorf("Timed out waiting for new table to be ready")
		case <-retry:
			tableInfo, err := client.DescribeTable(&dynamodb.DescribeTableInput{TableName: aws.String(testTableName)})
			if err == nil && *tableInfo.Table.TableStatus == dynamodb.TableStatusActive {
				return nil
			}
		}
	}
}

func clearExistingData() error {
	client, err := createTestClient()
	if err != nil {
		return err
	}
	var items []map[string]*dynamodb.AttributeValue

	err = client.ScanPages(&dynamodb.ScanInput{
		TableName:            aws.String(testTableName),
		ConsistentRead:       aws.Bool(true),
		ProjectionExpression: aws.String("#namespace, #key"),
		ExpressionAttributeNames: map[string]*string{
			"#namespace": aws.String(tablePartitionKey),
			"#key":       aws.String(tableSortKey),
		},
	}, func(out *dynamodb.ScanOutput, lastPage bool) bool {
		items = append(items, out.Items...)
		return !lastPage
	})
	if err != nil {
		return err
	}

	var requests []*dynamodb.WriteRequest
	for _, item := range items {
		requests = append(requests, &dynamodb.WriteRequest{
			DeleteRequest: &dynamodb.DeleteRequest{Key: item},
		})
	}
	return batchWriteRequests(client, testTableName, requests)
}
