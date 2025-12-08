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

type OfferItem struct {
	ID             string  `dynamodbav:"id"`
	ApplicationID  string  `dynamodbav:"application_id"`
	ProductSlug    string  `dynamodbav:"product_slug"`
	CoverageAmount int64   `dynamodbav:"coverage_amount"`
	TermYears      int     `dynamodbav:"term_years"`
	MonthlyPremium float64 `dynamodbav:"monthly_premium"`
	Status         string  `dynamodbav:"status"`
	CreatedAt      string  `dynamodbav:"created_at"`
	ExpiresAt      string  `dynamodbav:"expires_at"`
	AcceptedAt     string  `dynamodbav:"accepted_at,omitempty"`
	DeclinedAt     string  `dynamodbav:"declined_at,omitempty"`
}

func (i OfferItem) ToCore() core.Offer {
	createdAt, _ := time.Parse(time.RFC3339, i.CreatedAt)
	expiresAt, _ := time.Parse(time.RFC3339, i.ExpiresAt)
	var acceptedAt, declinedAt *time.Time
	if i.AcceptedAt != "" {
		t, _ := time.Parse(time.RFC3339, i.AcceptedAt)
		acceptedAt = &t
	}
	if i.DeclinedAt != "" {
		t, _ := time.Parse(time.RFC3339, i.DeclinedAt)
		declinedAt = &t
	}
	return core.Offer{
		ID:             i.ID,
		ApplicationID:  i.ApplicationID,
		ProductSlug:    i.ProductSlug,
		CoverageAmount: i.CoverageAmount,
		TermYears:      i.TermYears,
		MonthlyPremium: i.MonthlyPremium,
		Status:         core.OfferStatus(i.Status),
		CreatedAt:      createdAt,
		ExpiresAt:      expiresAt,
		AcceptedAt:     acceptedAt,
		DeclinedAt:     declinedAt,
	}
}

func offerItemFromCore(o core.Offer) OfferItem {
	item := OfferItem{
		ID:             o.ID,
		ApplicationID:  o.ApplicationID,
		ProductSlug:    o.ProductSlug,
		CoverageAmount: o.CoverageAmount,
		TermYears:      o.TermYears,
		MonthlyPremium: o.MonthlyPremium,
		Status:         string(o.Status),
		CreatedAt:      o.CreatedAt.Format(time.RFC3339),
		ExpiresAt:      o.ExpiresAt.Format(time.RFC3339),
	}
	if o.AcceptedAt != nil {
		item.AcceptedAt = o.AcceptedAt.Format(time.RFC3339)
	}
	if o.DeclinedAt != nil {
		item.DeclinedAt = o.DeclinedAt.Format(time.RFC3339)
	}
	return item
}

type OfferRepo struct {
	client *dynamodb.Client
}

func NewOfferRepo(client *dynamodb.Client) *OfferRepo {
	return &OfferRepo{client: client}
}

func (r *OfferRepo) Create(ctx context.Context, offer core.Offer) error {
	item := offerItemFromCore(offer)
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("offers.marshal: %w", err)
	}

	cond := expression.AttributeNotExists(expression.Name("id"))
	expr, err := expression.NewBuilder().WithCondition(cond).Build()
	if err != nil {
		return fmt.Errorf("offers.buildExpr: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:                 aws.String(TableOffers),
		Item:                      av,
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		var ccf *types.ConditionalCheckFailedException
		if errors.As(err, &ccf) {
			return core.ErrOfferExists
		}
		return fmt.Errorf("offers.putItem: %w", err)
	}

	return nil
}

func (r *OfferRepo) Get(ctx context.Context, id string) (core.Offer, error) {
	out, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(TableOffers),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return core.Offer{}, fmt.Errorf("offers.getItem: %w", err)
	}

	if out.Item == nil {
		return core.Offer{}, core.ErrOfferNotFound
	}

	var item OfferItem
	if err := attributevalue.UnmarshalMap(out.Item, &item); err != nil {
		return core.Offer{}, fmt.Errorf("offers.unmarshal: %w", err)
	}

	return item.ToCore(), nil
}

func (r *OfferRepo) GetByApplicationID(ctx context.Context, appID string) (core.Offer, error) {
	out, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(TableOffers),
		IndexName:              aws.String(GSIOffersAppID),
		KeyConditionExpression: aws.String("application_id = :app_id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":app_id": &types.AttributeValueMemberS{Value: appID},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return core.Offer{}, fmt.Errorf("offers.query: %w", err)
	}

	if len(out.Items) == 0 {
		return core.Offer{}, core.ErrOfferNotFound
	}

	var item OfferItem
	if err := attributevalue.UnmarshalMap(out.Items[0], &item); err != nil {
		return core.Offer{}, fmt.Errorf("offers.unmarshal: %w", err)
	}

	return item.ToCore(), nil
}

func (r *OfferRepo) Update(ctx context.Context, offer core.Offer) error {
	item := offerItemFromCore(offer)
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("offers.marshal: %w", err)
	}

	cond := expression.AttributeExists(expression.Name("id"))
	expr, err := expression.NewBuilder().WithCondition(cond).Build()
	if err != nil {
		return fmt.Errorf("offers.buildExpr: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:                 aws.String(TableOffers),
		Item:                      av,
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		var ccf *types.ConditionalCheckFailedException
		if errors.As(err, &ccf) {
			return core.ErrOfferNotFound
		}
		return fmt.Errorf("offers.putItem: %w", err)
	}

	return nil
}

func (r *OfferRepo) FindAccepted(ctx context.Context, limit int) ([]core.Offer, error) {
	out, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(TableOffers),
		IndexName:              aws.String(GSIOffersStatus),
		KeyConditionExpression: aws.String("#status = :status"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status": &types.AttributeValueMemberS{Value: string(core.OfferStatusAccepted)},
		},
		Limit: aws.Int32(int32(limit)),
	})
	if err != nil {
		return nil, fmt.Errorf("offers.query: %w", err)
	}

	var items []OfferItem
	if err := attributevalue.UnmarshalListOfMaps(out.Items, &items); err != nil {
		return nil, fmt.Errorf("offers.unmarshal: %w", err)
	}

	offers := make([]core.Offer, len(items))
	for i, item := range items {
		offers[i] = item.ToCore()
	}
	return offers, nil
}

func (r *OfferRepo) ExpireOffers(ctx context.Context, before time.Time) (int64, error) {
	// DynamoDB doesn't support bulk updates like MongoDB
	// We need to scan and update individually (or use batch write)
	// For simplicity, this is a basic implementation
	out, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(TableOffers),
		IndexName:              aws.String(GSIOffersStatus),
		KeyConditionExpression: aws.String("#status = :status"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status": &types.AttributeValueMemberS{Value: string(core.OfferStatusPending)},
		},
	})
	if err != nil {
		return 0, fmt.Errorf("offers.query: %w", err)
	}

	var count int64
	for _, item := range out.Items {
		var offer OfferItem
		if err := attributevalue.UnmarshalMap(item, &offer); err != nil {
			continue
		}

		expiresAt, _ := time.Parse(time.RFC3339, offer.ExpiresAt)
		if expiresAt.Before(before) {
			// Update to expired
			update := expression.Set(expression.Name("status"), expression.Value(string(core.OfferStatusExpired)))
			expr, _ := expression.NewBuilder().WithUpdate(update).Build()

			_, err := r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
				TableName: aws.String(TableOffers),
				Key: map[string]types.AttributeValue{
					"id": &types.AttributeValueMemberS{Value: offer.ID},
				},
				UpdateExpression:          expr.Update(),
				ExpressionAttributeNames:  expr.Names(),
				ExpressionAttributeValues: expr.Values(),
			})
			if err == nil {
				count++
			}
		}
	}

	return count, nil
}
