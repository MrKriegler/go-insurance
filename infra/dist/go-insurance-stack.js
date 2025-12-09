"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.GoInsuranceStack = void 0;
const cdk = require("aws-cdk-lib");
const s3 = require("aws-cdk-lib/aws-s3");
const cloudfront = require("aws-cdk-lib/aws-cloudfront");
const origins = require("aws-cdk-lib/aws-cloudfront-origins");
const ecr = require("aws-cdk-lib/aws-ecr");
const apprunner = require("@aws-cdk/aws-apprunner-alpha");
const iam = require("aws-cdk-lib/aws-iam");
const secretsmanager = require("aws-cdk-lib/aws-secretsmanager");
const route53 = require("aws-cdk-lib/aws-route53");
const route53Targets = require("aws-cdk-lib/aws-route53-targets");
const acm = require("aws-cdk-lib/aws-certificatemanager");
const PROJECT_NAME = "go-insurance";
// Domain configuration
const HOSTED_ZONE_NAME = "labs.iron-pig.com";
const FRONTEND_DOMAIN = "insurance.labs.iron-pig.com";
const API_DOMAIN = "api.insurance.labs.iron-pig.com";
class GoInsuranceStack extends cdk.Stack {
    constructor(scope, id, props) {
        super(scope, id, props);
        const skipAppRunner = this.node.tryGetContext("skipAppRunner") === "true";
        const githubOrg = this.node.tryGetContext("githubOrg") || "MrKriegler";
        const githubRepo = this.node.tryGetContext("githubRepo") || "go-insurance";
        // ========================================
        // TAGGING
        // ========================================
        const tags = cdk.Tags.of(this);
        tags.add("Project", "GoInsurance");
        tags.add("ManagedBy", "cdk");
        tags.add("Repository", `github.com/${githubOrg}/${githubRepo}`);
        // ========================================
        // ROUTE 53 HOSTED ZONE (existing)
        // ========================================
        const hostedZone = route53.HostedZone.fromLookup(this, "HostedZone", {
            domainName: HOSTED_ZONE_NAME,
        });
        // ========================================
        // ACM CERTIFICATES
        // ========================================
        // Certificate for CloudFront (MUST be in us-east-1)
        // Using DnsValidatedCertificate to create certificate in us-east-1
        // Note: DnsValidatedCertificate is deprecated but still the recommended way
        // for cross-region certificates with CloudFront
        const frontendCertificate = new acm.DnsValidatedCertificate(this, "FrontendCertificate", {
            domainName: FRONTEND_DOMAIN,
            hostedZone: hostedZone,
            region: "us-east-1", // Required for CloudFront
        });
        // Note: App Runner manages its own certificate for custom domains
        // No separate ACM certificate needed - it's handled during domain association
        // ========================================
        // DYNAMODB - Tables created by Go app on startup (see dynamo/tables.go)
        // Table names: insurance_products, insurance_quotes, insurance_applications,
        //              insurance_underwriting_cases, insurance_offers, insurance_policies, insurance_counters
        // ========================================
        // ========================================
        // SECRETS (API Key)
        // ========================================
        const apiSecrets = new secretsmanager.Secret(this, "ApiSecrets", {
            secretName: `${PROJECT_NAME}/api-secrets`,
            description: "API secrets for Go Insurance",
            generateSecretString: {
                secretStringTemplate: JSON.stringify({
                    API_KEY: "change-me-in-console",
                }),
                generateStringKey: "generated",
            },
        });
        // ========================================
        // S3 + CLOUDFRONT FOR FRONTEND (must be before App Runner for CORS)
        // ========================================
        const websiteBucket = new s3.Bucket(this, "WebBucket", {
            bucketName: `${PROJECT_NAME}-web-${this.account}`,
            blockPublicAccess: s3.BlockPublicAccess.BLOCK_ALL,
            removalPolicy: cdk.RemovalPolicy.DESTROY,
            autoDeleteObjects: true,
            encryption: s3.BucketEncryption.S3_MANAGED,
        });
        // CloudFront Function to handle directory index files
        // This rewrites /path/ to /path/index.html for static hosting
        const urlRewriteFunction = new cloudfront.Function(this, "UrlRewriteFunction", {
            code: cloudfront.FunctionCode.fromInline(`
function handler(event) {
  var request = event.request;
  var uri = request.uri;

  // If URI ends with '/', append index.html
  if (uri.endsWith('/')) {
    request.uri += 'index.html';
  }
  // If URI doesn't have an extension, append /index.html
  else if (!uri.includes('.')) {
    request.uri += '/index.html';
  }

  return request;
}
      `),
            functionName: `${PROJECT_NAME}-url-rewrite`,
        });
        const distribution = new cloudfront.Distribution(this, "Distribution", {
            comment: `${PROJECT_NAME} frontend`,
            domainNames: [FRONTEND_DOMAIN],
            certificate: frontendCertificate,
            defaultBehavior: {
                origin: origins.S3BucketOrigin.withOriginAccessControl(websiteBucket),
                viewerProtocolPolicy: cloudfront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
                allowedMethods: cloudfront.AllowedMethods.ALLOW_GET_HEAD_OPTIONS,
                cachePolicy: cloudfront.CachePolicy.CACHING_OPTIMIZED,
                functionAssociations: [
                    {
                        function: urlRewriteFunction,
                        eventType: cloudfront.FunctionEventType.VIEWER_REQUEST,
                    },
                ],
            },
            defaultRootObject: "index.html",
            errorResponses: [
                { httpStatus: 404, responseHttpStatus: 200, responsePagePath: "/404.html", ttl: cdk.Duration.minutes(5) },
                { httpStatus: 403, responseHttpStatus: 200, responsePagePath: "/index.html", ttl: cdk.Duration.minutes(5) },
            ],
            priceClass: cloudfront.PriceClass.PRICE_CLASS_100,
        });
        // Route 53 A record for frontend
        new route53.ARecord(this, "FrontendARecord", {
            zone: hostedZone,
            recordName: FRONTEND_DOMAIN,
            target: route53.RecordTarget.fromAlias(new route53Targets.CloudFrontTarget(distribution)),
        });
        // ========================================
        // ECR REPOSITORY
        // ========================================
        const ecrRepo = new ecr.Repository(this, "ApiRepository", {
            repositoryName: `${PROJECT_NAME}-api`,
            removalPolicy: cdk.RemovalPolicy.DESTROY,
            emptyOnDelete: true,
            imageScanOnPush: true,
            lifecycleRules: [{ description: "Keep last 5 images", maxImageCount: 5 }],
        });
        // ========================================
        // APP RUNNER IAM ROLES
        // ========================================
        const appRunnerAccessRole = new iam.Role(this, "AppRunnerAccessRole", {
            roleName: `${PROJECT_NAME}-apprunner-access`,
            assumedBy: new iam.ServicePrincipal("build.apprunner.amazonaws.com"),
        });
        ecrRepo.grantPull(appRunnerAccessRole);
        const appRunnerInstanceRole = new iam.Role(this, "AppRunnerInstanceRole", {
            roleName: `${PROJECT_NAME}-apprunner-instance`,
            assumedBy: new iam.ServicePrincipal("tasks.apprunner.amazonaws.com"),
        });
        apiSecrets.grantRead(appRunnerInstanceRole);
        // Grant DynamoDB access for tables created by Go app (insurance_*)
        appRunnerInstanceRole.addToPolicy(new iam.PolicyStatement({
            effect: iam.Effect.ALLOW,
            actions: [
                "dynamodb:CreateTable",
                "dynamodb:DescribeTable",
                "dynamodb:GetItem",
                "dynamodb:PutItem",
                "dynamodb:UpdateItem",
                "dynamodb:DeleteItem",
                "dynamodb:Query",
                "dynamodb:Scan",
            ],
            resources: [
                `arn:aws:dynamodb:${this.region}:${this.account}:table/insurance_*`,
                `arn:aws:dynamodb:${this.region}:${this.account}:table/insurance_*/index/*`,
            ],
        }));
        // ListTables needed for health checks and table creation
        appRunnerInstanceRole.addToPolicy(new iam.PolicyStatement({
            effect: iam.Effect.ALLOW,
            actions: ["dynamodb:ListTables"],
            resources: ["*"],
        }));
        // ========================================
        // APP RUNNER SERVICE
        // ========================================
        let apiService;
        if (!skipAppRunner) {
            apiService = new apprunner.Service(this, "ApiService", {
                serviceName: `${PROJECT_NAME}-api`,
                source: apprunner.Source.fromEcr({
                    repository: ecrRepo,
                    tagOrDigest: "latest",
                    imageConfiguration: {
                        port: 8080,
                        environmentVariables: {
                            PORT: "8080",
                            ENV: "prod",
                            DB_TYPE: "dynamodb",
                            AWS_REGION: this.region,
                            LOG_LEVEL: "info",
                            LOG_FORMAT: "json",
                            RATE_LIMIT_RPM: "100",
                            ALLOWED_ORIGINS: `https://${FRONTEND_DOMAIN}`,
                        },
                        environmentSecrets: {
                            API_KEY: apprunner.Secret.fromSecretsManager(apiSecrets, "API_KEY"),
                        },
                    },
                }),
                accessRole: appRunnerAccessRole,
                instanceRole: appRunnerInstanceRole,
                cpu: apprunner.Cpu.QUARTER_VCPU,
                memory: apprunner.Memory.HALF_GB,
                autoDeploymentsEnabled: false,
                healthCheck: apprunner.HealthCheck.http({
                    path: "/health",
                    interval: cdk.Duration.seconds(10),
                    timeout: cdk.Duration.seconds(5),
                    healthyThreshold: 1,
                    unhealthyThreshold: 3,
                }),
            });
            // Note: App Runner custom domain association is done via AWS CLI in the deploy workflow
            // because CloudFormation doesn't have native support for AWS::AppRunner::CustomDomainAssociation
            // The CNAME record will be created after custom domain is associated
        }
        // ========================================
        // GITHUB ACTIONS OIDC (reuse existing provider from account)
        // ========================================
        const githubOidcProvider = iam.OpenIdConnectProvider.fromOpenIdConnectProviderArn(this, "GithubOidcProvider", `arn:aws:iam::${this.account}:oidc-provider/token.actions.githubusercontent.com`);
        const githubActionsRole = new iam.Role(this, "GithubActionsRole", {
            roleName: `github-actions-${PROJECT_NAME}`,
            assumedBy: new iam.FederatedPrincipal(githubOidcProvider.openIdConnectProviderArn, {
                StringEquals: {
                    "token.actions.githubusercontent.com:aud": "sts.amazonaws.com",
                },
                StringLike: {
                    "token.actions.githubusercontent.com:sub": `repo:${githubOrg}/${githubRepo}:*`,
                },
            }, "sts:AssumeRoleWithWebIdentity"),
            description: "Role for GitHub Actions to deploy Go Insurance",
            maxSessionDuration: cdk.Duration.hours(1),
        });
        // Grant permissions for CDK deployment
        githubActionsRole.addToPolicy(new iam.PolicyStatement({
            effect: iam.Effect.ALLOW,
            actions: ["cloudformation:*"],
            resources: [`arn:aws:cloudformation:${this.region}:${this.account}:stack/GoInsurance/*`],
        }));
        githubActionsRole.addToPolicy(new iam.PolicyStatement({
            effect: iam.Effect.ALLOW,
            actions: ["cloudformation:DescribeStacks"],
            resources: ["*"],
        }));
        // Grant ECR permissions
        ecrRepo.grantPullPush(githubActionsRole);
        githubActionsRole.addToPolicy(new iam.PolicyStatement({
            effect: iam.Effect.ALLOW,
            actions: ["ecr:GetAuthorizationToken"],
            resources: ["*"],
        }));
        // Grant S3 permissions
        websiteBucket.grantReadWrite(githubActionsRole);
        websiteBucket.grantDelete(githubActionsRole);
        // Grant CloudFront permissions
        githubActionsRole.addToPolicy(new iam.PolicyStatement({
            effect: iam.Effect.ALLOW,
            actions: ["cloudfront:CreateInvalidation", "cloudfront:GetDistribution"],
            resources: [`arn:aws:cloudfront::${this.account}:distribution/${distribution.distributionId}`],
        }));
        // Grant App Runner permissions
        githubActionsRole.addToPolicy(new iam.PolicyStatement({
            effect: iam.Effect.ALLOW,
            actions: ["apprunner:StartDeployment", "apprunner:DescribeService"],
            resources: [`arn:aws:apprunner:${this.region}:${this.account}:service/${PROJECT_NAME}-api/*`],
        }));
        // Grant SSM and Secrets Manager for CDK
        githubActionsRole.addToPolicy(new iam.PolicyStatement({
            effect: iam.Effect.ALLOW,
            actions: ["ssm:GetParameter"],
            resources: [`arn:aws:ssm:${this.region}:${this.account}:parameter/cdk-bootstrap/*`],
        }));
        // Grant CDK bootstrap permissions
        githubActionsRole.addToPolicy(new iam.PolicyStatement({
            effect: iam.Effect.ALLOW,
            actions: ["sts:AssumeRole"],
            resources: [
                `arn:aws:iam::${this.account}:role/cdk-*`,
            ],
        }));
        // Grant Route 53 permissions for custom domain management
        githubActionsRole.addToPolicy(new iam.PolicyStatement({
            effect: iam.Effect.ALLOW,
            actions: [
                "route53:GetHostedZone",
                "route53:ChangeResourceRecordSets",
                "route53:GetChange",
                "route53:ListResourceRecordSets",
            ],
            resources: [hostedZone.hostedZoneArn],
        }));
        githubActionsRole.addToPolicy(new iam.PolicyStatement({
            effect: iam.Effect.ALLOW,
            actions: ["route53:ListHostedZones", "route53:ListHostedZonesByName"],
            resources: ["*"],
        }));
        // Grant ACM permissions for certificate management
        githubActionsRole.addToPolicy(new iam.PolicyStatement({
            effect: iam.Effect.ALLOW,
            actions: [
                "acm:RequestCertificate",
                "acm:DescribeCertificate",
                "acm:DeleteCertificate",
                "acm:ListCertificates",
            ],
            resources: ["*"],
        }));
        // Grant App Runner custom domain permissions
        githubActionsRole.addToPolicy(new iam.PolicyStatement({
            effect: iam.Effect.ALLOW,
            actions: [
                "apprunner:AssociateCustomDomain",
                "apprunner:DisassociateCustomDomain",
                "apprunner:DescribeCustomDomains",
            ],
            resources: [`arn:aws:apprunner:${this.region}:${this.account}:service/${PROJECT_NAME}-api/*`],
        }));
        // ========================================
        // OUTPUTS
        // ========================================
        new cdk.CfnOutput(this, "EcrRepoUri", {
            value: ecrRepo.repositoryUri,
            description: "ECR Repository URI",
        });
        new cdk.CfnOutput(this, "EcrLoginCommand", {
            value: `aws ecr get-login-password --region ${this.region} | docker login --username AWS --password-stdin ${this.account}.dkr.ecr.${this.region}.amazonaws.com`,
            description: "ECR login command",
        });
        if (apiService) {
            new cdk.CfnOutput(this, "ApiUrl", {
                value: `https://${API_DOMAIN}`,
                description: "API URL (custom domain)",
            });
            new cdk.CfnOutput(this, "ApiServiceUrl", {
                value: apiService.serviceUrl,
                description: "API URL (App Runner default)",
            });
            new cdk.CfnOutput(this, "ApiServiceArn", {
                value: apiService.serviceArn,
                description: "App Runner Service ARN",
            });
        }
        new cdk.CfnOutput(this, "FrontendUrl", {
            value: `https://${FRONTEND_DOMAIN}`,
            description: "Frontend URL (custom domain)",
        });
        new cdk.CfnOutput(this, "FrontendCloudFrontUrl", {
            value: `https://${distribution.distributionDomainName}`,
            description: "Frontend URL (CloudFront default)",
        });
        new cdk.CfnOutput(this, "WebBucketName", {
            value: websiteBucket.bucketName,
            description: "S3 bucket for frontend",
        });
        new cdk.CfnOutput(this, "DistributionId", {
            value: distribution.distributionId,
            description: "CloudFront Distribution ID",
        });
        new cdk.CfnOutput(this, "SecretsArn", {
            value: apiSecrets.secretArn,
            description: "Secrets Manager ARN - update API_KEY here",
        });
        new cdk.CfnOutput(this, "GithubActionsRoleArn", {
            value: githubActionsRole.roleArn,
            description: "IAM Role ARN for GitHub Actions",
        });
    }
}
exports.GoInsuranceStack = GoInsuranceStack;
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZ28taW5zdXJhbmNlLXN0YWNrLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiLi4vc3JjL2dvLWluc3VyYW5jZS1zdGFjay50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiOzs7QUFBQSxtQ0FBbUM7QUFDbkMseUNBQXlDO0FBQ3pDLHlEQUF5RDtBQUN6RCw4REFBOEQ7QUFDOUQsMkNBQTJDO0FBQzNDLDBEQUEwRDtBQUMxRCwyQ0FBMkM7QUFDM0MsaUVBQWlFO0FBQ2pFLG1EQUFtRDtBQUNuRCxrRUFBa0U7QUFDbEUsMERBQTBEO0FBRzFELE1BQU0sWUFBWSxHQUFHLGNBQWMsQ0FBQztBQUVwQyx1QkFBdUI7QUFDdkIsTUFBTSxnQkFBZ0IsR0FBRyxtQkFBbUIsQ0FBQztBQUM3QyxNQUFNLGVBQWUsR0FBRyw2QkFBNkIsQ0FBQztBQUN0RCxNQUFNLFVBQVUsR0FBRyxpQ0FBaUMsQ0FBQztBQUVyRCxNQUFhLGdCQUFpQixTQUFRLEdBQUcsQ0FBQyxLQUFLO0lBQzdDLFlBQVksS0FBZ0IsRUFBRSxFQUFVLEVBQUUsS0FBc0I7UUFDOUQsS0FBSyxDQUFDLEtBQUssRUFBRSxFQUFFLEVBQUUsS0FBSyxDQUFDLENBQUM7UUFFeEIsTUFBTSxhQUFhLEdBQUcsSUFBSSxDQUFDLElBQUksQ0FBQyxhQUFhLENBQUMsZUFBZSxDQUFDLEtBQUssTUFBTSxDQUFDO1FBQzFFLE1BQU0sU0FBUyxHQUFHLElBQUksQ0FBQyxJQUFJLENBQUMsYUFBYSxDQUFDLFdBQVcsQ0FBQyxJQUFJLFlBQVksQ0FBQztRQUN2RSxNQUFNLFVBQVUsR0FBRyxJQUFJLENBQUMsSUFBSSxDQUFDLGFBQWEsQ0FBQyxZQUFZLENBQUMsSUFBSSxjQUFjLENBQUM7UUFFM0UsMkNBQTJDO1FBQzNDLFVBQVU7UUFDViwyQ0FBMkM7UUFDM0MsTUFBTSxJQUFJLEdBQUcsR0FBRyxDQUFDLElBQUksQ0FBQyxFQUFFLENBQUMsSUFBSSxDQUFDLENBQUM7UUFDL0IsSUFBSSxDQUFDLEdBQUcsQ0FBQyxT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LFdBQVcsRUFBRSw4QkFBOEI7U0FDNUMsQ0FBQyxDQUFDO1FBRUgsSUFBSSxHQUFHLENBQUMsU0FBUyxDQUFDLElBQUksRUFBRSx1QkFBdUIsRUFBRTtZQUMvQyxLQUFLLEVBQUUsV0FBVyxZQUFZLENBQUMsc0JBQXNCLEVBQUU7WUFDdkQsV0FBVyxFQUFFLG1DQUFtQztTQUNqRCxDQUFDLENBQUM7UUFFSCxJQUFJLEdBQUcsQ0FBQyxTQUFTLENBQUMsSUFBSSxFQUFFLGVBQWUsRUFBRTtZQUN2QyxLQUFLLEVBQUUsYUFBYSxDQUFDLFVBQVU7WUFDL0IsV0FBVyxFQUFFLHdCQUF3QjtTQUN0QyxDQUFDLENBQUM7UUFFSCxJQUFJLEdBQUcsQ0FBQyxTQUFTLENBQUMsSUFBSSxFQUFFLGdCQUFnQixFQUFFO1lBQ3hDLEtBQUssRUFBRSxZQUFZLENBQUMsY0FBYztZQUNsQyxXQUFXLEVBQUUsNEJBQTRCO1NBQzFDLENBQUMsQ0FBQztRQUVILElBQUksR0FBRyxDQUFDLFNBQVMsQ0FBQyxJQUFJLEVBQUUsWUFBWSxFQUFFO1lBQ3BDLEtBQUssRUFBRSxVQUFVLENBQUMsU0FBUztZQUMzQixXQUFXLEVBQUUsMkNBQTJDO1NBQ3pELENBQUMsQ0FBQztRQUVILElBQUksR0FBRyxDQUFDLFNBQVMsQ0FBQyxJQUFJLEVBQUUsc0JBQXNCLEVBQUU7WUFDOUMsS0FBSyxFQUFFLGlCQUFpQixDQUFDLE9BQU87WUFDaEMsV0FBVyxFQUFFLGlDQUFpQztTQUMvQyxDQUFDLENBQUM7SUFDTCxDQUFDO0NBQ0Y7QUF4WkQsNENBd1pDIiwic291cmNlc0NvbnRlbnQiOlsiaW1wb3J0ICogYXMgY2RrIGZyb20gXCJhd3MtY2RrLWxpYlwiO1xyXG5pbXBvcnQgKiBhcyBzMyBmcm9tIFwiYXdzLWNkay1saWIvYXdzLXMzXCI7XHJcbmltcG9ydCAqIGFzIGNsb3VkZnJvbnQgZnJvbSBcImF3cy1jZGstbGliL2F3cy1jbG91ZGZyb250XCI7XHJcbmltcG9ydCAqIGFzIG9yaWdpbnMgZnJvbSBcImF3cy1jZGstbGliL2F3cy1jbG91ZGZyb250LW9yaWdpbnNcIjtcclxuaW1wb3J0ICogYXMgZWNyIGZyb20gXCJhd3MtY2RrLWxpYi9hd3MtZWNyXCI7XHJcbmltcG9ydCAqIGFzIGFwcHJ1bm5lciBmcm9tIFwiQGF3cy1jZGsvYXdzLWFwcHJ1bm5lci1hbHBoYVwiO1xyXG5pbXBvcnQgKiBhcyBpYW0gZnJvbSBcImF3cy1jZGstbGliL2F3cy1pYW1cIjtcclxuaW1wb3J0ICogYXMgc2VjcmV0c21hbmFnZXIgZnJvbSBcImF3cy1jZGstbGliL2F3cy1zZWNyZXRzbWFuYWdlclwiO1xyXG5pbXBvcnQgKiBhcyByb3V0ZTUzIGZyb20gXCJhd3MtY2RrLWxpYi9hd3Mtcm91dGU1M1wiO1xyXG5pbXBvcnQgKiBhcyByb3V0ZTUzVGFyZ2V0cyBmcm9tIFwiYXdzLWNkay1saWIvYXdzLXJvdXRlNTMtdGFyZ2V0c1wiO1xyXG5pbXBvcnQgKiBhcyBhY20gZnJvbSBcImF3cy1jZGstbGliL2F3cy1jZXJ0aWZpY2F0ZW1hbmFnZXJcIjtcclxuaW1wb3J0IHsgQ29uc3RydWN0IH0gZnJvbSBcImNvbnN0cnVjdHNcIjtcclxuXHJcbmNvbnN0IFBST0pFQ1RfTkFNRSA9IFwiZ28taW5zdXJhbmNlXCI7XHJcblxyXG4vLyBEb21haW4gY29uZmlndXJhdGlvblxyXG5jb25zdCBIT1NURURfWk9ORV9OQU1FID0gXCJsYWJzLmlyb24tcGlnLmNvbVwiO1xyXG5jb25zdCBGUk9OVEVORF9ET01BSU4gPSBcImluc3VyYW5jZS5sYWJzLmlyb24tcGlnLmNvbVwiO1xyXG5jb25zdCBBUElfRE9NQUlOID0gXCJhcGkuaW5zdXJhbmNlLmxhYnMuaXJvbi1waWcuY29tXCI7XHJcblxyXG5leHBvcnQgY2xhc3MgR29JbnN1cmFuY2VTdGFjayBleHRlbmRzIGNkay5TdGFjayB7XHJcbiAgY29uc3RydWN0b3Ioc2NvcGU6IENvbnN0cnVjdCwgaWQ6IHN0cmluZywgcHJvcHM/OiBjZGsuU3RhY2tQcm9wcykge1xyXG4gICAgc3VwZXIoc2NvcGUsIGlkLCBwcm9wcyk7XHJcblxyXG4gICAgY29uc3Qgc2tpcEFwcFJ1bm5lciA9IHRoaXMubm9kZS50cnlHZXRDb250ZXh0KFwic2tpcEFwcFJ1bm5lclwiKSA9PT0gXCJ0cnVlXCI7XHJcbiAgICBjb25zdCBnaXRodWJPcmcgPSB0aGlzLm5vZGUudHJ5R2V0Q29udGV4dChcImdpdGh1Yk9yZ1wiKSB8fCBcIk1yS3JpZWdsZXJcIjtcclxuICAgIGNvbnN0IGdpdGh1YlJlcG8gPSB0aGlzLm5vZGUudHJ5R2V0Q29udGV4dChcImdpdGh1YlJlcG9cIikgfHwgXCJnby1pbnN1cmFuY2VcIjtcclxuXHJcbiAgICAvLyA9PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09XHJcbiAgICAvLyBUQUdHSU5HXHJcbiAgICAvLyA9PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09XHJcbiAgICBjb25zdCB0YWdzID0gY2RrLlRhZ3Mub2YodGhpcyk7XHJcbiAgICB0YWdzLmFkZChcIlByb2plY3RcIiwgXCJHb0luc3VyYW5jZVwiKTtcclxuICAgIHRhZ3MuYWRkKFwiTWFuYWdlZEJ5XCIsIFwiY2RrXCIpO1xyXG4gICAgdGFncy5hZGQoXCJSZXBvc2l0b3J5XCIsIGBnaXRodWIuY29tLyR7Z2l0aHViT3JnfS8ke2dpdGh1YlJlcG99YCk7XHJcblxyXG4gICAgLy8gPT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PVxyXG4gICAgLy8gUk9VVEUgNTMgSE9TVEVEIFpPTkUgKGV4aXN0aW5nKVxyXG4gICAgLy8gPT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PVxyXG4gICAgY29uc3QgaG9zdGVkWm9uZSA9IHJvdXRlNTMuSG9zdGVkWm9uZS5mcm9tTG9va3VwKHRoaXMsIFwiSG9zdGVkWm9uZVwiLCB7XHJcbiAgICAgIGRvbWFpbk5hbWU6IEhPU1RFRF9aT05FX05BTUUsXHJcbiAgICB9KTtcclxuXHJcbiAgICAvLyA9PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09XHJcbiAgICAvLyBBQ00gQ0VSVElGSUNBVEVTXHJcbiAgICAvLyA9PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09XHJcbiAgICAvLyBDZXJ0aWZpY2F0ZSBmb3IgQ2xvdWRGcm9udCAoTVVTVCBiZSBpbiB1cy1lYXN0LTEpXHJcbiAgICAvLyBVc2luZyBEbnNWYWxpZGF0ZWRDZXJ0aWZpY2F0ZSB0byBjcmVhdGUgY2VydGlmaWNhdGUgaW4gdXMtZWFzdC0xXHJcbiAgICAvLyBOb3RlOiBEbnNWYWxpZGF0ZWRDZXJ0aWZpY2F0ZSBpcyBkZXByZWNhdGVkIGJ1dCBzdGlsbCB0aGUgcmVjb21tZW5kZWQgd2F5XHJcbiAgICAvLyBmb3IgY3Jvc3MtcmVnaW9uIGNlcnRpZmljYXRlcyB3aXRoIENsb3VkRnJvbnRcclxuICAgIGNvbnN0IGZyb250ZW5kQ2VydGlmaWNhdGUgPSBuZXcgYWNtLkRuc1ZhbGlkYXRlZENlcnRpZmljYXRlKFxyXG4gICAgICB0aGlzLFxyXG4gICAgICBcIkZyb250ZW5kQ2VydGlmaWNhdGVcIixcclxuICAgICAge1xyXG4gICAgICAgIGRvbWFpbk5hbWU6IEZST05URU5EX0RPTUFJTixcclxuICAgICAgICBob3N0ZWRab25lOiBob3N0ZWRab25lLFxyXG4gICAgICAgIHJlZ2lvbjogXCJ1cy1lYXN0LTFcIiwgLy8gUmVxdWlyZWQgZm9yIENsb3VkRnJvbnRcclxuICAgICAgfVxyXG4gICAgKTtcclxuXHJcbiAgICAvLyBOb3RlOiBBcHAgUnVubmVyIG1hbmFnZXMgaXRzIG93biBjZXJ0aWZpY2F0ZSBmb3IgY3VzdG9tIGRvbWFpbnNcclxuICAgIC8vIE5vIHNlcGFyYXRlIEFDTSBjZXJ0aWZpY2F0ZSBuZWVkZWQgLSBpdCdzIGhhbmRsZWQgZHVyaW5nIGRvbWFpbiBhc3NvY2lhdGlvblxyXG5cclxuICAgIC8vID09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT1cclxuICAgIC8vIERZTkFNT0RCIC0gVGFibGVzIGNyZWF0ZWQgYnkgR28gYXBwIG9uIHN0YXJ0dXAgKHNlZSBkeW5hbW8vdGFibGVzLmdvKVxyXG4gICAgLy8gVGFibGUgbmFtZXM6IGluc3VyYW5jZV9wcm9kdWN0cywgaW5zdXJhbmNlX3F1b3RlcywgaW5zdXJhbmNlX2FwcGxpY2F0aW9ucyxcclxuICAgIC8vICAgICAgICAgICAgICBpbnN1cmFuY2VfdW5kZXJ3cml0aW5nX2Nhc2VzLCBpbnN1cmFuY2Vfb2ZmZXJzLCBpbnN1cmFuY2VfcG9saWNpZXMsIGluc3VyYW5jZV9jb3VudGVyc1xyXG4gICAgLy8gPT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PVxyXG5cclxuICAgIC8vID09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT1cclxuICAgIC8vIFNFQ1JFVFMgKEFQSSBLZXkpXHJcbiAgICAvLyA9PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09XHJcbiAgICBjb25zdCBhcGlTZWNyZXRzID0gbmV3IHNlY3JldHNtYW5hZ2VyLlNlY3JldCh0aGlzLCBcIkFwaVNlY3JldHNcIiwge1xyXG4gICAgICBzZWNyZXROYW1lOiBgJHtQUk9KRUNUX05BTUV9L2FwaS1zZWNyZXRzYCxcclxuICAgICAgZGVzY3JpcHRpb246IFwiQVBJIHNlY3JldHMgZm9yIEdvIEluc3VyYW5jZVwiLFxyXG4gICAgICBnZW5lcmF0ZVNlY3JldFN0cmluZzoge1xyXG4gICAgICAgIHNlY3JldFN0cmluZ1RlbXBsYXRlOiBKU09OLnN0cmluZ2lmeSh7XHJcbiAgICAgICAgICBBUElfS0VZOiBcImNoYW5nZS1tZS1pbi1jb25zb2xlXCIsXHJcbiAgICAgICAgfSksXHJcbiAgICAgICAgZ2VuZXJhdGVTdHJpbmdLZXk6IFwiZ2VuZXJhdGVkXCIsXHJcbiAgICAgIH0sXHJcbiAgICB9KTtcclxuXHJcbiAgICAvLyA9PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09XHJcbiAgICAvLyBTMyArIENMT1VERlJPTlQgRk9SIEZST05URU5EIChtdXN0IGJlIGJlZm9yZSBBcHAgUnVubmVyIGZvciBDT1JTKVxyXG4gICAgLy8gPT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PVxyXG4gICAgY29uc3Qgd2Vic2l0ZUJ1Y2tldCA9IG5ldyBzMy5CdWNrZXQodGhpcywgXCJXZWJCdWNrZXRcIiwge1xyXG4gICAgICBidWNrZXROYW1lOiBgJHtQUk9KRUNUX05BTUV9LXdlYi0ke3RoaXMuYWNjb3VudH1gLFxyXG4gICAgICBibG9ja1B1YmxpY0FjY2VzczogczMuQmxvY2tQdWJsaWNBY2Nlc3MuQkxPQ0tfQUxMLFxyXG4gICAgICByZW1vdmFsUG9saWN5OiBjZGsuUmVtb3ZhbFBvbGljeS5ERVNUUk9ZLFxyXG4gICAgICBhdXRvRGVsZXRlT2JqZWN0czogdHJ1ZSxcclxuICAgICAgZW5jcnlwdGlvbjogczMuQnVja2V0RW5jcnlwdGlvbi5TM19NQU5BR0VELFxyXG4gICAgfSk7XHJcblxyXG4gICAgLy8gQ2xvdWRGcm9udCBGdW5jdGlvbiB0byBoYW5kbGUgZGlyZWN0b3J5IGluZGV4IGZpbGVzXHJcbiAgICAvLyBUaGlzIHJld3JpdGVzIC9wYXRoLyB0byAvcGF0aC9pbmRleC5odG1sIGZvciBzdGF0aWMgaG9zdGluZ1xyXG4gICAgY29uc3QgdXJsUmV3cml0ZUZ1bmN0aW9uID0gbmV3IGNsb3VkZnJvbnQuRnVuY3Rpb24odGhpcywgXCJVcmxSZXdyaXRlRnVuY3Rpb25cIiwge1xyXG4gICAgICBjb2RlOiBjbG91ZGZyb250LkZ1bmN0aW9uQ29kZS5mcm9tSW5saW5lKGBcclxuZnVuY3Rpb24gaGFuZGxlcihldmVudCkge1xyXG4gIHZhciByZXF1ZXN0ID0gZXZlbnQucmVxdWVzdDtcclxuICB2YXIgdXJpID0gcmVxdWVzdC51cmk7XHJcblxyXG4gIC8vIElmIFVSSSBlbmRzIHdpdGggJy8nLCBhcHBlbmQgaW5kZXguaHRtbFxyXG4gIGlmICh1cmkuZW5kc1dpdGgoJy8nKSkge1xyXG4gICAgcmVxdWVzdC51cmkgKz0gJ2luZGV4Lmh0bWwnO1xyXG4gIH1cclxuICAvLyBJZiBVUkkgZG9lc24ndCBoYXZlIGFuIGV4dGVuc2lvbiwgYXBwZW5kIC9pbmRleC5odG1sXHJcbiAgZWxzZSBpZiAoIXVyaS5pbmNsdWRlcygnLicpKSB7XHJcbiAgICByZXF1ZXN0LnVyaSArPSAnL2luZGV4Lmh0bWwnO1xyXG4gIH1cclxuXHJcbiAgcmV0dXJuIHJlcXVlc3Q7XHJcbn1cclxuICAgICAgYCksXHJcbiAgICAgIGZ1bmN0aW9uTmFtZTogYCR7UFJPSkVDVF9OQU1FfS11cmwtcmV3cml0ZWAsXHJcbiAgICB9KTtcclxuXHJcbiAgICBjb25zdCBkaXN0cmlidXRpb24gPSBuZXcgY2xvdWRmcm9udC5EaXN0cmlidXRpb24odGhpcywgXCJEaXN0cmlidXRpb25cIiwge1xyXG4gICAgICBjb21tZW50OiBgJHtQUk9KRUNUX05BTUV9IGZyb250ZW5kYCxcclxuICAgICAgZG9tYWluTmFtZXM6IFtGUk9OVEVORF9ET01BSU5dLFxyXG4gICAgICBjZXJ0aWZpY2F0ZTogZnJvbnRlbmRDZXJ0aWZpY2F0ZSxcclxuICAgICAgZGVmYXVsdEJlaGF2aW9yOiB7XHJcbiAgICAgICAgb3JpZ2luOiBvcmlnaW5zLlMzQnVja2V0T3JpZ2luLndpdGhPcmlnaW5BY2Nlc3NDb250cm9sKHdlYnNpdGVCdWNrZXQpLFxyXG4gICAgICAgIHZpZXdlclByb3RvY29sUG9saWN5OiBjbG91ZGZyb250LlZpZXdlclByb3RvY29sUG9saWN5LlJFRElSRUNUX1RPX0hUVFBTLFxyXG4gICAgICAgIGFsbG93ZWRNZXRob2RzOiBjbG91ZGZyb250LkFsbG93ZWRNZXRob2RzLkFMTE9XX0dFVF9IRUFEX09QVElPTlMsXHJcbiAgICAgICAgY2FjaGVQb2xpY3k6IGNsb3VkZnJvbnQuQ2FjaGVQb2xpY3kuQ0FDSElOR19PUFRJTUlaRUQsXHJcbiAgICAgICAgZnVuY3Rpb25Bc3NvY2lhdGlvbnM6IFtcclxuICAgICAgICAgIHtcclxuICAgICAgICAgICAgZnVuY3Rpb246IHVybFJld3JpdGVGdW5jdGlvbixcclxuICAgICAgICAgICAgZXZlbnRUeXBlOiBjbG91ZGZyb250LkZ1bmN0aW9uRXZlbnRUeXBlLlZJRVdFUl9SRVFVRVNULFxyXG4gICAgICAgICAgfSxcclxuICAgICAgICBdLFxyXG4gICAgICB9LFxyXG4gICAgICBkZWZhdWx0Um9vdE9iamVjdDogXCJpbmRleC5odG1sXCIsXHJcbiAgICAgIGVycm9yUmVzcG9uc2VzOiBbXHJcbiAgICAgICAgeyBodHRwU3RhdHVzOiA0MDQsIHJlc3BvbnNlSHR0cFN0YXR1czogMjAwLCByZXNwb25zZVBhZ2VQYXRoOiBcIi80MDQuaHRtbFwiLCB0dGw6IGNkay5EdXJhdGlvbi5taW51dGVzKDUpIH0sXHJcbiAgICAgICAgeyBodHRwU3RhdHVzOiA0MDMsIHJlc3BvbnNlSHR0cFN0YXR1czogMjAwLCByZXNwb25zZVBhZ2VQYXRoOiBcIi9pbmRleC5odG1sXCIsIHR0bDogY2RrLkR1cmF0aW9uLm1pbnV0ZXMoNSkgfSxcclxuICAgICAgXSxcclxuICAgICAgcHJpY2VDbGFzczogY2xvdWRmcm9udC5QcmljZUNsYXNzLlBSSUNFX0NMQVNTXzEwMCxcclxuICAgIH0pO1xyXG5cclxuICAgIC8vIFJvdXRlIDUzIEEgcmVjb3JkIGZvciBmcm9udGVuZFxyXG4gICAgbmV3IHJvdXRlNTMuQVJlY29yZCh0aGlzLCBcIkZyb250ZW5kQVJlY29yZFwiLCB7XHJcbiAgICAgIHpvbmU6IGhvc3RlZFpvbmUsXHJcbiAgICAgIHJlY29yZE5hbWU6IEZST05URU5EX0RPTUFJTixcclxuICAgICAgdGFyZ2V0OiByb3V0ZTUzLlJlY29yZFRhcmdldC5mcm9tQWxpYXMoXHJcbiAgICAgICAgbmV3IHJvdXRlNTNUYXJnZXRzLkNsb3VkRnJvbnRUYXJnZXQoZGlzdHJpYnV0aW9uKVxyXG4gICAgICApLFxyXG4gICAgfSk7XHJcblxyXG4gICAgLy8gPT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PVxyXG4gICAgLy8gRUNSIFJFUE9TSVRPUllcclxuICAgIC8vID09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT1cclxuICAgIGNvbnN0IGVjclJlcG8gPSBuZXcgZWNyLlJlcG9zaXRvcnkodGhpcywgXCJBcGlSZXBvc2l0b3J5XCIsIHtcclxuICAgICAgcmVwb3NpdG9yeU5hbWU6IGAke1BST0pFQ1RfTkFNRX0tYXBpYCxcclxuICAgICAgcmVtb3ZhbFBvbGljeTogY2RrLlJlbW92YWxQb2xpY3kuREVTVFJPWSxcclxuICAgICAgZW1wdHlPbkRlbGV0ZTogdHJ1ZSxcclxuICAgICAgaW1hZ2VTY2FuT25QdXNoOiB0cnVlLFxyXG4gICAgICBsaWZlY3ljbGVSdWxlczogW3sgZGVzY3JpcHRpb246IFwiS2VlcCBsYXN0IDUgaW1hZ2VzXCIsIG1heEltYWdlQ291bnQ6IDUgfV0sXHJcbiAgICB9KTtcclxuXHJcbiAgICAvLyA9PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09XHJcbiAgICAvLyBBUFAgUlVOTkVSIElBTSBST0xFU1xyXG4gICAgLy8gPT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PVxyXG4gICAgY29uc3QgYXBwUnVubmVyQWNjZXNzUm9sZSA9IG5ldyBpYW0uUm9sZSh0aGlzLCBcIkFwcFJ1bm5lckFjY2Vzc1JvbGVcIiwge1xyXG4gICAgICByb2xlTmFtZTogYCR7UFJPSkVDVF9OQU1FfS1hcHBydW5uZXItYWNjZXNzYCxcclxuICAgICAgYXNzdW1lZEJ5OiBuZXcgaWFtLlNlcnZpY2VQcmluY2lwYWwoXCJidWlsZC5hcHBydW5uZXIuYW1hem9uYXdzLmNvbVwiKSxcclxuICAgIH0pO1xyXG4gICAgZWNyUmVwby5ncmFudFB1bGwoYXBwUnVubmVyQWNjZXNzUm9sZSk7XHJcblxyXG4gICAgY29uc3QgYXBwUnVubmVySW5zdGFuY2VSb2xlID0gbmV3IGlhbS5Sb2xlKHRoaXMsIFwiQXBwUnVubmVySW5zdGFuY2VSb2xlXCIsIHtcclxuICAgICAgcm9sZU5hbWU6IGAke1BST0pFQ1RfTkFNRX0tYXBwcnVubmVyLWluc3RhbmNlYCxcclxuICAgICAgYXNzdW1lZEJ5OiBuZXcgaWFtLlNlcnZpY2VQcmluY2lwYWwoXCJ0YXNrcy5hcHBydW5uZXIuYW1hem9uYXdzLmNvbVwiKSxcclxuICAgIH0pO1xyXG4gICAgYXBpU2VjcmV0cy5ncmFudFJlYWQoYXBwUnVubmVySW5zdGFuY2VSb2xlKTtcclxuXHJcbiAgICAvLyBHcmFudCBEeW5hbW9EQiBhY2Nlc3MgZm9yIHRhYmxlcyBjcmVhdGVkIGJ5IEdvIGFwcCAoaW5zdXJhbmNlXyopXHJcbiAgICBhcHBSdW5uZXJJbnN0YW5jZVJvbGUuYWRkVG9Qb2xpY3kobmV3IGlhbS5Qb2xpY3lTdGF0ZW1lbnQoe1xyXG4gICAgICBlZmZlY3Q6IGlhbS5FZmZlY3QuQUxMT1csXHJcbiAgICAgIGFjdGlvbnM6IFtcclxuICAgICAgICBcImR5bmFtb2RiOkNyZWF0ZVRhYmxlXCIsXHJcbiAgICAgICAgXCJkeW5hbW9kYjpEZXNjcmliZVRhYmxlXCIsXHJcbiAgICAgICAgXCJkeW5hbW9kYjpHZXRJdGVtXCIsXHJcbiAgICAgICAgXCJkeW5hbW9kYjpQdXRJdGVtXCIsXHJcbiAgICAgICAgXCJkeW5hbW9kYjpVcGRhdGVJdGVtXCIsXHJcbiAgICAgICAgXCJkeW5hbW9kYjpEZWxldGVJdGVtXCIsXHJcbiAgICAgICAgXCJkeW5hbW9kYjpRdWVyeVwiLFxyXG4gICAgICAgIFwiZHluYW1vZGI6U2NhblwiLFxyXG4gICAgICBdLFxyXG4gICAgICByZXNvdXJjZXM6IFtcclxuICAgICAgICBgYXJuOmF3czpkeW5hbW9kYjoke3RoaXMucmVnaW9ufToke3RoaXMuYWNjb3VudH06dGFibGUvaW5zdXJhbmNlXypgLFxyXG4gICAgICAgIGBhcm46YXdzOmR5bmFtb2RiOiR7dGhpcy5yZWdpb259OiR7dGhpcy5hY2NvdW50fTp0YWJsZS9pbnN1cmFuY2VfKi9pbmRleC8qYCxcclxuICAgICAgXSxcclxuICAgIH0pKTtcclxuICAgIC8vIExpc3RUYWJsZXMgbmVlZGVkIGZvciBoZWFsdGggY2hlY2tzIGFuZCB0YWJsZSBjcmVhdGlvblxyXG4gICAgYXBwUnVubmVySW5zdGFuY2VSb2xlLmFkZFRvUG9saWN5KG5ldyBpYW0uUG9saWN5U3RhdGVtZW50KHtcclxuICAgICAgZWZmZWN0OiBpYW0uRWZmZWN0LkFMTE9XLFxyXG4gICAgICBhY3Rpb25zOiBbXCJkeW5hbW9kYjpMaXN0VGFibGVzXCJdLFxyXG4gICAgICByZXNvdXJjZXM6IFtcIipcIl0sXHJcbiAgICB9KSk7XHJcblxyXG4gICAgLy8gPT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PVxyXG4gICAgLy8gQVBQIFJVTk5FUiBTRVJWSUNFXHJcbiAgICAvLyA9PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09XHJcbiAgICBsZXQgYXBpU2VydmljZTogYXBwcnVubmVyLlNlcnZpY2UgfCB1bmRlZmluZWQ7XHJcbiAgICBpZiAoIXNraXBBcHBSdW5uZXIpIHtcclxuICAgICAgYXBpU2VydmljZSA9IG5ldyBhcHBydW5uZXIuU2VydmljZSh0aGlzLCBcIkFwaVNlcnZpY2VcIiwge1xyXG4gICAgICAgIHNlcnZpY2VOYW1lOiBgJHtQUk9KRUNUX05BTUV9LWFwaWAsXHJcbiAgICAgICAgc291cmNlOiBhcHBydW5uZXIuU291cmNlLmZyb21FY3Ioe1xyXG4gICAgICAgICAgcmVwb3NpdG9yeTogZWNyUmVwbyxcclxuICAgICAgICAgIHRhZ09yRGlnZXN0OiBcImxhdGVzdFwiLFxyXG4gICAgICAgICAgaW1hZ2VDb25maWd1cmF0aW9uOiB7XHJcbiAgICAgICAgICAgIHBvcnQ6IDgwODAsXHJcbiAgICAgICAgICAgIGVudmlyb25tZW50VmFyaWFibGVzOiB7XHJcbiAgICAgICAgICAgICAgUE9SVDogXCI4MDgwXCIsXHJcbiAgICAgICAgICAgICAgRU5WOiBcInByb2RcIixcclxuICAgICAgICAgICAgICBEQl9UWVBFOiBcImR5bmFtb2RiXCIsXHJcbiAgICAgICAgICAgICAgQVdTX1JFR0lPTjogdGhpcy5yZWdpb24sXHJcbiAgICAgICAgICAgICAgTE9HX0xFVkVMOiBcImluZm9cIixcclxuICAgICAgICAgICAgICBMT0dfRk9STUFUOiBcImpzb25cIixcclxuICAgICAgICAgICAgICBSQVRFX0xJTUlUX1JQTTogXCIxMDBcIixcclxuICAgICAgICAgICAgICBBTExPV0VEX09SSUdJTlM6IGBodHRwczovLyR7RlJPTlRFTkRfRE9NQUlOfWAsXHJcbiAgICAgICAgICAgIH0sXHJcbiAgICAgICAgICAgIGVudmlyb25tZW50U2VjcmV0czoge1xyXG4gICAgICAgICAgICAgIEFQSV9LRVk6IGFwcHJ1bm5lci5TZWNyZXQuZnJvbVNlY3JldHNNYW5hZ2VyKGFwaVNlY3JldHMsIFwiQVBJX0tFWVwiKSxcclxuICAgICAgICAgICAgfSxcclxuICAgICAgICAgIH0sXHJcbiAgICAgICAgfSksXHJcbiAgICAgICAgYWNjZXNzUm9sZTogYXBwUnVubmVyQWNjZXNzUm9sZSxcclxuICAgICAgICBpbnN0YW5jZVJvbGU6IGFwcFJ1bm5lckluc3RhbmNlUm9sZSxcclxuICAgICAgICBjcHU6IGFwcHJ1bm5lci5DcHUuUVVBUlRFUl9WQ1BVLFxyXG4gICAgICAgIG1lbW9yeTogYXBwcnVubmVyLk1lbW9yeS5IQUxGX0dCLFxyXG4gICAgICAgIGF1dG9EZXBsb3ltZW50c0VuYWJsZWQ6IGZhbHNlLFxyXG4gICAgICAgIGhlYWx0aENoZWNrOiBhcHBydW5uZXIuSGVhbHRoQ2hlY2suaHR0cCh7XHJcbiAgICAgICAgICBwYXRoOiBcIi9oZWFsdGhcIixcclxuICAgICAgICAgIGludGVydmFsOiBjZGsuRHVyYXRpb24uc2Vjb25kcygxMCksXHJcbiAgICAgICAgICB0aW1lb3V0OiBjZGsuRHVyYXRpb24uc2Vjb25kcyg1KSxcclxuICAgICAgICAgIGhlYWx0aHlUaHJlc2hvbGQ6IDEsXHJcbiAgICAgICAgICB1bmhlYWx0aHlUaHJlc2hvbGQ6IDMsXHJcbiAgICAgICAgfSksXHJcbiAgICAgIH0pO1xyXG5cclxuICAgICAgLy8gTm90ZTogQXBwIFJ1bm5lciBjdXN0b20gZG9tYWluIGFzc29jaWF0aW9uIGlzIGRvbmUgdmlhIEFXUyBDTEkgaW4gdGhlIGRlcGxveSB3b3JrZmxvd1xyXG4gICAgICAvLyBiZWNhdXNlIENsb3VkRm9ybWF0aW9uIGRvZXNuJ3QgaGF2ZSBuYXRpdmUgc3VwcG9ydCBmb3IgQVdTOjpBcHBSdW5uZXI6OkN1c3RvbURvbWFpbkFzc29jaWF0aW9uXHJcbiAgICAgIC8vIFRoZSBDTkFNRSByZWNvcmQgd2lsbCBiZSBjcmVhdGVkIGFmdGVyIGN1c3RvbSBkb21haW4gaXMgYXNzb2NpYXRlZFxyXG4gICAgfVxyXG5cclxuICAgIC8vID09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT1cclxuICAgIC8vIEdJVEhVQiBBQ1RJT05TIE9JREMgKHJldXNlIGV4aXN0aW5nIHByb3ZpZGVyIGZyb20gYWNjb3VudClcclxuICAgIC8vID09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT1cclxuICAgIGNvbnN0IGdpdGh1Yk9pZGNQcm92aWRlciA9IGlhbS5PcGVuSWRDb25uZWN0UHJvdmlkZXIuZnJvbU9wZW5JZENvbm5lY3RQcm92aWRlckFybihcclxuICAgICAgdGhpcyxcclxuICAgICAgXCJHaXRodWJPaWRjUHJvdmlkZXJcIixcclxuICAgICAgYGFybjphd3M6aWFtOjoke3RoaXMuYWNjb3VudH06b2lkYy1wcm92aWRlci90b2tlbi5hY3Rpb25zLmdpdGh1YnVzZXJjb250ZW50LmNvbWBcclxuICAgICk7XHJcblxyXG4gICAgY29uc3QgZ2l0aHViQWN0aW9uc1JvbGUgPSBuZXcgaWFtLlJvbGUodGhpcywgXCJHaXRodWJBY3Rpb25zUm9sZVwiLCB7XHJcbiAgICAgIHJvbGVOYW1lOiBgZ2l0aHViLWFjdGlvbnMtJHtQUk9KRUNUX05BTUV9YCxcclxuICAgICAgYXNzdW1lZEJ5OiBuZXcgaWFtLkZlZGVyYXRlZFByaW5jaXBhbChcclxuICAgICAgICBnaXRodWJPaWRjUHJvdmlkZXIub3BlbklkQ29ubmVjdFByb3ZpZGVyQXJuLFxyXG4gICAgICAgIHtcclxuICAgICAgICAgIFN0cmluZ0VxdWFsczoge1xyXG4gICAgICAgICAgICBcInRva2VuLmFjdGlvbnMuZ2l0aHVidXNlcmNvbnRlbnQuY29tOmF1ZFwiOiBcInN0cy5hbWF6b25hd3MuY29tXCIsXHJcbiAgICAgICAgICB9LFxyXG4gICAgICAgICAgU3RyaW5nTGlrZToge1xyXG4gICAgICAgICAgICBcInRva2VuLmFjdGlvbnMuZ2l0aHVidXNlcmNvbnRlbnQuY29tOnN1YlwiOiBgcmVwbzoke2dpdGh1Yk9yZ30vJHtnaXRodWJSZXBvfToqYCxcclxuICAgICAgICAgIH0sXHJcbiAgICAgICAgfSxcclxuICAgICAgICBcInN0czpBc3N1bWVSb2xlV2l0aFdlYklkZW50aXR5XCJcclxuICAgICAgKSxcclxuICAgICAgZGVzY3JpcHRpb246IFwiUm9sZSBmb3IgR2l0SHViIEFjdGlvbnMgdG8gZGVwbG95IEdvIEluc3VyYW5jZVwiLFxyXG4gICAgICBtYXhTZXNzaW9uRHVyYXRpb246IGNkay5EdXJhdGlvbi5ob3VycygxKSxcclxuICAgIH0pO1xyXG5cclxuICAgIC8vIEdyYW50IHBlcm1pc3Npb25zIGZvciBDREsgZGVwbG95bWVudFxyXG4gICAgZ2l0aHViQWN0aW9uc1JvbGUuYWRkVG9Qb2xpY3kobmV3IGlhbS5Qb2xpY3lTdGF0ZW1lbnQoe1xyXG4gICAgICBlZmZlY3Q6IGlhbS5FZmZlY3QuQUxMT1csXHJcbiAgICAgIGFjdGlvbnM6IFtcImNsb3VkZm9ybWF0aW9uOipcIl0sXHJcbiAgICAgIHJlc291cmNlczogW2Bhcm46YXdzOmNsb3VkZm9ybWF0aW9uOiR7dGhpcy5yZWdpb259OiR7dGhpcy5hY2NvdW50fTpzdGFjay9Hb0luc3VyYW5jZS8qYF0sXHJcbiAgICB9KSk7XHJcbiAgICBnaXRodWJBY3Rpb25zUm9sZS5hZGRUb1BvbGljeShuZXcgaWFtLlBvbGljeVN0YXRlbWVudCh7XHJcbiAgICAgIGVmZmVjdDogaWFtLkVmZmVjdC5BTExPVyxcclxuICAgICAgYWN0aW9uczogW1wiY2xvdWRmb3JtYXRpb246RGVzY3JpYmVTdGFja3NcIl0sXHJcbiAgICAgIHJlc291cmNlczogW1wiKlwiXSxcclxuICAgIH0pKTtcclxuXHJcbiAgICAvLyBHcmFudCBFQ1IgcGVybWlzc2lvbnNcclxuICAgIGVjclJlcG8uZ3JhbnRQdWxsUHVzaChnaXRodWJBY3Rpb25zUm9sZSk7XHJcbiAgICBnaXRodWJBY3Rpb25zUm9sZS5hZGRUb1BvbGljeShuZXcgaWFtLlBvbGljeVN0YXRlbWVudCh7XHJcbiAgICAgIGVmZmVjdDogaWFtLkVmZmVjdC5BTExPVyxcclxuICAgICAgYWN0aW9uczogW1wiZWNyOkdldEF1dGhvcml6YXRpb25Ub2tlblwiXSxcclxuICAgICAgcmVzb3VyY2VzOiBbXCIqXCJdLFxyXG4gICAgfSkpO1xyXG5cclxuICAgIC8vIEdyYW50IFMzIHBlcm1pc3Npb25zXHJcbiAgICB3ZWJzaXRlQnVja2V0LmdyYW50UmVhZFdyaXRlKGdpdGh1YkFjdGlvbnNSb2xlKTtcclxuICAgIHdlYnNpdGVCdWNrZXQuZ3JhbnREZWxldGUoZ2l0aHViQWN0aW9uc1JvbGUpO1xyXG5cclxuICAgIC8vIEdyYW50IENsb3VkRnJvbnQgcGVybWlzc2lvbnNcclxuICAgIGdpdGh1YkFjdGlvbnNSb2xlLmFkZFRvUG9saWN5KG5ldyBpYW0uUG9saWN5U3RhdGVtZW50KHtcclxuICAgICAgZWZmZWN0OiBpYW0uRWZmZWN0LkFMTE9XLFxyXG4gICAgICBhY3Rpb25zOiBbXCJjbG91ZGZyb250OkNyZWF0ZUludmFsaWRhdGlvblwiLCBcImNsb3VkZnJvbnQ6R2V0RGlzdHJpYnV0aW9uXCJdLFxyXG4gICAgICByZXNvdXJjZXM6IFtgYXJuOmF3czpjbG91ZGZyb250Ojoke3RoaXMuYWNjb3VudH06ZGlzdHJpYnV0aW9uLyR7ZGlzdHJpYnV0aW9uLmRpc3RyaWJ1dGlvbklkfWBdLFxyXG4gICAgfSkpO1xyXG5cclxuICAgIC8vIEdyYW50IEFwcCBSdW5uZXIgcGVybWlzc2lvbnNcclxuICAgIGdpdGh1YkFjdGlvbnNSb2xlLmFkZFRvUG9saWN5KG5ldyBpYW0uUG9saWN5U3RhdGVtZW50KHtcclxuICAgICAgZWZmZWN0OiBpYW0uRWZmZWN0LkFMTE9XLFxyXG4gICAgICBhY3Rpb25zOiBbXCJhcHBydW5uZXI6U3RhcnREZXBsb3ltZW50XCIsIFwiYXBwcnVubmVyOkRlc2NyaWJlU2VydmljZVwiXSxcclxuICAgICAgcmVzb3VyY2VzOiBbYGFybjphd3M6YXBwcnVubmVyOiR7dGhpcy5yZWdpb259OiR7dGhpcy5hY2NvdW50fTpzZXJ2aWNlLyR7UFJPSkVDVF9OQU1FfS1hcGkvKmBdLFxyXG4gICAgfSkpO1xyXG5cclxuICAgIC8vIEdyYW50IFNTTSBhbmQgU2VjcmV0cyBNYW5hZ2VyIGZvciBDREtcclxuICAgIGdpdGh1YkFjdGlvbnNSb2xlLmFkZFRvUG9saWN5KG5ldyBpYW0uUG9saWN5U3RhdGVtZW50KHtcclxuICAgICAgZWZmZWN0OiBpYW0uRWZmZWN0LkFMTE9XLFxyXG4gICAgICBhY3Rpb25zOiBbXCJzc206R2V0UGFyYW1ldGVyXCJdLFxyXG4gICAgICByZXNvdXJjZXM6IFtgYXJuOmF3czpzc206JHt0aGlzLnJlZ2lvbn06JHt0aGlzLmFjY291bnR9OnBhcmFtZXRlci9jZGstYm9vdHN0cmFwLypgXSxcclxuICAgIH0pKTtcclxuXHJcbiAgICAvLyBHcmFudCBDREsgYm9vdHN0cmFwIHBlcm1pc3Npb25zXHJcbiAgICBnaXRodWJBY3Rpb25zUm9sZS5hZGRUb1BvbGljeShuZXcgaWFtLlBvbGljeVN0YXRlbWVudCh7XHJcbiAgICAgIGVmZmVjdDogaWFtLkVmZmVjdC5BTExPVyxcclxuICAgICAgYWN0aW9uczogW1wic3RzOkFzc3VtZVJvbGVcIl0sXHJcbiAgICAgIHJlc291cmNlczogW1xyXG4gICAgICAgIGBhcm46YXdzOmlhbTo6JHt0aGlzLmFjY291bnR9OnJvbGUvY2RrLSpgLFxyXG4gICAgICBdLFxyXG4gICAgfSkpO1xyXG5cclxuICAgIC8vIEdyYW50IFJvdXRlIDUzIHBlcm1pc3Npb25zIGZvciBjdXN0b20gZG9tYWluIG1hbmFnZW1lbnRcclxuICAgIGdpdGh1YkFjdGlvbnNSb2xlLmFkZFRvUG9saWN5KG5ldyBpYW0uUG9saWN5U3RhdGVtZW50KHtcclxuICAgICAgZWZmZWN0OiBpYW0uRWZmZWN0LkFMTE9XLFxyXG4gICAgICBhY3Rpb25zOiBbXHJcbiAgICAgICAgXCJyb3V0ZTUzOkdldEhvc3RlZFpvbmVcIixcclxuICAgICAgICBcInJvdXRlNTM6Q2hhbmdlUmVzb3VyY2VSZWNvcmRTZXRzXCIsXHJcbiAgICAgICAgXCJyb3V0ZTUzOkdldENoYW5nZVwiLFxyXG4gICAgICAgIFwicm91dGU1MzpMaXN0UmVzb3VyY2VSZWNvcmRTZXRzXCIsXHJcbiAgICAgIF0sXHJcbiAgICAgIHJlc291cmNlczogW2hvc3RlZFpvbmUuaG9zdGVkWm9uZUFybl0sXHJcbiAgICB9KSk7XHJcbiAgICBnaXRodWJBY3Rpb25zUm9sZS5hZGRUb1BvbGljeShuZXcgaWFtLlBvbGljeVN0YXRlbWVudCh7XHJcbiAgICAgIGVmZmVjdDogaWFtLkVmZmVjdC5BTExPVyxcclxuICAgICAgYWN0aW9uczogW1wicm91dGU1MzpMaXN0SG9zdGVkWm9uZXNcIiwgXCJyb3V0ZTUzOkxpc3RIb3N0ZWRab25lc0J5TmFtZVwiXSxcclxuICAgICAgcmVzb3VyY2VzOiBbXCIqXCJdLFxyXG4gICAgfSkpO1xyXG5cclxuICAgIC8vIEdyYW50IEFDTSBwZXJtaXNzaW9ucyBmb3IgY2VydGlmaWNhdGUgbWFuYWdlbWVudFxyXG4gICAgZ2l0aHViQWN0aW9uc1JvbGUuYWRkVG9Qb2xpY3kobmV3IGlhbS5Qb2xpY3lTdGF0ZW1lbnQoe1xyXG4gICAgICBlZmZlY3Q6IGlhbS5FZmZlY3QuQUxMT1csXHJcbiAgICAgIGFjdGlvbnM6IFtcclxuICAgICAgICBcImFjbTpSZXF1ZXN0Q2VydGlmaWNhdGVcIixcclxuICAgICAgICBcImFjbTpEZXNjcmliZUNlcnRpZmljYXRlXCIsXHJcbiAgICAgICAgXCJhY206RGVsZXRlQ2VydGlmaWNhdGVcIixcclxuICAgICAgICBcImFjbTpMaXN0Q2VydGlmaWNhdGVzXCIsXHJcbiAgICAgIF0sXHJcbiAgICAgIHJlc291cmNlczogW1wiKlwiXSxcclxuICAgIH0pKTtcclxuXHJcbiAgICAvLyBHcmFudCBBcHAgUnVubmVyIGN1c3RvbSBkb21haW4gcGVybWlzc2lvbnNcclxuICAgIGdpdGh1YkFjdGlvbnNSb2xlLmFkZFRvUG9saWN5KG5ldyBpYW0uUG9saWN5U3RhdGVtZW50KHtcclxuICAgICAgZWZmZWN0OiBpYW0uRWZmZWN0LkFMTE9XLFxyXG4gICAgICBhY3Rpb25zOiBbXHJcbiAgICAgICAgXCJhcHBydW5uZXI6QXNzb2NpYXRlQ3VzdG9tRG9tYWluXCIsXHJcbiAgICAgICAgXCJhcHBydW5uZXI6RGlzYXNzb2NpYXRlQ3VzdG9tRG9tYWluXCIsXHJcbiAgICAgICAgXCJhcHBydW5uZXI6RGVzY3JpYmVDdXN0b21Eb21haW5zXCIsXHJcbiAgICAgIF0sXHJcbiAgICAgIHJlc291cmNlczogW2Bhcm46YXdzOmFwcHJ1bm5lcjoke3RoaXMucmVnaW9ufToke3RoaXMuYWNjb3VudH06c2VydmljZS8ke1BST0pFQ1RfTkFNRX0tYXBpLypgXSxcclxuICAgIH0pKTtcclxuXHJcbiAgICAvLyA9PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09XHJcbiAgICAvLyBPVVRQVVRTXHJcbiAgICAvLyA9PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09XHJcbiAgICBuZXcgY2RrLkNmbk91dHB1dCh0aGlzLCBcIkVjclJlcG9VcmlcIiwge1xyXG4gICAgICB2YWx1ZTogZWNyUmVwby5yZXBvc2l0b3J5VXJpLFxyXG4gICAgICBkZXNjcmlwdGlvbjogXCJFQ1IgUmVwb3NpdG9yeSBVUklcIixcclxuICAgIH0pO1xyXG5cclxuICAgIG5ldyBjZGsuQ2ZuT3V0cHV0KHRoaXMsIFwiRWNyTG9naW5Db21tYW5kXCIsIHtcclxuICAgICAgdmFsdWU6IGBhd3MgZWNyIGdldC1sb2dpbi1wYXNzd29yZCAtLXJlZ2lvbiAke3RoaXMucmVnaW9ufSB8IGRvY2tlciBsb2dpbiAtLXVzZXJuYW1lIEFXUyAtLXBhc3N3b3JkLXN0ZGluICR7dGhpcy5hY2NvdW50fS5ka3IuZWNyLiR7dGhpcy5yZWdpb259LmFtYXpvbmF3cy5jb21gLFxyXG4gICAgICBkZXNjcmlwdGlvbjogXCJFQ1IgbG9naW4gY29tbWFuZFwiLFxyXG4gICAgfSk7XHJcblxyXG4gICAgaWYgKGFwaVNlcnZpY2UpIHtcclxuICAgICAgbmV3IGNkay5DZm5PdXRwdXQodGhpcywgXCJBcGlVcmxcIiwge1xyXG4gICAgICAgIHZhbHVlOiBgaHR0cHM6Ly8ke0FQSV9ET01BSU59YCxcclxuICAgICAgICBkZXNjcmlwdGlvbjogXCJBUEkgVVJMIChjdXN0b20gZG9tYWluKVwiLFxyXG4gICAgICB9KTtcclxuXHJcbiAgICAgIG5ldyBjZGsuQ2ZuT3V0cHV0KHRoaXMsIFwiQXBpU2VydmljZVVybFwiLCB7XHJcbiAgICAgICAgdmFsdWU6IGFwaVNlcnZpY2Uuc2VydmljZVVybCxcclxuICAgICAgICBkZXNjcmlwdGlvbjogXCJBUEkgVVJMIChBcHAgUnVubmVyIGRlZmF1bHQpXCIsXHJcbiAgICAgIH0pO1xyXG5cclxuICAgICAgbmV3IGNkay5DZm5PdXRwdXQodGhpcywgXCJBcGlTZXJ2aWNlQXJuXCIsIHtcclxuICAgICAgICB2YWx1ZTogYXBpU2VydmljZS5zZXJ2aWNlQXJuLFxyXG4gICAgICAgIGRlc2NyaXB0aW9uOiBcIkFwcCBSdW5uZXIgU2VydmljZSBBUk5cIixcclxuICAgICAgfSk7XHJcbiAgICB9XHJcblxyXG4gICAgbmV3IGNkay5DZm5PdXRwdXQodGhpcywgXCJGcm9udGVuZFVybFwiLCB7XHJcbiAgICAgIHZhbHVlOiBgaHR0cHM6Ly8ke0ZST05URU5EX0RPTUFJTn1gLFxyXG4gICAgICBkZXNjcmlwdGlvbjogXCJGcm9udGVuZCBVUkwgKGN1c3RvbSBkb21haW4pXCIsXHJcbiAgICB9KTtcclxuXHJcbiAgICBuZXcgY2RrLkNmbk91dHB1dCh0aGlzLCBcIkZyb250ZW5kQ2xvdWRGcm9udFVybFwiLCB7XHJcbiAgICAgIHZhbHVlOiBgaHR0cHM6Ly8ke2Rpc3RyaWJ1dGlvbi5kaXN0cmlidXRpb25Eb21haW5OYW1lfWAsXHJcbiAgICAgIGRlc2NyaXB0aW9uOiBcIkZyb250ZW5kIFVSTCAoQ2xvdWRGcm9udCBkZWZhdWx0KVwiLFxyXG4gICAgfSk7XHJcblxyXG4gICAgbmV3IGNkay5DZm5PdXRwdXQodGhpcywgXCJXZWJCdWNrZXROYW1lXCIsIHtcclxuICAgICAgdmFsdWU6IHdlYnNpdGVCdWNrZXQuYnVja2V0TmFtZSxcclxuICAgICAgZGVzY3JpcHRpb246IFwiUzMgYnVja2V0IGZvciBmcm9udGVuZFwiLFxyXG4gICAgfSk7XHJcblxyXG4gICAgbmV3IGNkay5DZm5PdXRwdXQodGhpcywgXCJEaXN0cmlidXRpb25JZFwiLCB7XHJcbiAgICAgIHZhbHVlOiBkaXN0cmlidXRpb24uZGlzdHJpYnV0aW9uSWQsXHJcbiAgICAgIGRlc2NyaXB0aW9uOiBcIkNsb3VkRnJvbnQgRGlzdHJpYnV0aW9uIElEXCIsXHJcbiAgICB9KTtcclxuXHJcbiAgICBuZXcgY2RrLkNmbk91dHB1dCh0aGlzLCBcIlNlY3JldHNBcm5cIiwge1xyXG4gICAgICB2YWx1ZTogYXBpU2VjcmV0cy5zZWNyZXRBcm4sXHJcbiAgICAgIGRlc2NyaXB0aW9uOiBcIlNlY3JldHMgTWFuYWdlciBBUk4gLSB1cGRhdGUgQVBJX0tFWSBoZXJlXCIsXHJcbiAgICB9KTtcclxuXHJcbiAgICBuZXcgY2RrLkNmbk91dHB1dCh0aGlzLCBcIkdpdGh1YkFjdGlvbnNSb2xlQXJuXCIsIHtcclxuICAgICAgdmFsdWU6IGdpdGh1YkFjdGlvbnNSb2xlLnJvbGVBcm4sXHJcbiAgICAgIGRlc2NyaXB0aW9uOiBcIklBTSBSb2xlIEFSTiBmb3IgR2l0SHViIEFjdGlvbnNcIixcclxuICAgIH0pO1xyXG4gIH1cclxufVxyXG4iXX0=