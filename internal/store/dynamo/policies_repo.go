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

type PolicyItem struct {
	ID             string        `dynamodbav:"id"`
	Number         string        `dynamodbav:"number"`
	ApplicationID  string        `dynamodbav:"application_id"`
	OfferID        string        `dynamodbav:"offer_id"`
	ProductSlug    string        `dynamodbav:"product_slug"`
	CoverageAmount int64         `dynamodbav:"coverage_amount"`
	TermYears      int           `dynamodbav:"term_years"`
	MonthlyPremium float64       `dynamodbav:"monthly_premium"`
	Insured        ApplicantItem `dynamodbav:"insured"`
	Status         string        `dynamodbav:"status"`
	EffectiveDate  string        `dynamodbav:"effective_date"`
	ExpiryDate     string        `dynamodbav:"expiry_date"`
	IssuedAt       string        `dynamodbav:"issued_at"`
}

func (i PolicyItem) ToCore() core.Policy {
	effectiveDate, _ := time.Parse(time.RFC3339, i.EffectiveDate)
	expiryDate, _ := time.Parse(time.RFC3339, i.ExpiryDate)
	issuedAt, _ := time.Parse(time.RFC3339, i.IssuedAt)
	return core.Policy{
		ID:             i.ID,
		Number:         i.Number,
		ApplicationID:  i.ApplicationID,
		OfferID:        i.OfferID,
		ProductSlug:    i.ProductSlug,
		CoverageAmount: i.CoverageAmount,
		TermYears:      i.TermYears,
		MonthlyPremium: i.MonthlyPremium,
		Insured: core.Applicant{
			FirstName:   i.Insured.FirstName,
			LastName:    i.Insured.LastName,
			Email:       i.Insured.Email,
			DateOfBirth: i.Insured.DateOfBirth,
			Age:         i.Insured.Age,
			Smoker:      i.Insured.Smoker,
			State:       i.Insured.State,
		},
		Status:        core.PolicyStatus(i.Status),
		EffectiveDate: effectiveDate,
		ExpiryDate:    expiryDate,
		IssuedAt:      issuedAt,
	}
}

func policyItemFromCore(p core.Policy) PolicyItem {
	return PolicyItem{
		ID:             p.ID,
		Number:         p.Number,
		ApplicationID:  p.ApplicationID,
		OfferID:        p.OfferID,
		ProductSlug:    p.ProductSlug,
		CoverageAmount: p.CoverageAmount,
		TermYears:      p.TermYears,
		MonthlyPremium: p.MonthlyPremium,
		Insured: ApplicantItem{
			FirstName:   p.Insured.FirstName,
			LastName:    p.Insured.LastName,
			Email:       p.Insured.Email,
			DateOfBirth: p.Insured.DateOfBirth,
			Age:         p.Insured.Age,
			Smoker:      p.Insured.Smoker,
			State:       p.Insured.State,
		},
		Status:        string(p.Status),
		EffectiveDate: p.EffectiveDate.Format(time.RFC3339),
		ExpiryDate:    p.ExpiryDate.Format(time.RFC3339),
		IssuedAt:      p.IssuedAt.Format(time.RFC3339),
	}
}

type PolicyRepo struct {
	client *dynamodb.Client
}

func NewPolicyRepo(client *dynamodb.Client) *PolicyRepo {
	return &PolicyRepo{client: client}
}

func (r *PolicyRepo) Create(ctx context.Context, policy core.Policy) error {
	item := policyItemFromCore(policy)
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("policies.marshal: %w", err)
	}

	cond := expression.AttributeNotExists(expression.Name("id"))
	expr, err := expression.NewBuilder().WithCondition(cond).Build()
	if err != nil {
		return fmt.Errorf("policies.buildExpr: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:                 aws.String(TablePolicies),
		Item:                      av,
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		var ccf *types.ConditionalCheckFailedException
		if errors.As(err, &ccf) {
			return core.ErrPolicyExists
		}
		return fmt.Errorf("policies.putItem: %w", err)
	}

	return nil
}

func (r *PolicyRepo) Get(ctx context.Context, id string) (core.Policy, error) {
	out, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(TablePolicies),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return core.Policy{}, fmt.Errorf("policies.getItem: %w", err)
	}

	if out.Item == nil {
		return core.Policy{}, core.ErrPolicyNotFound
	}

	var item PolicyItem
	if err := attributevalue.UnmarshalMap(out.Item, &item); err != nil {
		return core.Policy{}, fmt.Errorf("policies.unmarshal: %w", err)
	}

	return item.ToCore(), nil
}

func (r *PolicyRepo) GetByNumber(ctx context.Context, number string) (core.Policy, error) {
	out, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(TablePolicies),
		IndexName:              aws.String(GSIPoliciesNumber),
		KeyConditionExpression: aws.String("#number = :number"),
		ExpressionAttributeNames: map[string]string{
			"#number": "number",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":number": &types.AttributeValueMemberS{Value: number},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return core.Policy{}, fmt.Errorf("policies.query: %w", err)
	}

	if len(out.Items) == 0 {
		return core.Policy{}, core.ErrPolicyNotFound
	}

	var item PolicyItem
	if err := attributevalue.UnmarshalMap(out.Items[0], &item); err != nil {
		return core.Policy{}, fmt.Errorf("policies.unmarshal: %w", err)
	}

	return item.ToCore(), nil
}

func (r *PolicyRepo) GetByOfferID(ctx context.Context, offerID string) (core.Policy, error) {
	out, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(TablePolicies),
		IndexName:              aws.String(GSIPoliciesOfferID),
		KeyConditionExpression: aws.String("offer_id = :offer_id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":offer_id": &types.AttributeValueMemberS{Value: offerID},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return core.Policy{}, fmt.Errorf("policies.query: %w", err)
	}

	if len(out.Items) == 0 {
		return core.Policy{}, core.ErrPolicyNotFound
	}

	var item PolicyItem
	if err := attributevalue.UnmarshalMap(out.Items[0], &item); err != nil {
		return core.Policy{}, fmt.Errorf("policies.unmarshal: %w", err)
	}

	return item.ToCore(), nil
}

func (r *PolicyRepo) GetByApplicationID(ctx context.Context, appID string) (core.Policy, error) {
	out, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(TablePolicies),
		IndexName:              aws.String(GSIPoliciesAppID),
		KeyConditionExpression: aws.String("application_id = :app_id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":app_id": &types.AttributeValueMemberS{Value: appID},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return core.Policy{}, fmt.Errorf("policies.query: %w", err)
	}

	if len(out.Items) == 0 {
		return core.Policy{}, core.ErrPolicyNotFound
	}

	var item PolicyItem
	if err := attributevalue.UnmarshalMap(out.Items[0], &item); err != nil {
		return core.Policy{}, fmt.Errorf("policies.unmarshal: %w", err)
	}

	return item.ToCore(), nil
}

func (r *PolicyRepo) List(ctx context.Context, filter core.PolicyFilter, limit, offset int) ([]core.Policy, int64, error) {
	// For DynamoDB, we'll use Scan with filters (not ideal for large datasets)
	// In production, consider using GSIs for common query patterns
	var filterExpr expression.ConditionBuilder
	hasFilter := false

	if filter.ApplicationID != "" {
		filterExpr = expression.Name("application_id").Equal(expression.Value(filter.ApplicationID))
		hasFilter = true
	}
	if filter.Status != "" {
		statusFilter := expression.Name("status").Equal(expression.Value(string(filter.Status)))
		if hasFilter {
			filterExpr = filterExpr.And(statusFilter)
		} else {
			filterExpr = statusFilter
			hasFilter = true
		}
	}

	var scanInput *dynamodb.ScanInput
	if hasFilter {
		expr, err := expression.NewBuilder().WithFilter(filterExpr).Build()
		if err != nil {
			return nil, 0, fmt.Errorf("policies.buildExpr: %w", err)
		}
		scanInput = &dynamodb.ScanInput{
			TableName:                 aws.String(TablePolicies),
			FilterExpression:          expr.Filter(),
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
		}
	} else {
		scanInput = &dynamodb.ScanInput{
			TableName: aws.String(TablePolicies),
		}
	}

	out, err := r.client.Scan(ctx, scanInput)
	if err != nil {
		return nil, 0, fmt.Errorf("policies.scan: %w", err)
	}

	var items []PolicyItem
	if err := attributevalue.UnmarshalListOfMaps(out.Items, &items); err != nil {
		return nil, 0, fmt.Errorf("policies.unmarshal: %w", err)
	}

	total := int64(len(items))

	// Apply offset and limit manually (DynamoDB pagination is different)
	if offset >= len(items) {
		return []core.Policy{}, total, nil
	}

	end := offset + limit
	if end > len(items) {
		end = len(items)
	}

	items = items[offset:end]

	policies := make([]core.Policy, len(items))
	for i, item := range items {
		policies[i] = item.ToCore()
	}

	return policies, total, nil
}

func (r *PolicyRepo) NextPolicyNumber(ctx context.Context) (string, error) {
	// Use atomic counter for policy numbers
	year := time.Now().Year()
	counterName := fmt.Sprintf("policy_%d", year)

	out, err := r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(TableCounters),
		Key: map[string]types.AttributeValue{
			"counter_name": &types.AttributeValueMemberS{Value: counterName},
		},
		UpdateExpression: aws.String("SET counter_value = if_not_exists(counter_value, :zero) + :inc"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":zero": &types.AttributeValueMemberN{Value: "0"},
			":inc":  &types.AttributeValueMemberN{Value: "1"},
		},
		ReturnValues: types.ReturnValueUpdatedNew,
	})
	if err != nil {
		return "", fmt.Errorf("counters.updateItem: %w", err)
	}

	counterValue := out.Attributes["counter_value"].(*types.AttributeValueMemberN).Value
	// Parse counter value to int for proper zero-padding
	var num int
	fmt.Sscanf(counterValue, "%d", &num)
	return fmt.Sprintf("POL-%d-%06d", year, num), nil
}
