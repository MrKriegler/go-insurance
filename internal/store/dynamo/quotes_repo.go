package dynamo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/MrKriegler/go-insurance/internal/core"
)

type QuoteItem struct {
	ID             string  `dynamodbav:"id"`
	ProductID      string  `dynamodbav:"product_id"`
	ProductSlug    string  `dynamodbav:"product_slug"`
	CoverageAmount int64   `dynamodbav:"coverage_amount"`
	TermYears      int     `dynamodbav:"term_years"`
	MonthlyPremium float64 `dynamodbav:"monthly_premium"`
	Status         string  `dynamodbav:"status"`
	CreatedAt      string  `dynamodbav:"created_at"`
	ExpiresAt      string  `dynamodbav:"expires_at"`
}

func (i QuoteItem) ToCore() core.Quote {
	createdAt, _ := time.Parse(time.RFC3339, i.CreatedAt)
	expiresAt, _ := time.Parse(time.RFC3339, i.ExpiresAt)
	return core.Quote{
		ID:             i.ID,
		ProductID:      i.ProductID,
		ProductSlug:    i.ProductSlug,
		CoverageAmount: i.CoverageAmount,
		TermYears:      i.TermYears,
		MonthlyPremium: i.MonthlyPremium,
		Status:         core.QuoteStatus(i.Status),
		CreatedAt:      createdAt,
		ExpiresAt:      expiresAt,
	}
}

func quoteItemFromCore(q core.Quote) QuoteItem {
	return QuoteItem{
		ID:             q.ID,
		ProductID:      q.ProductID,
		ProductSlug:    q.ProductSlug,
		CoverageAmount: q.CoverageAmount,
		TermYears:      q.TermYears,
		MonthlyPremium: q.MonthlyPremium,
		Status:         string(q.Status),
		CreatedAt:      q.CreatedAt.Format(time.RFC3339),
		ExpiresAt:      q.ExpiresAt.Format(time.RFC3339),
	}
}

type QuoteRepo struct {
	client *dynamodb.Client
}

func NewQuoteRepo(client *dynamodb.Client) *QuoteRepo {
	return &QuoteRepo{client: client}
}

func (r *QuoteRepo) Create(ctx context.Context, q core.Quote) error {
	item := quoteItemFromCore(q)
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("quotes.marshal: %w", err)
	}

	cond := expression.AttributeNotExists(expression.Name("id"))
	expr, err := expression.NewBuilder().WithCondition(cond).Build()
	if err != nil {
		return fmt.Errorf("quotes.buildExpr: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:                 aws.String(TableQuotes),
		Item:                      av,
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		var ccf *types.ConditionalCheckFailedException
		if errors.As(err, &ccf) {
			return core.ErrConflict
		}
		return fmt.Errorf("quotes.putItem: %w", err)
	}

	return nil
}

func (r *QuoteRepo) Get(ctx context.Context, id string) (core.Quote, error) {
	out, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(TableQuotes),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return core.Quote{}, fmt.Errorf("quotes.getItem: %w", err)
	}

	if out.Item == nil {
		return core.Quote{}, core.ErrQuoteNotFound
	}

	var item QuoteItem
	if err := attributevalue.UnmarshalMap(out.Item, &item); err != nil {
		return core.Quote{}, fmt.Errorf("quotes.unmarshal: %w", err)
	}

	return item.ToCore(), nil
}
