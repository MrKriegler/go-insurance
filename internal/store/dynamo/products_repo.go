package dynamo

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/MrKriegler/go-insurance/internal/core"
	"github.com/MrKriegler/go-insurance/internal/platform/ids"
)

type ProductItem struct {
	ID          string  `dynamodbav:"id"`
	Slug        string  `dynamodbav:"slug"`
	Name        string  `dynamodbav:"name"`
	TermYears   int     `dynamodbav:"term_years"`
	MinCoverage int64   `dynamodbav:"min_coverage"`
	MaxCoverage int64   `dynamodbav:"max_coverage"`
	BaseRate    float64 `dynamodbav:"base_rate"`
}

func (i ProductItem) ToCore() core.Product {
	return core.Product{
		ID:          i.ID,
		Slug:        i.Slug,
		Name:        i.Name,
		TermYears:   i.TermYears,
		MinCoverage: i.MinCoverage,
		MaxCoverage: i.MaxCoverage,
		BaseRate:    i.BaseRate,
	}
}

func productItemFromCore(p core.Product) ProductItem {
	return ProductItem{
		ID:          p.ID,
		Slug:        p.Slug,
		Name:        p.Name,
		TermYears:   p.TermYears,
		MinCoverage: p.MinCoverage,
		MaxCoverage: p.MaxCoverage,
		BaseRate:    p.BaseRate,
	}
}

type ProductRepo struct {
	client *dynamodb.Client
}

func NewProductRepo(client *dynamodb.Client) *ProductRepo {
	return &ProductRepo{client: client}
}

func (r *ProductRepo) List(ctx context.Context) ([]core.Product, error) {
	out, err := r.client.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(TableProducts),
	})
	if err != nil {
		return nil, fmt.Errorf("products.scan: %w", err)
	}

	var items []ProductItem
	if err := attributevalue.UnmarshalListOfMaps(out.Items, &items); err != nil {
		return nil, fmt.Errorf("products.unmarshal: %w", err)
	}

	products := make([]core.Product, len(items))
	for i, item := range items {
		products[i] = item.ToCore()
	}
	return products, nil
}

func (r *ProductRepo) GetBySlug(ctx context.Context, slug string) (core.Product, error) {
	out, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(TableProducts),
		IndexName:              aws.String(GSIProductsSlug),
		KeyConditionExpression: aws.String("slug = :slug"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":slug": &types.AttributeValueMemberS{Value: slug},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return core.Product{}, fmt.Errorf("products.queryBySlug: %w", err)
	}

	if len(out.Items) == 0 {
		return core.Product{}, core.ErrNotFound
	}

	var item ProductItem
	if err := attributevalue.UnmarshalMap(out.Items[0], &item); err != nil {
		return core.Product{}, fmt.Errorf("products.unmarshal: %w", err)
	}

	return item.ToCore(), nil
}

func (r *ProductRepo) GetByID(ctx context.Context, id string) (core.Product, error) {
	out, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(TableProducts),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return core.Product{}, fmt.Errorf("products.getItem: %w", err)
	}

	if out.Item == nil {
		return core.Product{}, core.ErrNotFound
	}

	var item ProductItem
	if err := attributevalue.UnmarshalMap(out.Item, &item); err != nil {
		return core.Product{}, fmt.Errorf("products.unmarshal: %w", err)
	}

	return item.ToCore(), nil
}

func (r *ProductRepo) UpsertBySlug(ctx context.Context, p core.Product) error {
	// First check if product with this slug exists
	existing, err := r.GetBySlug(ctx, p.Slug)
	if err != nil && err != core.ErrNotFound {
		return err
	}

	item := productItemFromCore(p)
	if err == nil {
		// Update existing - keep the same ID
		item.ID = existing.ID
	} else {
		// New product - generate ID
		item.ID = ids.New()
	}

	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("products.marshal: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(TableProducts),
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("products.putItem: %w", err)
	}

	return nil
}

func (r *ProductRepo) Create(ctx context.Context, p core.Product) error {
	item := productItemFromCore(p)
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("products.marshal: %w", err)
	}

	// Use condition to prevent overwriting
	cond := expression.AttributeNotExists(expression.Name("id"))
	expr, err := expression.NewBuilder().WithCondition(cond).Build()
	if err != nil {
		return fmt.Errorf("products.buildExpr: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:                 aws.String(TableProducts),
		Item:                      av,
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		var ccf *types.ConditionalCheckFailedException
		if ok := isConditionCheckFailed(err, &ccf); ok {
			return core.ErrConflict
		}
		return fmt.Errorf("products.putItem: %w", err)
	}

	return nil
}

func isConditionCheckFailed(err error, target **types.ConditionalCheckFailedException) bool {
	if err == nil {
		return false
	}
	var ccf *types.ConditionalCheckFailedException
	if ok := errorAs(err, &ccf); ok {
		*target = ccf
		return true
	}
	return false
}

func errorAs[T error](err error, target *T) bool {
	for err != nil {
		if e, ok := err.(T); ok {
			*target = e
			return true
		}
		if u, ok := err.(interface{ Unwrap() error }); ok {
			err = u.Unwrap()
		} else {
			return false
		}
	}
	return false
}
