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

type RiskFactorsItem struct {
	Age            int   `dynamodbav:"age"`
	Smoker         bool  `dynamodbav:"smoker"`
	CoverageAmount int64 `dynamodbav:"coverage_amount"`
	TermYears      int   `dynamodbav:"term_years"`
}

type RiskScoreItem struct {
	Score       int      `dynamodbav:"score"`
	Flags       []string `dynamodbav:"flags"`
	Recommended string   `dynamodbav:"recommended"`
}

type UnderwritingCaseItem struct {
	ID            string          `dynamodbav:"id"`
	ApplicationID string          `dynamodbav:"application_id"`
	RiskFactors   RiskFactorsItem `dynamodbav:"risk_factors"`
	RiskScore     RiskScoreItem   `dynamodbav:"risk_score"`
	Decision      string          `dynamodbav:"decision"`
	Method        string          `dynamodbav:"method"`
	DecidedBy     string          `dynamodbav:"decided_by"`
	Reason        string          `dynamodbav:"reason"`
	CreatedAt     string          `dynamodbav:"created_at"`
	UpdatedAt     string          `dynamodbav:"updated_at"`
	DecidedAt     string          `dynamodbav:"decided_at,omitempty"`
}

func (i UnderwritingCaseItem) ToCore() core.UnderwritingCase {
	createdAt, _ := time.Parse(time.RFC3339, i.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, i.UpdatedAt)
	var decidedAt *time.Time
	if i.DecidedAt != "" {
		t, _ := time.Parse(time.RFC3339, i.DecidedAt)
		decidedAt = &t
	}

	flags := i.RiskScore.Flags
	if flags == nil {
		flags = []string{}
	}

	return core.UnderwritingCase{
		ID:            i.ID,
		ApplicationID: i.ApplicationID,
		RiskFactors: core.RiskFactors{
			Age:            i.RiskFactors.Age,
			Smoker:         i.RiskFactors.Smoker,
			CoverageAmount: i.RiskFactors.CoverageAmount,
			TermYears:      i.RiskFactors.TermYears,
		},
		RiskScore: core.RiskScore{
			Score:       i.RiskScore.Score,
			Flags:       flags,
			Recommended: core.UWDecision(i.RiskScore.Recommended),
		},
		Decision:  core.UWDecision(i.Decision),
		Method:    core.UWMethod(i.Method),
		DecidedBy: i.DecidedBy,
		Reason:    i.Reason,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		DecidedAt: decidedAt,
	}
}

func uwCaseItemFromCore(uw core.UnderwritingCase) UnderwritingCaseItem {
	item := UnderwritingCaseItem{
		ID:            uw.ID,
		ApplicationID: uw.ApplicationID,
		RiskFactors: RiskFactorsItem{
			Age:            uw.RiskFactors.Age,
			Smoker:         uw.RiskFactors.Smoker,
			CoverageAmount: uw.RiskFactors.CoverageAmount,
			TermYears:      uw.RiskFactors.TermYears,
		},
		RiskScore: RiskScoreItem{
			Score:       uw.RiskScore.Score,
			Flags:       uw.RiskScore.Flags,
			Recommended: string(uw.RiskScore.Recommended),
		},
		Decision:  string(uw.Decision),
		Method:    string(uw.Method),
		DecidedBy: uw.DecidedBy,
		Reason:    uw.Reason,
		CreatedAt: uw.CreatedAt.Format(time.RFC3339),
		UpdatedAt: uw.UpdatedAt.Format(time.RFC3339),
	}
	if uw.DecidedAt != nil {
		item.DecidedAt = uw.DecidedAt.Format(time.RFC3339)
	}
	return item
}

type UnderwritingRepo struct {
	client *dynamodb.Client
}

func NewUnderwritingRepo(client *dynamodb.Client) *UnderwritingRepo {
	return &UnderwritingRepo{client: client}
}

func (r *UnderwritingRepo) Create(ctx context.Context, uw core.UnderwritingCase) error {
	item := uwCaseItemFromCore(uw)
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("underwriting.marshal: %w", err)
	}

	cond := expression.AttributeNotExists(expression.Name("id"))
	expr, err := expression.NewBuilder().WithCondition(cond).Build()
	if err != nil {
		return fmt.Errorf("underwriting.buildExpr: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:                 aws.String(TableUWCases),
		Item:                      av,
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		var ccf *types.ConditionalCheckFailedException
		if errors.As(err, &ccf) {
			return core.ErrUWCaseExists
		}
		return fmt.Errorf("underwriting.putItem: %w", err)
	}

	return nil
}

func (r *UnderwritingRepo) Get(ctx context.Context, id string) (core.UnderwritingCase, error) {
	out, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(TableUWCases),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return core.UnderwritingCase{}, fmt.Errorf("underwriting.getItem: %w", err)
	}

	if out.Item == nil {
		return core.UnderwritingCase{}, core.ErrUWCaseNotFound
	}

	var item UnderwritingCaseItem
	if err := attributevalue.UnmarshalMap(out.Item, &item); err != nil {
		return core.UnderwritingCase{}, fmt.Errorf("underwriting.unmarshal: %w", err)
	}

	return item.ToCore(), nil
}

func (r *UnderwritingRepo) GetByApplicationID(ctx context.Context, appID string) (core.UnderwritingCase, error) {
	out, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(TableUWCases),
		IndexName:              aws.String(GSIUWCasesAppID),
		KeyConditionExpression: aws.String("application_id = :app_id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":app_id": &types.AttributeValueMemberS{Value: appID},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return core.UnderwritingCase{}, fmt.Errorf("underwriting.query: %w", err)
	}

	if len(out.Items) == 0 {
		return core.UnderwritingCase{}, core.ErrUWCaseNotFound
	}

	var item UnderwritingCaseItem
	if err := attributevalue.UnmarshalMap(out.Items[0], &item); err != nil {
		return core.UnderwritingCase{}, fmt.Errorf("underwriting.unmarshal: %w", err)
	}

	return item.ToCore(), nil
}

func (r *UnderwritingRepo) Update(ctx context.Context, uw core.UnderwritingCase) error {
	item := uwCaseItemFromCore(uw)
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("underwriting.marshal: %w", err)
	}

	cond := expression.AttributeExists(expression.Name("id"))
	expr, err := expression.NewBuilder().WithCondition(cond).Build()
	if err != nil {
		return fmt.Errorf("underwriting.buildExpr: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:                 aws.String(TableUWCases),
		Item:                      av,
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		var ccf *types.ConditionalCheckFailedException
		if errors.As(err, &ccf) {
			return core.ErrUWCaseNotFound
		}
		return fmt.Errorf("underwriting.putItem: %w", err)
	}

	return nil
}

func (r *UnderwritingRepo) FindPending(ctx context.Context, limit int) ([]core.UnderwritingCase, error) {
	return r.findByDecision(ctx, string(core.UWDecisionPending), limit)
}

func (r *UnderwritingRepo) FindReferred(ctx context.Context, limit int) ([]core.UnderwritingCase, error) {
	return r.findByDecision(ctx, string(core.UWDecisionReferred), limit)
}

func (r *UnderwritingRepo) findByDecision(ctx context.Context, decision string, limit int) ([]core.UnderwritingCase, error) {
	out, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(TableUWCases),
		IndexName:              aws.String(GSIUWCasesDecision),
		KeyConditionExpression: aws.String("decision = :decision"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":decision": &types.AttributeValueMemberS{Value: decision},
		},
		Limit: aws.Int32(int32(limit)),
	})
	if err != nil {
		return nil, fmt.Errorf("underwriting.query: %w", err)
	}

	var items []UnderwritingCaseItem
	if err := attributevalue.UnmarshalListOfMaps(out.Items, &items); err != nil {
		return nil, fmt.Errorf("underwriting.unmarshal: %w", err)
	}

	cases := make([]core.UnderwritingCase, len(items))
	for i, item := range items {
		cases[i] = item.ToCore()
	}
	return cases, nil
}
