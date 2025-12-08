# Go Insurance API

A Life Insurance Quote and Policy Management API built with Go. This project demonstrates a complete insurance workflow from quoting to policy issuance.

## Features

- **Product Catalog** - Browse available term life insurance products
- **Quote Engine** - Real-time pricing based on applicant factors
- **Application Management** - Create and submit insurance applications
- **Auto-Underwriting** - Rules-based risk scoring with auto-approve/decline
- **Manual Review** - Referred cases queue for underwriters
- **Offer Management** - 30-day validity period, accept/decline workflow
- **Policy Issuance** - Automatic policy generation from accepted offers
- **Background Workers** - Async processing for underwriting and issuance

## Architecture

```
cmd/
  api/          - Main application entry point
  seed/         - Database seeding utility
internal/
  core/         - Domain models, services, business logic
  http/         - HTTP handlers and routing
  jobs/         - Background workers
  middleware/   - HTTP middleware
  platform/     - Infrastructure (config, logging, IDs)
  store/
    dynamo/     - DynamoDB repositories
    mongo/      - MongoDB repositories
docs/           - Swagger/OpenAPI documentation
pkg/
  problem/      - RFC 7807 Problem Details responses
```

## API Workflow

```
1. GET  /products              → Browse insurance products
2. POST /quotes                → Get pricing
3. POST /applications          → Create application
4. POST /applications/:id:submit → Submit for underwriting
     ↓ (Background worker processes)
5. GET  /underwriting/cases    → View referred cases (if not auto-decided)
6. POST /underwriting/cases/:id:decide → Manual decision
     ↓ (Offer auto-generated on approval)
7. GET  /offers/:id            → View offer
8. POST /offers/:id:accept     → Accept offer
     ↓ (Background worker issues policy)
9. GET  /policies/:number      → View issued policy
```

## Quick Start

### Prerequisites

- Go 1.21+
- DynamoDB Local or AWS DynamoDB (default)
- OR MongoDB 6+ (alternative)

### DynamoDB Setup (Default)

```bash
# 1. Start DynamoDB Local
docker compose up -d dynamodb

# 2. Configure environment
cp .env.example .env
# (defaults are already set for DynamoDB)

# 3. Run the application (tables created automatically)
go run cmd/api/main.go

# 4. Seed products
go run cmd/seed/main.go
```

### MongoDB Setup (Alternative)

```bash
# 1. Start MongoDB
docker compose up -d mongo

# 2. Configure environment
# Edit .env:
#   DB_TYPE=mongo
#   MONGO_URI=mongodb://go_insurance_user:go_insurance_pass@localhost:27017
#   MONGO_DB=go_insurance_db

# 3. Run and seed (same as above)
go run cmd/api/main.go
go run cmd/seed/main.go
```

## API Documentation

Swagger UI is available at: **http://localhost:8080/swagger/**

### Endpoints Overview

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /health | Health check |
| GET | /ready | Readiness check (includes DB ping) |
| GET | /api/v1/products | List all products |
| GET | /api/v1/products/{slug} | Get product by slug |
| POST | /api/v1/quotes | Create a quote |
| GET | /api/v1/quotes/{id} | Get a quote |
| POST | /api/v1/applications | Create an application |
| GET | /api/v1/applications/{id} | Get an application |
| PATCH | /api/v1/applications/{id} | Update application (draft only) |
| POST | /api/v1/applications/{id}:submit | Submit for underwriting |
| GET | /api/v1/underwriting/cases | List referred cases |
| GET | /api/v1/underwriting/cases/{id} | Get UW case details |
| POST | /api/v1/underwriting/cases/{id}:decide | Manual decision |
| POST | /api/v1/applications/{id}/offers | Generate offer |
| GET | /api/v1/offers/{id} | Get offer |
| POST | /api/v1/offers/{id}:accept | Accept offer |
| POST | /api/v1/offers/{id}:decline | Decline offer |
| GET | /api/v1/policies | List policies |
| GET | /api/v1/policies/{number} | Get policy by number |

## Auto-Underwriting Rules

| Condition | Decision |
|-----------|----------|
| Age > 80 | Auto-decline |
| Age < 45 AND non-smoker AND coverage < $250k AND score <= 30 | Auto-approve |
| Score <= 20 | Auto-approve |
| All other cases | Refer to manual review |

### Risk Score Components
- Age 51-60: +25 | Age 61-65: +35 | Age 66-80: +50
- Smoker: +25
- Coverage $100k-$250k: +10 | $250k-$500k: +15 | > $500k: +25

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8080 | HTTP server port |
| ENV | dev | Environment (dev/prod) |
| DB_TYPE | dynamodb | Database type (dynamodb/mongo) |
| AWS_REGION | us-east-1 | AWS region for DynamoDB |
| DYNAMODB_ENDPOINT | | Local DynamoDB endpoint (leave empty for AWS) |
| AWS_ACCESS_KEY_ID | | AWS credentials (optional for local) |
| AWS_SECRET_ACCESS_KEY | | AWS credentials (optional for local) |
| MONGO_URI | | MongoDB connection string |
| MONGO_DB | go_insurance | MongoDB database name |
| WORKER_INTERVAL_SEC | 5 | Background worker polling interval |
| HTTP_REQUEST_TIMEOUT_SEC | 30 | HTTP request timeout |

## Example Usage

```bash
# 1. Create a quote
curl -X POST http://localhost:8080/api/v1/quotes \
  -H "Content-Type: application/json" \
  -d '{
    "product_slug": "term-life-10",
    "coverage_amount": 150000,
    "term_years": 10,
    "age": 35,
    "smoker": false
  }'

# 2. Create an application (use quote_id from step 1)
curl -X POST http://localhost:8080/api/v1/applications \
  -H "Content-Type: application/json" \
  -d '{
    "quote_id": "YOUR_QUOTE_ID",
    "applicant": {
      "first_name": "John",
      "last_name": "Doe",
      "email": "john@example.com",
      "date_of_birth": "1989-06-15",
      "age": 35,
      "smoker": false,
      "state": "CA"
    }
  }'

# 3. Submit application (use application_id from step 2)
curl -X POST http://localhost:8080/api/v1/applications/YOUR_APP_ID:submit

# 4. Wait for worker to process, then check for offer
# (For auto-approved cases, offer is created automatically)

# 5. Accept offer (use offer_id)
curl -X POST http://localhost:8080/api/v1/offers/YOUR_OFFER_ID:accept

# 6. Check for issued policy
curl http://localhost:8080/api/v1/policies
```

## AWS Deployment

For production DynamoDB:

1. Set `DYNAMODB_ENDPOINT` to empty (uses AWS)
2. Configure proper IAM credentials
3. Tables are created automatically on first run

DynamoDB tables created:
- `insurance_products`
- `insurance_quotes`
- `insurance_applications`
- `insurance_underwriting_cases`
- `insurance_offers`
- `insurance_policies`
- `insurance_counters`

## Tech Stack

- **Go 1.21+** - Language
- **Chi** - HTTP router
- **AWS SDK v2** - DynamoDB client
- **MongoDB Driver** - MongoDB client (alternative)
- **ULID** - Unique identifiers
- **Swagger** - API documentation

## License

MIT
