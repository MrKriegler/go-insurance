package dynamo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Table names
const (
	TableProducts     = "insurance_products"
	TableQuotes       = "insurance_quotes"
	TableApplications = "insurance_applications"
	TableUWCases      = "insurance_underwriting_cases"
	TableOffers       = "insurance_offers"
	TablePolicies     = "insurance_policies"
	TableCounters     = "insurance_counters" // For policy number generation
)

// GSI names
const (
	GSIApplicationsStatus   = "status-index"
	GSIApplicationsQuoteID  = "quote_id-index"
	GSIUWCasesAppID         = "application_id-index"
	GSIUWCasesDecision      = "decision-index"
	GSIOffersAppID          = "application_id-index"
	GSIOffersStatus         = "status-index"
	GSIPoliciesNumber       = "number-index"
	GSIPoliciesAppID        = "application_id-index"
	GSIPoliciesOfferID      = "offer_id-index"
	GSIProductsSlug         = "slug-index"
)

// EnsureTables creates all required tables if they don't exist.
func EnsureTables(ctx context.Context, client *dynamodb.Client, log *slog.Logger) error {
	tables := []struct {
		name   string
		create func(context.Context, *dynamodb.Client) error
	}{
		{TableProducts, createProductsTable},
		{TableQuotes, createQuotesTable},
		{TableApplications, createApplicationsTable},
		{TableUWCases, createUWCasesTable},
		{TableOffers, createOffersTable},
		{TablePolicies, createPoliciesTable},
		{TableCounters, createCountersTable},
	}

	for _, t := range tables {
		exists, err := tableExists(ctx, client, t.name)
		if err != nil {
			return fmt.Errorf("check table %s: %w", t.name, err)
		}
		if exists {
			log.Info("table exists", "table", t.name)
			continue
		}

		log.Info("creating table", "table", t.name)
		if err := t.create(ctx, client); err != nil {
			return fmt.Errorf("create table %s: %w", t.name, err)
		}
		log.Info("table created", "table", t.name)
	}

	return nil
}

func tableExists(ctx context.Context, client *dynamodb.Client, name string) (bool, error) {
	_, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(name),
	})
	if err != nil {
		var notFound *types.ResourceNotFoundException
		if errors.As(err, &notFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func createProductsTable(ctx context.Context, client *dynamodb.Client) error {
	_, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(TableProducts),
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("id"), KeyType: types.KeyTypeHash},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("slug"), AttributeType: types.ScalarAttributeTypeS},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String(GSIProductsSlug),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String("slug"), KeyType: types.KeyTypeHash},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
			},
		},
		BillingMode: types.BillingModePayPerRequest,
	})
	return err
}

func createQuotesTable(ctx context.Context, client *dynamodb.Client) error {
	_, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(TableQuotes),
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("id"), KeyType: types.KeyTypeHash},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
		},
		BillingMode: types.BillingModePayPerRequest,
	})
	return err
}

func createApplicationsTable(ctx context.Context, client *dynamodb.Client) error {
	_, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(TableApplications),
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("id"), KeyType: types.KeyTypeHash},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("status"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("quote_id"), AttributeType: types.ScalarAttributeTypeS},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String(GSIApplicationsStatus),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String("status"), KeyType: types.KeyTypeHash},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
			},
			{
				IndexName: aws.String(GSIApplicationsQuoteID),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String("quote_id"), KeyType: types.KeyTypeHash},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
			},
		},
		BillingMode: types.BillingModePayPerRequest,
	})
	return err
}

func createUWCasesTable(ctx context.Context, client *dynamodb.Client) error {
	_, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(TableUWCases),
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("id"), KeyType: types.KeyTypeHash},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("application_id"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("decision"), AttributeType: types.ScalarAttributeTypeS},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String(GSIUWCasesAppID),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String("application_id"), KeyType: types.KeyTypeHash},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
			},
			{
				IndexName: aws.String(GSIUWCasesDecision),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String("decision"), KeyType: types.KeyTypeHash},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
			},
		},
		BillingMode: types.BillingModePayPerRequest,
	})
	return err
}

func createOffersTable(ctx context.Context, client *dynamodb.Client) error {
	_, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(TableOffers),
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("id"), KeyType: types.KeyTypeHash},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("application_id"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("status"), AttributeType: types.ScalarAttributeTypeS},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String(GSIOffersAppID),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String("application_id"), KeyType: types.KeyTypeHash},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
			},
			{
				IndexName: aws.String(GSIOffersStatus),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String("status"), KeyType: types.KeyTypeHash},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
			},
		},
		BillingMode: types.BillingModePayPerRequest,
	})
	return err
}

func createPoliciesTable(ctx context.Context, client *dynamodb.Client) error {
	_, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(TablePolicies),
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("id"), KeyType: types.KeyTypeHash},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("number"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("application_id"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("offer_id"), AttributeType: types.ScalarAttributeTypeS},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String(GSIPoliciesNumber),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String("number"), KeyType: types.KeyTypeHash},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
			},
			{
				IndexName: aws.String(GSIPoliciesAppID),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String("application_id"), KeyType: types.KeyTypeHash},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
			},
			{
				IndexName: aws.String(GSIPoliciesOfferID),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String("offer_id"), KeyType: types.KeyTypeHash},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
			},
		},
		BillingMode: types.BillingModePayPerRequest,
	})
	return err
}

func createCountersTable(ctx context.Context, client *dynamodb.Client) error {
	_, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(TableCounters),
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("counter_name"), KeyType: types.KeyTypeHash},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("counter_name"), AttributeType: types.ScalarAttributeTypeS},
		},
		BillingMode: types.BillingModePayPerRequest,
	})
	return err
}
