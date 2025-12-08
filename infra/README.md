# Go Insurance - AWS Infrastructure

Secure, low-cost deployment using AWS CDK with DynamoDB, with automated CI/CD via GitHub Actions.

## Architecture

```
┌─────────────────┐     ┌──────────────────┐
│   CloudFront    │     │    App Runner    │
│   (Frontend)    │     │      (API)       │
└────────┬────────┘     └────────┬─────────┘
         │                       │
         ▼                       ▼
┌─────────────────┐     ┌──────────────────┐
│    S3 Bucket    │     │    DynamoDB      │
│  (Static Site)  │     │  (Pay-per-use)   │
└─────────────────┘     └──────────────────┘
```

## Estimated Cost: ~$5-15/mo

| Service | Cost |
|---------|------|
| App Runner (0.25 vCPU) | ~$5-12/mo |
| DynamoDB (on-demand) | ~$0-2/mo |
| CloudFront | ~$1/mo |
| S3 | ~$0.50/mo |
| ECR | ~$0.50/mo |

## Prerequisites

1. **AWS CLI** configured with credentials
2. **Node.js** 18+
3. **Docker** for building images (for manual deployment)

## CI/CD with GitHub Actions (Recommended)

The easiest way to deploy is via GitHub Actions. Push to `main` triggers automatic deployment.

### One-Time Setup

1. **Bootstrap CDK** (if not already done):
   ```bash
   npx cdk bootstrap aws://<ACCOUNT_ID>/eu-west-2
   ```

2. **Initial deployment** (creates IAM role for GitHub):
   ```bash
   cd infra
   npm install
   npx cdk deploy -c skipAppRunner=true -c githubOrg=YOUR_GITHUB_ORG -c githubRepo=YOUR_REPO_NAME
   ```

3. **Configure GitHub Secrets** (Settings → Secrets and variables → Actions):
   - `AWS_ACCOUNT_ID`: Your AWS account ID
   - `API_KEY`: Your API key for the frontend (generate with `openssl rand -hex 32`)

4. **Create GitHub Environment**:
   - Go to Settings → Environments → New environment
   - Name it `production`

5. **Update API Key in AWS**:
   - Go to AWS Secrets Manager
   - Find `go-insurance/api-secrets`
   - Update `API_KEY` to match your GitHub secret

6. **Push to trigger deployment**:
   ```bash
   git push origin main
   ```

GitHub Actions will automatically:
- Deploy CDK infrastructure
- Build and push Docker image to ECR
- Deploy API to App Runner
- Build and deploy frontend to S3/CloudFront

## Manual Deployment Steps

If you prefer manual deployment over GitHub Actions:

### 1. Deploy Infrastructure (First Time)

```bash
cd infra
npm install

# First deploy - skip App Runner (no image yet)
npx cdk deploy -c skipAppRunner=true
```

This creates:
- DynamoDB tables
- ECR repository
- S3 bucket + CloudFront
- Secrets Manager (update with real API key)
- IAM roles

### 2. Update API Key Secret

After first deploy:
1. Go to **AWS Secrets Manager**
2. Find `go-insurance/api-secrets`
3. Update `API_KEY` to a secure value (e.g., `openssl rand -hex 32`)

### 3. Build & Push Docker Image

```bash
# From project root
cd ..

# Get ECR login (copy from CDK output)
aws ecr get-login-password --region eu-west-2 | docker login --username AWS --password-stdin <account>.dkr.ecr.eu-west-2.amazonaws.com

# Build
docker build -t go-insurance-api .

# Tag
docker tag go-insurance-api:latest <ECR_URI>:latest

# Push
docker push <ECR_URI>:latest
```

### 4. Deploy App Runner

```bash
cd infra
npx cdk deploy
```

### 5. Seed Products

After API is running, seed the products:
```bash
# Get the API URL from CDK output
curl -X POST https://<APP_RUNNER_URL>/api/v1/products \
  -H "Content-Type: application/json" \
  -H "X-API-Key: <YOUR_API_KEY>" \
  -d '{"slug":"term-life-10","name":"Term Life 10","term_years":10,"min_coverage":50000,"max_coverage":500000,"base_rate":0.25}'
```

### 6. Deploy Frontend

```bash
cd ../frontend

# Update API URL in environment
echo "NEXT_PUBLIC_API_URL=https://<APP_RUNNER_URL>/api/v1" > .env.production
echo "NEXT_PUBLIC_API_KEY=<YOUR_API_KEY>" >> .env.production

# Build static export
npm run build

# Upload to S3
aws s3 sync out/ s3://<BUCKET_NAME> --delete

# Invalidate CloudFront cache
aws cloudfront create-invalidation --distribution-id <DIST_ID> --paths "/*"
```

## Security Features

- **HTTPS only** - CloudFront and App Runner enforce TLS
- **API Key auth** - All API calls require X-API-Key header
- **Rate limiting** - 100 requests/minute per IP
- **Security headers** - CSP, X-Frame-Options, etc.
- **IAM roles** - Least privilege DynamoDB access
- **Secrets Manager** - No secrets in code
- **ECR scanning** - Automatic vulnerability scanning

## Useful Commands

```bash
# View logs
aws logs tail /aws/apprunner/go-insurance-api/service --follow

# Update API (after pushing new image)
aws apprunner start-deployment --service-arn <SERVICE_ARN>

# Destroy everything
npx cdk destroy
```

## DynamoDB Tables

| Table | Purpose |
|-------|---------|
| go-insurance-products | Insurance products |
| go-insurance-quotes | Price quotes |
| go-insurance-applications | Customer applications |
| go-insurance-underwriting | UW cases |
| go-insurance-offers | Approved offers |
| go-insurance-policies | Issued policies |
| go-insurance-counters | Policy number sequences |
