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

type ApplicantItem struct {
	FirstName   string `dynamodbav:"first_name"`
	LastName    string `dynamodbav:"last_name"`
	Email       string `dynamodbav:"email"`
	DateOfBirth string `dynamodbav:"date_of_birth"`
	Age         int    `dynamodbav:"age"`
	Smoker      bool   `dynamodbav:"smoker"`
	State       string `dynamodbav:"state"`
}

type ApplicationItem struct {
	ID             string        `dynamodbav:"id"`
	QuoteID        string        `dynamodbav:"quote_id"`
	ProductID      string        `dynamodbav:"product_id"`
	ProductSlug    string        `dynamodbav:"product_slug"`
	CoverageAmount int64         `dynamodbav:"coverage_amount"`
	TermYears      int           `dynamodbav:"term_years"`
	MonthlyPremium float64       `dynamodbav:"monthly_premium"`
	Applicant      ApplicantItem `dynamodbav:"applicant"`
	Status         string        `dynamodbav:"status"`
	CreatedAt      string        `dynamodbav:"created_at"`
	UpdatedAt      string        `dynamodbav:"updated_at"`
	SubmittedAt    string        `dynamodbav:"submitted_at,omitempty"`
}

func (i ApplicationItem) ToCore() core.Application {
	createdAt, _ := time.Parse(time.RFC3339, i.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, i.UpdatedAt)
	var submittedAt *time.Time
	if i.SubmittedAt != "" {
		t, _ := time.Parse(time.RFC3339, i.SubmittedAt)
		submittedAt = &t
	}
	return core.Application{
		ID:             i.ID,
		QuoteID:        i.QuoteID,
		ProductID:      i.ProductID,
		ProductSlug:    i.ProductSlug,
		CoverageAmount: i.CoverageAmount,
		TermYears:      i.TermYears,
		MonthlyPremium: i.MonthlyPremium,
		Applicant: core.Applicant{
			FirstName:   i.Applicant.FirstName,
			LastName:    i.Applicant.LastName,
			Email:       i.Applicant.Email,
			DateOfBirth: i.Applicant.DateOfBirth,
			Age:         i.Applicant.Age,
			Smoker:      i.Applicant.Smoker,
			State:       i.Applicant.State,
		},
		Status:      core.ApplicationStatus(i.Status),
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		SubmittedAt: submittedAt,
	}
}

func applicationItemFromCore(a core.Application) ApplicationItem {
	item := ApplicationItem{
		ID:             a.ID,
		QuoteID:        a.QuoteID,
		ProductID:      a.ProductID,
		ProductSlug:    a.ProductSlug,
		CoverageAmount: a.CoverageAmount,
		TermYears:      a.TermYears,
		MonthlyPremium: a.MonthlyPremium,
		Applicant: ApplicantItem{
			FirstName:   a.Applicant.FirstName,
			LastName:    a.Applicant.LastName,
			Email:       a.Applicant.Email,
			DateOfBirth: a.Applicant.DateOfBirth,
			Age:         a.Applicant.Age,
			Smoker:      a.Applicant.Smoker,
			State:       a.Applicant.State,
		},
		Status:    string(a.Status),
		CreatedAt: a.CreatedAt.Format(time.RFC3339),
		UpdatedAt: a.UpdatedAt.Format(time.RFC3339),
	}
	if a.SubmittedAt != nil {
		item.SubmittedAt = a.SubmittedAt.Format(time.RFC3339)
	}
	return item
}

type ApplicationRepo struct {
	client *dynamodb.Client
}

func NewApplicationRepo(client *dynamodb.Client) *ApplicationRepo {
	return &ApplicationRepo{client: client}
}

func (r *ApplicationRepo) Create(ctx context.Context, app core.Application) error {
	item := applicationItemFromCore(app)
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("applications.marshal: %w", err)
	}

	cond := expression.AttributeNotExists(expression.Name("id"))
	expr, err := expression.NewBuilder().WithCondition(cond).Build()
	if err != nil {
		return fmt.Errorf("applications.buildExpr: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:                 aws.String(TableApplications),
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
		return fmt.Errorf("applications.putItem: %w", err)
	}

	return nil
}

func (r *ApplicationRepo) Get(ctx context.Context, id string) (core.Application, error) {
	out, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(TableApplications),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return core.Application{}, fmt.Errorf("applications.getItem: %w", err)
	}

	if out.Item == nil {
		return core.Application{}, core.ErrApplicationNotFound
	}

	var item ApplicationItem
	if err := attributevalue.UnmarshalMap(out.Item, &item); err != nil {
		return core.Application{}, fmt.Errorf("applications.unmarshal: %w", err)
	}

	return item.ToCore(), nil
}

func (r *ApplicationRepo) Update(ctx context.Context, app core.Application) error {
	item := applicationItemFromCore(app)
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("applications.marshal: %w", err)
	}

	cond := expression.AttributeExists(expression.Name("id"))
	expr, err := expression.NewBuilder().WithCondition(cond).Build()
	if err != nil {
		return fmt.Errorf("applications.buildExpr: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:                 aws.String(TableApplications),
		Item:                      av,
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		var ccf *types.ConditionalCheckFailedException
		if errors.As(err, &ccf) {
			return core.ErrApplicationNotFound
		}
		return fmt.Errorf("applications.putItem: %w", err)
	}

	return nil
}

func (r *ApplicationRepo) UpdateStatus(ctx context.Context, id string, status core.ApplicationStatus, updatedAt time.Time) error {
	update := expression.Set(
		expression.Name("status"), expression.Value(string(status)),
	).Set(
		expression.Name("updated_at"), expression.Value(updatedAt.Format(time.RFC3339)),
	)
	cond := expression.AttributeExists(expression.Name("id"))
	expr, err := expression.NewBuilder().WithUpdate(update).WithCondition(cond).Build()
	if err != nil {
		return fmt.Errorf("applications.buildExpr: %w", err)
	}

	_, err = r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(TableApplications),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		UpdateExpression:          expr.Update(),
		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		var ccf *types.ConditionalCheckFailedException
		if errors.As(err, &ccf) {
			return core.ErrApplicationNotFound
		}
		return fmt.Errorf("applications.updateItem: %w", err)
	}

	return nil
}

func (r *ApplicationRepo) FindByStatus(ctx context.Context, status core.ApplicationStatus, limit int) ([]core.Application, error) {
	out, err := r.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(TableApplications),
		IndexName:              aws.String(GSIApplicationsStatus),
		KeyConditionExpression: aws.String("#status = :status"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status": &types.AttributeValueMemberS{Value: string(status)},
		},
		Limit: aws.Int32(int32(limit)),
	})
	if err != nil {
		return nil, fmt.Errorf("applications.query: %w", err)
	}

	var items []ApplicationItem
	if err := attributevalue.UnmarshalListOfMaps(out.Items, &items); err != nil {
		return nil, fmt.Errorf("applications.unmarshal: %w", err)
	}

	apps := make([]core.Application, len(items))
	for i, item := range items {
		apps[i] = item.ToCore()
	}
	return apps, nil
}
