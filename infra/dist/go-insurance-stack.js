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
        const distribution = new cloudfront.Distribution(this, "Distribution", {
            comment: `${PROJECT_NAME} frontend`,
            domainNames: [FRONTEND_DOMAIN],
            certificate: frontendCertificate,
            defaultBehavior: {
                origin: origins.S3BucketOrigin.withOriginAccessControl(websiteBucket),
                viewerProtocolPolicy: cloudfront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
                allowedMethods: cloudfront.AllowedMethods.ALLOW_GET_HEAD_OPTIONS,
                cachePolicy: cloudfront.CachePolicy.CACHING_OPTIMIZED,
            },
            defaultRootObject: "index.html",
            errorResponses: [
                { httpStatus: 404, responseHttpStatus: 200, responsePagePath: "/index.html", ttl: cdk.Duration.minutes(5) },
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoiZ28taW5zdXJhbmNlLXN0YWNrLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiLi4vc3JjL2dvLWluc3VyYW5jZS1zdGFjay50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiOzs7QUFBQSxtQ0FBbUM7QUFDbkMseUNBQXlDO0FBQ3pDLHlEQUF5RDtBQUN6RCw4REFBOEQ7QUFDOUQsMkNBQTJDO0FBQzNDLDBEQUEwRDtBQUMxRCwyQ0FBMkM7QUFDM0MsaUVBQWlFO0FBQ2pFLG1EQUFtRDtBQUNuRCxrRUFBa0U7QUFDbEUsMERBQTBEO0FBRzFELE1BQU0sWUFBWSxHQUFHLGNBQWMsQ0FBQztBQUVwQyx1QkFBdUI7QUFDdkIsTUFBTSxnQkFBZ0IsR0FBRyxtQkFBbUIsQ0FBQztBQUM3QyxNQUFNLGVBQWUsR0FBRyw2QkFBNkIsQ0FBQztBQUN0RCxNQUFNLFVBQVUsR0FBRyxpQ0FBaUMsQ0FBQztBQUVyRCxNQUFhLGdCQUFpQixTQUFRLEdBQUcsQ0FBQyxLQUFLO0lBQzdDLFlBQVksS0FBZ0IsRUFBRSxFQUFVLEVBQUUsS0FBc0I7UUFDOUQsS0FBSyxDQUFDLEtBQUssRUFBRSxFQUFFLEVBQUUsS0FBSyxDQUFDLENBQUM7UUFFeEIsTUFBTSxhQUFhLEdBQUcsSUFBSSxDQUFDLElBQUksQ0FBQyxhQUFhLENBQUMsZUFBZSxDQUFDLEtBQUssTUFBTSxDQUFDO1FBQzFFLE1BQU0sU0FBUyxHQUFHLElBQUksQ0FBQyxJQUFJLENBQUMsYUFBYSxDQUFDLFdBQVcsQ0FBQyxJQUFJLFlBQVksQ0FBQztRQUN2RSxNQUFNLFVBQVUsR0FBRyxJQUFJLENBQUMsSUFBSSxDQUFDLGFBQWEsQ0FBQyxZQUFZLENBQUMsSUFBSSxjQUFjLENBQUM7UUFFM0UsMkNBQTJDO1FBQzNDLFVBQVU7UUFDViwyQ0FBMkM7UUFDM0MsTUFBTSxJQUFJLEdBQUcsR0FBRyxDQUFDLElBQUksQ0FBQyxFQUFFLENBQUMsSUFBSSxDQUFDLENBQUM7UUFDL0IsSUFBSSxDQUFDLEdBQUcsQ0FBQyxT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QUFJLEVBQUUsZ0JBQWdCLEVBQUU7WUFDeEMsS0FBSyxFQUFFLFlBQVksQ0FBQyxjQUFjO1lBQ2xDLFdBQVcsRUFBRSw0QkFBNEI7U0FDMUMsQ0FBQyxDQUFDO1FBRUgsSUFBSSxHQUFHLENBQUMsU0FBUyxDQUFDLElBQUksRUFBRSxZQUFZLEVBQUU7WUFDcEMsS0FBSyxFQUFFLFVBQVUsQ0FBQyxTQUFTO1lBQzNCLFdBQVcsRUFBRSwyQ0FBMkM7U0FDekQsQ0FBQyxDQUFDO1FBRUgsSUFBSSxHQUFHLENBQUMsU0FBUyxDQUFDLElBQUksRUFBRSxzQkFBc0IsRUFBRTtZQUM5QyxLQUFLLEVBQUUsaUJBQWlCLENBQUMsT0FBTztZQUNoQyxXQUFXLEVBQUUsaUNBQWlDO1NBQy9DLENBQUMsQ0FBQztJQUNMLENBQUM7Q0FDRjtBQTNYRCw0Q0EyWEMiLCJzb3VyY2VzQ29udGVudCI6WyJpbXBvcnQgKiBhcyBjZGsgZnJvbSBcImF3cy1jZGstbGliXCI7XHJcbmltcG9ydCAqIGFzIHMzIGZyb20gXCJhd3MtY2RrLWxpYi9hd3MtczNcIjtcclxuaW1wb3J0ICogYXMgY2xvdWRmcm9udCBmcm9tIFwiYXdzLWNkay1saWIvYXdzLWNsb3VkZnJvbnRcIjtcclxuaW1wb3J0ICogYXMgb3JpZ2lucyBmcm9tIFwiYXdzLWNkay1saWIvYXdzLWNsb3VkZnJvbnQtb3JpZ2luc1wiO1xyXG5pbXBvcnQgKiBhcyBlY3IgZnJvbSBcImF3cy1jZGstbGliL2F3cy1lY3JcIjtcclxuaW1wb3J0ICogYXMgYXBwcnVubmVyIGZyb20gXCJAYXdzLWNkay9hd3MtYXBwcnVubmVyLWFscGhhXCI7XHJcbmltcG9ydCAqIGFzIGlhbSBmcm9tIFwiYXdzLWNkay1saWIvYXdzLWlhbVwiO1xyXG5pbXBvcnQgKiBhcyBzZWNyZXRzbWFuYWdlciBmcm9tIFwiYXdzLWNkay1saWIvYXdzLXNlY3JldHNtYW5hZ2VyXCI7XHJcbmltcG9ydCAqIGFzIHJvdXRlNTMgZnJvbSBcImF3cy1jZGstbGliL2F3cy1yb3V0ZTUzXCI7XHJcbmltcG9ydCAqIGFzIHJvdXRlNTNUYXJnZXRzIGZyb20gXCJhd3MtY2RrLWxpYi9hd3Mtcm91dGU1My10YXJnZXRzXCI7XHJcbmltcG9ydCAqIGFzIGFjbSBmcm9tIFwiYXdzLWNkay1saWIvYXdzLWNlcnRpZmljYXRlbWFuYWdlclwiO1xyXG5pbXBvcnQgeyBDb25zdHJ1Y3QgfSBmcm9tIFwiY29uc3RydWN0c1wiO1xyXG5cclxuY29uc3QgUFJPSkVDVF9OQU1FID0gXCJnby1pbnN1cmFuY2VcIjtcclxuXHJcbi8vIERvbWFpbiBjb25maWd1cmF0aW9uXHJcbmNvbnN0IEhPU1RFRF9aT05FX05BTUUgPSBcImxhYnMuaXJvbi1waWcuY29tXCI7XHJcbmNvbnN0IEZST05URU5EX0RPTUFJTiA9IFwiaW5zdXJhbmNlLmxhYnMuaXJvbi1waWcuY29tXCI7XHJcbmNvbnN0IEFQSV9ET01BSU4gPSBcImFwaS5pbnN1cmFuY2UubGFicy5pcm9uLXBpZy5jb21cIjtcclxuXHJcbmV4cG9ydCBjbGFzcyBHb0luc3VyYW5jZVN0YWNrIGV4dGVuZHMgY2RrLlN0YWNrIHtcclxuICBjb25zdHJ1Y3RvcihzY29wZTogQ29uc3RydWN0LCBpZDogc3RyaW5nLCBwcm9wcz86IGNkay5TdGFja1Byb3BzKSB7XHJcbiAgICBzdXBlcihzY29wZSwgaWQsIHByb3BzKTtcclxuXHJcbiAgICBjb25zdCBza2lwQXBwUnVubmVyID0gdGhpcy5ub2RlLnRyeUdldENvbnRleHQoXCJza2lwQXBwUnVubmVyXCIpID09PSBcInRydWVcIjtcclxuICAgIGNvbnN0IGdpdGh1Yk9yZyA9IHRoaXMubm9kZS50cnlHZXRDb250ZXh0KFwiZ2l0aHViT3JnXCIpIHx8IFwiTXJLcmllZ2xlclwiO1xyXG4gICAgY29uc3QgZ2l0aHViUmVwbyA9IHRoaXMubm9kZS50cnlHZXRDb250ZXh0KFwiZ2l0aHViUmVwb1wiKSB8fCBcImdvLWluc3VyYW5jZVwiO1xyXG5cclxuICAgIC8vID09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT1cclxuICAgIC8vIFRBR0dJTkdcclxuICAgIC8vID09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT1cclxuICAgIGNvbnN0IHRhZ3MgPSBjZGsuVGFncy5vZih0aGlzKTtcclxuICAgIHRhZ3MuYWRkKFwiUHJvamVjdFwiLCBcIkdvSW5zdXJhbmNlXCIpO1xyXG4gICAgdGFncy5hZGQoXCJNYW5hZ2VkQnlcIiwgXCJjZGtcIik7XHJcbiAgICB0YWdzLmFkZChcIlJlcG9zaXRvcnlcIiwgYGdpdGh1Yi5jb20vJHtnaXRodWJPcmd9LyR7Z2l0aHViUmVwb31gKTtcclxuXHJcbiAgICAvLyA9PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09XHJcbiAgICAvLyBST1VURSA1MyBIT1NURUQgWk9ORSAoZXhpc3RpbmcpXHJcbiAgICAvLyA9PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09XHJcbiAgICBjb25zdCBob3N0ZWRab25lID0gcm91dGU1My5Ib3N0ZWRab25lLmZyb21Mb29rdXAodGhpcywgXCJIb3N0ZWRab25lXCIsIHtcclxuICAgICAgZG9tYWluTmFtZTogSE9TVEVEX1pPTkVfTkFNRSxcclxuICAgIH0pO1xyXG5cclxuICAgIC8vID09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT1cclxuICAgIC8vIEFDTSBDRVJUSUZJQ0FURVNcclxuICAgIC8vID09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT1cclxuICAgIC8vIENlcnRpZmljYXRlIGZvciBDbG91ZEZyb250IChNVVNUIGJlIGluIHVzLWVhc3QtMSlcclxuICAgIC8vIFVzaW5nIERuc1ZhbGlkYXRlZENlcnRpZmljYXRlIHRvIGNyZWF0ZSBjZXJ0aWZpY2F0ZSBpbiB1cy1lYXN0LTFcclxuICAgIC8vIE5vdGU6IERuc1ZhbGlkYXRlZENlcnRpZmljYXRlIGlzIGRlcHJlY2F0ZWQgYnV0IHN0aWxsIHRoZSByZWNvbW1lbmRlZCB3YXlcclxuICAgIC8vIGZvciBjcm9zcy1yZWdpb24gY2VydGlmaWNhdGVzIHdpdGggQ2xvdWRGcm9udFxyXG4gICAgY29uc3QgZnJvbnRlbmRDZXJ0aWZpY2F0ZSA9IG5ldyBhY20uRG5zVmFsaWRhdGVkQ2VydGlmaWNhdGUoXHJcbiAgICAgIHRoaXMsXHJcbiAgICAgIFwiRnJvbnRlbmRDZXJ0aWZpY2F0ZVwiLFxyXG4gICAgICB7XHJcbiAgICAgICAgZG9tYWluTmFtZTogRlJPTlRFTkRfRE9NQUlOLFxyXG4gICAgICAgIGhvc3RlZFpvbmU6IGhvc3RlZFpvbmUsXHJcbiAgICAgICAgcmVnaW9uOiBcInVzLWVhc3QtMVwiLCAvLyBSZXF1aXJlZCBmb3IgQ2xvdWRGcm9udFxyXG4gICAgICB9XHJcbiAgICApO1xyXG5cclxuICAgIC8vIE5vdGU6IEFwcCBSdW5uZXIgbWFuYWdlcyBpdHMgb3duIGNlcnRpZmljYXRlIGZvciBjdXN0b20gZG9tYWluc1xyXG4gICAgLy8gTm8gc2VwYXJhdGUgQUNNIGNlcnRpZmljYXRlIG5lZWRlZCAtIGl0J3MgaGFuZGxlZCBkdXJpbmcgZG9tYWluIGFzc29jaWF0aW9uXHJcblxyXG4gICAgLy8gPT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PVxyXG4gICAgLy8gRFlOQU1PREIgLSBUYWJsZXMgY3JlYXRlZCBieSBHbyBhcHAgb24gc3RhcnR1cCAoc2VlIGR5bmFtby90YWJsZXMuZ28pXHJcbiAgICAvLyBUYWJsZSBuYW1lczogaW5zdXJhbmNlX3Byb2R1Y3RzLCBpbnN1cmFuY2VfcXVvdGVzLCBpbnN1cmFuY2VfYXBwbGljYXRpb25zLFxyXG4gICAgLy8gICAgICAgICAgICAgIGluc3VyYW5jZV91bmRlcndyaXRpbmdfY2FzZXMsIGluc3VyYW5jZV9vZmZlcnMsIGluc3VyYW5jZV9wb2xpY2llcywgaW5zdXJhbmNlX2NvdW50ZXJzXHJcbiAgICAvLyA9PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09XHJcblxyXG4gICAgLy8gPT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PVxyXG4gICAgLy8gU0VDUkVUUyAoQVBJIEtleSlcclxuICAgIC8vID09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT1cclxuICAgIGNvbnN0IGFwaVNlY3JldHMgPSBuZXcgc2VjcmV0c21hbmFnZXIuU2VjcmV0KHRoaXMsIFwiQXBpU2VjcmV0c1wiLCB7XHJcbiAgICAgIHNlY3JldE5hbWU6IGAke1BST0pFQ1RfTkFNRX0vYXBpLXNlY3JldHNgLFxyXG4gICAgICBkZXNjcmlwdGlvbjogXCJBUEkgc2VjcmV0cyBmb3IgR28gSW5zdXJhbmNlXCIsXHJcbiAgICAgIGdlbmVyYXRlU2VjcmV0U3RyaW5nOiB7XHJcbiAgICAgICAgc2VjcmV0U3RyaW5nVGVtcGxhdGU6IEpTT04uc3RyaW5naWZ5KHtcclxuICAgICAgICAgIEFQSV9LRVk6IFwiY2hhbmdlLW1lLWluLWNvbnNvbGVcIixcclxuICAgICAgICB9KSxcclxuICAgICAgICBnZW5lcmF0ZVN0cmluZ0tleTogXCJnZW5lcmF0ZWRcIixcclxuICAgICAgfSxcclxuICAgIH0pO1xyXG5cclxuICAgIC8vID09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT1cclxuICAgIC8vIFMzICsgQ0xPVURGUk9OVCBGT1IgRlJPTlRFTkQgKG11c3QgYmUgYmVmb3JlIEFwcCBSdW5uZXIgZm9yIENPUlMpXHJcbiAgICAvLyA9PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09XHJcbiAgICBjb25zdCB3ZWJzaXRlQnVja2V0ID0gbmV3IHMzLkJ1Y2tldCh0aGlzLCBcIldlYkJ1Y2tldFwiLCB7XHJcbiAgICAgIGJ1Y2tldE5hbWU6IGAke1BST0pFQ1RfTkFNRX0td2ViLSR7dGhpcy5hY2NvdW50fWAsXHJcbiAgICAgIGJsb2NrUHVibGljQWNjZXNzOiBzMy5CbG9ja1B1YmxpY0FjY2Vzcy5CTE9DS19BTEwsXHJcbiAgICAgIHJlbW92YWxQb2xpY3k6IGNkay5SZW1vdmFsUG9saWN5LkRFU1RST1ksXHJcbiAgICAgIGF1dG9EZWxldGVPYmplY3RzOiB0cnVlLFxyXG4gICAgICBlbmNyeXB0aW9uOiBzMy5CdWNrZXRFbmNyeXB0aW9uLlMzX01BTkFHRUQsXHJcbiAgICB9KTtcclxuXHJcbiAgICBjb25zdCBkaXN0cmlidXRpb24gPSBuZXcgY2xvdWRmcm9udC5EaXN0cmlidXRpb24odGhpcywgXCJEaXN0cmlidXRpb25cIiwge1xyXG4gICAgICBjb21tZW50OiBgJHtQUk9KRUNUX05BTUV9IGZyb250ZW5kYCxcclxuICAgICAgZG9tYWluTmFtZXM6IFtGUk9OVEVORF9ET01BSU5dLFxyXG4gICAgICBjZXJ0aWZpY2F0ZTogZnJvbnRlbmRDZXJ0aWZpY2F0ZSxcclxuICAgICAgZGVmYXVsdEJlaGF2aW9yOiB7XHJcbiAgICAgICAgb3JpZ2luOiBvcmlnaW5zLlMzQnVja2V0T3JpZ2luLndpdGhPcmlnaW5BY2Nlc3NDb250cm9sKHdlYnNpdGVCdWNrZXQpLFxyXG4gICAgICAgIHZpZXdlclByb3RvY29sUG9saWN5OiBjbG91ZGZyb250LlZpZXdlclByb3RvY29sUG9saWN5LlJFRElSRUNUX1RPX0hUVFBTLFxyXG4gICAgICAgIGFsbG93ZWRNZXRob2RzOiBjbG91ZGZyb250LkFsbG93ZWRNZXRob2RzLkFMTE9XX0dFVF9IRUFEX09QVElPTlMsXHJcbiAgICAgICAgY2FjaGVQb2xpY3k6IGNsb3VkZnJvbnQuQ2FjaGVQb2xpY3kuQ0FDSElOR19PUFRJTUlaRUQsXHJcbiAgICAgIH0sXHJcbiAgICAgIGRlZmF1bHRSb290T2JqZWN0OiBcImluZGV4Lmh0bWxcIixcclxuICAgICAgZXJyb3JSZXNwb25zZXM6IFtcclxuICAgICAgICB7IGh0dHBTdGF0dXM6IDQwNCwgcmVzcG9uc2VIdHRwU3RhdHVzOiAyMDAsIHJlc3BvbnNlUGFnZVBhdGg6IFwiL2luZGV4Lmh0bWxcIiwgdHRsOiBjZGsuRHVyYXRpb24ubWludXRlcyg1KSB9LFxyXG4gICAgICAgIHsgaHR0cFN0YXR1czogNDAzLCByZXNwb25zZUh0dHBTdGF0dXM6IDIwMCwgcmVzcG9uc2VQYWdlUGF0aDogXCIvaW5kZXguaHRtbFwiLCB0dGw6IGNkay5EdXJhdGlvbi5taW51dGVzKDUpIH0sXHJcbiAgICAgIF0sXHJcbiAgICAgIHByaWNlQ2xhc3M6IGNsb3VkZnJvbnQuUHJpY2VDbGFzcy5QUklDRV9DTEFTU18xMDAsXHJcbiAgICB9KTtcclxuXHJcbiAgICAvLyBSb3V0ZSA1MyBBIHJlY29yZCBmb3IgZnJvbnRlbmRcclxuICAgIG5ldyByb3V0ZTUzLkFSZWNvcmQodGhpcywgXCJGcm9udGVuZEFSZWNvcmRcIiwge1xyXG4gICAgICB6b25lOiBob3N0ZWRab25lLFxyXG4gICAgICByZWNvcmROYW1lOiBGUk9OVEVORF9ET01BSU4sXHJcbiAgICAgIHRhcmdldDogcm91dGU1My5SZWNvcmRUYXJnZXQuZnJvbUFsaWFzKFxyXG4gICAgICAgIG5ldyByb3V0ZTUzVGFyZ2V0cy5DbG91ZEZyb250VGFyZ2V0KGRpc3RyaWJ1dGlvbilcclxuICAgICAgKSxcclxuICAgIH0pO1xyXG5cclxuICAgIC8vID09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT1cclxuICAgIC8vIEVDUiBSRVBPU0lUT1JZXHJcbiAgICAvLyA9PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09XHJcbiAgICBjb25zdCBlY3JSZXBvID0gbmV3IGVjci5SZXBvc2l0b3J5KHRoaXMsIFwiQXBpUmVwb3NpdG9yeVwiLCB7XHJcbiAgICAgIHJlcG9zaXRvcnlOYW1lOiBgJHtQUk9KRUNUX05BTUV9LWFwaWAsXHJcbiAgICAgIHJlbW92YWxQb2xpY3k6IGNkay5SZW1vdmFsUG9saWN5LkRFU1RST1ksXHJcbiAgICAgIGVtcHR5T25EZWxldGU6IHRydWUsXHJcbiAgICAgIGltYWdlU2Nhbk9uUHVzaDogdHJ1ZSxcclxuICAgICAgbGlmZWN5Y2xlUnVsZXM6IFt7IGRlc2NyaXB0aW9uOiBcIktlZXAgbGFzdCA1IGltYWdlc1wiLCBtYXhJbWFnZUNvdW50OiA1IH1dLFxyXG4gICAgfSk7XHJcblxyXG4gICAgLy8gPT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PVxyXG4gICAgLy8gQVBQIFJVTk5FUiBJQU0gUk9MRVNcclxuICAgIC8vID09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT1cclxuICAgIGNvbnN0IGFwcFJ1bm5lckFjY2Vzc1JvbGUgPSBuZXcgaWFtLlJvbGUodGhpcywgXCJBcHBSdW5uZXJBY2Nlc3NSb2xlXCIsIHtcclxuICAgICAgcm9sZU5hbWU6IGAke1BST0pFQ1RfTkFNRX0tYXBwcnVubmVyLWFjY2Vzc2AsXHJcbiAgICAgIGFzc3VtZWRCeTogbmV3IGlhbS5TZXJ2aWNlUHJpbmNpcGFsKFwiYnVpbGQuYXBwcnVubmVyLmFtYXpvbmF3cy5jb21cIiksXHJcbiAgICB9KTtcclxuICAgIGVjclJlcG8uZ3JhbnRQdWxsKGFwcFJ1bm5lckFjY2Vzc1JvbGUpO1xyXG5cclxuICAgIGNvbnN0IGFwcFJ1bm5lckluc3RhbmNlUm9sZSA9IG5ldyBpYW0uUm9sZSh0aGlzLCBcIkFwcFJ1bm5lckluc3RhbmNlUm9sZVwiLCB7XHJcbiAgICAgIHJvbGVOYW1lOiBgJHtQUk9KRUNUX05BTUV9LWFwcHJ1bm5lci1pbnN0YW5jZWAsXHJcbiAgICAgIGFzc3VtZWRCeTogbmV3IGlhbS5TZXJ2aWNlUHJpbmNpcGFsKFwidGFza3MuYXBwcnVubmVyLmFtYXpvbmF3cy5jb21cIiksXHJcbiAgICB9KTtcclxuICAgIGFwaVNlY3JldHMuZ3JhbnRSZWFkKGFwcFJ1bm5lckluc3RhbmNlUm9sZSk7XHJcblxyXG4gICAgLy8gR3JhbnQgRHluYW1vREIgYWNjZXNzIGZvciB0YWJsZXMgY3JlYXRlZCBieSBHbyBhcHAgKGluc3VyYW5jZV8qKVxyXG4gICAgYXBwUnVubmVySW5zdGFuY2VSb2xlLmFkZFRvUG9saWN5KG5ldyBpYW0uUG9saWN5U3RhdGVtZW50KHtcclxuICAgICAgZWZmZWN0OiBpYW0uRWZmZWN0LkFMTE9XLFxyXG4gICAgICBhY3Rpb25zOiBbXHJcbiAgICAgICAgXCJkeW5hbW9kYjpDcmVhdGVUYWJsZVwiLFxyXG4gICAgICAgIFwiZHluYW1vZGI6RGVzY3JpYmVUYWJsZVwiLFxyXG4gICAgICAgIFwiZHluYW1vZGI6R2V0SXRlbVwiLFxyXG4gICAgICAgIFwiZHluYW1vZGI6UHV0SXRlbVwiLFxyXG4gICAgICAgIFwiZHluYW1vZGI6VXBkYXRlSXRlbVwiLFxyXG4gICAgICAgIFwiZHluYW1vZGI6RGVsZXRlSXRlbVwiLFxyXG4gICAgICAgIFwiZHluYW1vZGI6UXVlcnlcIixcclxuICAgICAgICBcImR5bmFtb2RiOlNjYW5cIixcclxuICAgICAgXSxcclxuICAgICAgcmVzb3VyY2VzOiBbXHJcbiAgICAgICAgYGFybjphd3M6ZHluYW1vZGI6JHt0aGlzLnJlZ2lvbn06JHt0aGlzLmFjY291bnR9OnRhYmxlL2luc3VyYW5jZV8qYCxcclxuICAgICAgICBgYXJuOmF3czpkeW5hbW9kYjoke3RoaXMucmVnaW9ufToke3RoaXMuYWNjb3VudH06dGFibGUvaW5zdXJhbmNlXyovaW5kZXgvKmAsXHJcbiAgICAgIF0sXHJcbiAgICB9KSk7XHJcbiAgICAvLyBMaXN0VGFibGVzIG5lZWRlZCBmb3IgaGVhbHRoIGNoZWNrcyBhbmQgdGFibGUgY3JlYXRpb25cclxuICAgIGFwcFJ1bm5lckluc3RhbmNlUm9sZS5hZGRUb1BvbGljeShuZXcgaWFtLlBvbGljeVN0YXRlbWVudCh7XHJcbiAgICAgIGVmZmVjdDogaWFtLkVmZmVjdC5BTExPVyxcclxuICAgICAgYWN0aW9uczogW1wiZHluYW1vZGI6TGlzdFRhYmxlc1wiXSxcclxuICAgICAgcmVzb3VyY2VzOiBbXCIqXCJdLFxyXG4gICAgfSkpO1xyXG5cclxuICAgIC8vID09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT1cclxuICAgIC8vIEFQUCBSVU5ORVIgU0VSVklDRVxyXG4gICAgLy8gPT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PVxyXG4gICAgbGV0IGFwaVNlcnZpY2U6IGFwcHJ1bm5lci5TZXJ2aWNlIHwgdW5kZWZpbmVkO1xyXG4gICAgaWYgKCFza2lwQXBwUnVubmVyKSB7XHJcbiAgICAgIGFwaVNlcnZpY2UgPSBuZXcgYXBwcnVubmVyLlNlcnZpY2UodGhpcywgXCJBcGlTZXJ2aWNlXCIsIHtcclxuICAgICAgICBzZXJ2aWNlTmFtZTogYCR7UFJPSkVDVF9OQU1FfS1hcGlgLFxyXG4gICAgICAgIHNvdXJjZTogYXBwcnVubmVyLlNvdXJjZS5mcm9tRWNyKHtcclxuICAgICAgICAgIHJlcG9zaXRvcnk6IGVjclJlcG8sXHJcbiAgICAgICAgICB0YWdPckRpZ2VzdDogXCJsYXRlc3RcIixcclxuICAgICAgICAgIGltYWdlQ29uZmlndXJhdGlvbjoge1xyXG4gICAgICAgICAgICBwb3J0OiA4MDgwLFxyXG4gICAgICAgICAgICBlbnZpcm9ubWVudFZhcmlhYmxlczoge1xyXG4gICAgICAgICAgICAgIFBPUlQ6IFwiODA4MFwiLFxyXG4gICAgICAgICAgICAgIEVOVjogXCJwcm9kXCIsXHJcbiAgICAgICAgICAgICAgREJfVFlQRTogXCJkeW5hbW9kYlwiLFxyXG4gICAgICAgICAgICAgIEFXU19SRUdJT046IHRoaXMucmVnaW9uLFxyXG4gICAgICAgICAgICAgIExPR19MRVZFTDogXCJpbmZvXCIsXHJcbiAgICAgICAgICAgICAgTE9HX0ZPUk1BVDogXCJqc29uXCIsXHJcbiAgICAgICAgICAgICAgUkFURV9MSU1JVF9SUE06IFwiMTAwXCIsXHJcbiAgICAgICAgICAgICAgQUxMT1dFRF9PUklHSU5TOiBgaHR0cHM6Ly8ke0ZST05URU5EX0RPTUFJTn1gLFxyXG4gICAgICAgICAgICB9LFxyXG4gICAgICAgICAgICBlbnZpcm9ubWVudFNlY3JldHM6IHtcclxuICAgICAgICAgICAgICBBUElfS0VZOiBhcHBydW5uZXIuU2VjcmV0LmZyb21TZWNyZXRzTWFuYWdlcihhcGlTZWNyZXRzLCBcIkFQSV9LRVlcIiksXHJcbiAgICAgICAgICAgIH0sXHJcbiAgICAgICAgICB9LFxyXG4gICAgICAgIH0pLFxyXG4gICAgICAgIGFjY2Vzc1JvbGU6IGFwcFJ1bm5lckFjY2Vzc1JvbGUsXHJcbiAgICAgICAgaW5zdGFuY2VSb2xlOiBhcHBSdW5uZXJJbnN0YW5jZVJvbGUsXHJcbiAgICAgICAgY3B1OiBhcHBydW5uZXIuQ3B1LlFVQVJURVJfVkNQVSxcclxuICAgICAgICBtZW1vcnk6IGFwcHJ1bm5lci5NZW1vcnkuSEFMRl9HQixcclxuICAgICAgICBhdXRvRGVwbG95bWVudHNFbmFibGVkOiBmYWxzZSxcclxuICAgICAgICBoZWFsdGhDaGVjazogYXBwcnVubmVyLkhlYWx0aENoZWNrLmh0dHAoe1xyXG4gICAgICAgICAgcGF0aDogXCIvaGVhbHRoXCIsXHJcbiAgICAgICAgICBpbnRlcnZhbDogY2RrLkR1cmF0aW9uLnNlY29uZHMoMTApLFxyXG4gICAgICAgICAgdGltZW91dDogY2RrLkR1cmF0aW9uLnNlY29uZHMoNSksXHJcbiAgICAgICAgICBoZWFsdGh5VGhyZXNob2xkOiAxLFxyXG4gICAgICAgICAgdW5oZWFsdGh5VGhyZXNob2xkOiAzLFxyXG4gICAgICAgIH0pLFxyXG4gICAgICB9KTtcclxuXHJcbiAgICAgIC8vIE5vdGU6IEFwcCBSdW5uZXIgY3VzdG9tIGRvbWFpbiBhc3NvY2lhdGlvbiBpcyBkb25lIHZpYSBBV1MgQ0xJIGluIHRoZSBkZXBsb3kgd29ya2Zsb3dcclxuICAgICAgLy8gYmVjYXVzZSBDbG91ZEZvcm1hdGlvbiBkb2Vzbid0IGhhdmUgbmF0aXZlIHN1cHBvcnQgZm9yIEFXUzo6QXBwUnVubmVyOjpDdXN0b21Eb21haW5Bc3NvY2lhdGlvblxyXG4gICAgICAvLyBUaGUgQ05BTUUgcmVjb3JkIHdpbGwgYmUgY3JlYXRlZCBhZnRlciBjdXN0b20gZG9tYWluIGlzIGFzc29jaWF0ZWRcclxuICAgIH1cclxuXHJcbiAgICAvLyA9PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09XHJcbiAgICAvLyBHSVRIVUIgQUNUSU9OUyBPSURDIChyZXVzZSBleGlzdGluZyBwcm92aWRlciBmcm9tIGFjY291bnQpXHJcbiAgICAvLyA9PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09XHJcbiAgICBjb25zdCBnaXRodWJPaWRjUHJvdmlkZXIgPSBpYW0uT3BlbklkQ29ubmVjdFByb3ZpZGVyLmZyb21PcGVuSWRDb25uZWN0UHJvdmlkZXJBcm4oXHJcbiAgICAgIHRoaXMsXHJcbiAgICAgIFwiR2l0aHViT2lkY1Byb3ZpZGVyXCIsXHJcbiAgICAgIGBhcm46YXdzOmlhbTo6JHt0aGlzLmFjY291bnR9Om9pZGMtcHJvdmlkZXIvdG9rZW4uYWN0aW9ucy5naXRodWJ1c2VyY29udGVudC5jb21gXHJcbiAgICApO1xyXG5cclxuICAgIGNvbnN0IGdpdGh1YkFjdGlvbnNSb2xlID0gbmV3IGlhbS5Sb2xlKHRoaXMsIFwiR2l0aHViQWN0aW9uc1JvbGVcIiwge1xyXG4gICAgICByb2xlTmFtZTogYGdpdGh1Yi1hY3Rpb25zLSR7UFJPSkVDVF9OQU1FfWAsXHJcbiAgICAgIGFzc3VtZWRCeTogbmV3IGlhbS5GZWRlcmF0ZWRQcmluY2lwYWwoXHJcbiAgICAgICAgZ2l0aHViT2lkY1Byb3ZpZGVyLm9wZW5JZENvbm5lY3RQcm92aWRlckFybixcclxuICAgICAgICB7XHJcbiAgICAgICAgICBTdHJpbmdFcXVhbHM6IHtcclxuICAgICAgICAgICAgXCJ0b2tlbi5hY3Rpb25zLmdpdGh1YnVzZXJjb250ZW50LmNvbTphdWRcIjogXCJzdHMuYW1hem9uYXdzLmNvbVwiLFxyXG4gICAgICAgICAgfSxcclxuICAgICAgICAgIFN0cmluZ0xpa2U6IHtcclxuICAgICAgICAgICAgXCJ0b2tlbi5hY3Rpb25zLmdpdGh1YnVzZXJjb250ZW50LmNvbTpzdWJcIjogYHJlcG86JHtnaXRodWJPcmd9LyR7Z2l0aHViUmVwb306KmAsXHJcbiAgICAgICAgICB9LFxyXG4gICAgICAgIH0sXHJcbiAgICAgICAgXCJzdHM6QXNzdW1lUm9sZVdpdGhXZWJJZGVudGl0eVwiXHJcbiAgICAgICksXHJcbiAgICAgIGRlc2NyaXB0aW9uOiBcIlJvbGUgZm9yIEdpdEh1YiBBY3Rpb25zIHRvIGRlcGxveSBHbyBJbnN1cmFuY2VcIixcclxuICAgICAgbWF4U2Vzc2lvbkR1cmF0aW9uOiBjZGsuRHVyYXRpb24uaG91cnMoMSksXHJcbiAgICB9KTtcclxuXHJcbiAgICAvLyBHcmFudCBwZXJtaXNzaW9ucyBmb3IgQ0RLIGRlcGxveW1lbnRcclxuICAgIGdpdGh1YkFjdGlvbnNSb2xlLmFkZFRvUG9saWN5KG5ldyBpYW0uUG9saWN5U3RhdGVtZW50KHtcclxuICAgICAgZWZmZWN0OiBpYW0uRWZmZWN0LkFMTE9XLFxyXG4gICAgICBhY3Rpb25zOiBbXCJjbG91ZGZvcm1hdGlvbjoqXCJdLFxyXG4gICAgICByZXNvdXJjZXM6IFtgYXJuOmF3czpjbG91ZGZvcm1hdGlvbjoke3RoaXMucmVnaW9ufToke3RoaXMuYWNjb3VudH06c3RhY2svR29JbnN1cmFuY2UvKmBdLFxyXG4gICAgfSkpO1xyXG4gICAgZ2l0aHViQWN0aW9uc1JvbGUuYWRkVG9Qb2xpY3kobmV3IGlhbS5Qb2xpY3lTdGF0ZW1lbnQoe1xyXG4gICAgICBlZmZlY3Q6IGlhbS5FZmZlY3QuQUxMT1csXHJcbiAgICAgIGFjdGlvbnM6IFtcImNsb3VkZm9ybWF0aW9uOkRlc2NyaWJlU3RhY2tzXCJdLFxyXG4gICAgICByZXNvdXJjZXM6IFtcIipcIl0sXHJcbiAgICB9KSk7XHJcblxyXG4gICAgLy8gR3JhbnQgRUNSIHBlcm1pc3Npb25zXHJcbiAgICBlY3JSZXBvLmdyYW50UHVsbFB1c2goZ2l0aHViQWN0aW9uc1JvbGUpO1xyXG4gICAgZ2l0aHViQWN0aW9uc1JvbGUuYWRkVG9Qb2xpY3kobmV3IGlhbS5Qb2xpY3lTdGF0ZW1lbnQoe1xyXG4gICAgICBlZmZlY3Q6IGlhbS5FZmZlY3QuQUxMT1csXHJcbiAgICAgIGFjdGlvbnM6IFtcImVjcjpHZXRBdXRob3JpemF0aW9uVG9rZW5cIl0sXHJcbiAgICAgIHJlc291cmNlczogW1wiKlwiXSxcclxuICAgIH0pKTtcclxuXHJcbiAgICAvLyBHcmFudCBTMyBwZXJtaXNzaW9uc1xyXG4gICAgd2Vic2l0ZUJ1Y2tldC5ncmFudFJlYWRXcml0ZShnaXRodWJBY3Rpb25zUm9sZSk7XHJcbiAgICB3ZWJzaXRlQnVja2V0LmdyYW50RGVsZXRlKGdpdGh1YkFjdGlvbnNSb2xlKTtcclxuXHJcbiAgICAvLyBHcmFudCBDbG91ZEZyb250IHBlcm1pc3Npb25zXHJcbiAgICBnaXRodWJBY3Rpb25zUm9sZS5hZGRUb1BvbGljeShuZXcgaWFtLlBvbGljeVN0YXRlbWVudCh7XHJcbiAgICAgIGVmZmVjdDogaWFtLkVmZmVjdC5BTExPVyxcclxuICAgICAgYWN0aW9uczogW1wiY2xvdWRmcm9udDpDcmVhdGVJbnZhbGlkYXRpb25cIiwgXCJjbG91ZGZyb250OkdldERpc3RyaWJ1dGlvblwiXSxcclxuICAgICAgcmVzb3VyY2VzOiBbYGFybjphd3M6Y2xvdWRmcm9udDo6JHt0aGlzLmFjY291bnR9OmRpc3RyaWJ1dGlvbi8ke2Rpc3RyaWJ1dGlvbi5kaXN0cmlidXRpb25JZH1gXSxcclxuICAgIH0pKTtcclxuXHJcbiAgICAvLyBHcmFudCBBcHAgUnVubmVyIHBlcm1pc3Npb25zXHJcbiAgICBnaXRodWJBY3Rpb25zUm9sZS5hZGRUb1BvbGljeShuZXcgaWFtLlBvbGljeVN0YXRlbWVudCh7XHJcbiAgICAgIGVmZmVjdDogaWFtLkVmZmVjdC5BTExPVyxcclxuICAgICAgYWN0aW9uczogW1wiYXBwcnVubmVyOlN0YXJ0RGVwbG95bWVudFwiLCBcImFwcHJ1bm5lcjpEZXNjcmliZVNlcnZpY2VcIl0sXHJcbiAgICAgIHJlc291cmNlczogW2Bhcm46YXdzOmFwcHJ1bm5lcjoke3RoaXMucmVnaW9ufToke3RoaXMuYWNjb3VudH06c2VydmljZS8ke1BST0pFQ1RfTkFNRX0tYXBpLypgXSxcclxuICAgIH0pKTtcclxuXHJcbiAgICAvLyBHcmFudCBTU00gYW5kIFNlY3JldHMgTWFuYWdlciBmb3IgQ0RLXHJcbiAgICBnaXRodWJBY3Rpb25zUm9sZS5hZGRUb1BvbGljeShuZXcgaWFtLlBvbGljeVN0YXRlbWVudCh7XHJcbiAgICAgIGVmZmVjdDogaWFtLkVmZmVjdC5BTExPVyxcclxuICAgICAgYWN0aW9uczogW1wic3NtOkdldFBhcmFtZXRlclwiXSxcclxuICAgICAgcmVzb3VyY2VzOiBbYGFybjphd3M6c3NtOiR7dGhpcy5yZWdpb259OiR7dGhpcy5hY2NvdW50fTpwYXJhbWV0ZXIvY2RrLWJvb3RzdHJhcC8qYF0sXHJcbiAgICB9KSk7XHJcblxyXG4gICAgLy8gR3JhbnQgQ0RLIGJvb3RzdHJhcCBwZXJtaXNzaW9uc1xyXG4gICAgZ2l0aHViQWN0aW9uc1JvbGUuYWRkVG9Qb2xpY3kobmV3IGlhbS5Qb2xpY3lTdGF0ZW1lbnQoe1xyXG4gICAgICBlZmZlY3Q6IGlhbS5FZmZlY3QuQUxMT1csXHJcbiAgICAgIGFjdGlvbnM6IFtcInN0czpBc3N1bWVSb2xlXCJdLFxyXG4gICAgICByZXNvdXJjZXM6IFtcclxuICAgICAgICBgYXJuOmF3czppYW06OiR7dGhpcy5hY2NvdW50fTpyb2xlL2Nkay0qYCxcclxuICAgICAgXSxcclxuICAgIH0pKTtcclxuXHJcbiAgICAvLyBHcmFudCBSb3V0ZSA1MyBwZXJtaXNzaW9ucyBmb3IgY3VzdG9tIGRvbWFpbiBtYW5hZ2VtZW50XHJcbiAgICBnaXRodWJBY3Rpb25zUm9sZS5hZGRUb1BvbGljeShuZXcgaWFtLlBvbGljeVN0YXRlbWVudCh7XHJcbiAgICAgIGVmZmVjdDogaWFtLkVmZmVjdC5BTExPVyxcclxuICAgICAgYWN0aW9uczogW1xyXG4gICAgICAgIFwicm91dGU1MzpHZXRIb3N0ZWRab25lXCIsXHJcbiAgICAgICAgXCJyb3V0ZTUzOkNoYW5nZVJlc291cmNlUmVjb3JkU2V0c1wiLFxyXG4gICAgICAgIFwicm91dGU1MzpHZXRDaGFuZ2VcIixcclxuICAgICAgICBcInJvdXRlNTM6TGlzdFJlc291cmNlUmVjb3JkU2V0c1wiLFxyXG4gICAgICBdLFxyXG4gICAgICByZXNvdXJjZXM6IFtob3N0ZWRab25lLmhvc3RlZFpvbmVBcm5dLFxyXG4gICAgfSkpO1xyXG4gICAgZ2l0aHViQWN0aW9uc1JvbGUuYWRkVG9Qb2xpY3kobmV3IGlhbS5Qb2xpY3lTdGF0ZW1lbnQoe1xyXG4gICAgICBlZmZlY3Q6IGlhbS5FZmZlY3QuQUxMT1csXHJcbiAgICAgIGFjdGlvbnM6IFtcInJvdXRlNTM6TGlzdEhvc3RlZFpvbmVzXCIsIFwicm91dGU1MzpMaXN0SG9zdGVkWm9uZXNCeU5hbWVcIl0sXHJcbiAgICAgIHJlc291cmNlczogW1wiKlwiXSxcclxuICAgIH0pKTtcclxuXHJcbiAgICAvLyBHcmFudCBBQ00gcGVybWlzc2lvbnMgZm9yIGNlcnRpZmljYXRlIG1hbmFnZW1lbnRcclxuICAgIGdpdGh1YkFjdGlvbnNSb2xlLmFkZFRvUG9saWN5KG5ldyBpYW0uUG9saWN5U3RhdGVtZW50KHtcclxuICAgICAgZWZmZWN0OiBpYW0uRWZmZWN0LkFMTE9XLFxyXG4gICAgICBhY3Rpb25zOiBbXHJcbiAgICAgICAgXCJhY206UmVxdWVzdENlcnRpZmljYXRlXCIsXHJcbiAgICAgICAgXCJhY206RGVzY3JpYmVDZXJ0aWZpY2F0ZVwiLFxyXG4gICAgICAgIFwiYWNtOkRlbGV0ZUNlcnRpZmljYXRlXCIsXHJcbiAgICAgICAgXCJhY206TGlzdENlcnRpZmljYXRlc1wiLFxyXG4gICAgICBdLFxyXG4gICAgICByZXNvdXJjZXM6IFtcIipcIl0sXHJcbiAgICB9KSk7XHJcblxyXG4gICAgLy8gR3JhbnQgQXBwIFJ1bm5lciBjdXN0b20gZG9tYWluIHBlcm1pc3Npb25zXHJcbiAgICBnaXRodWJBY3Rpb25zUm9sZS5hZGRUb1BvbGljeShuZXcgaWFtLlBvbGljeVN0YXRlbWVudCh7XHJcbiAgICAgIGVmZmVjdDogaWFtLkVmZmVjdC5BTExPVyxcclxuICAgICAgYWN0aW9uczogW1xyXG4gICAgICAgIFwiYXBwcnVubmVyOkFzc29jaWF0ZUN1c3RvbURvbWFpblwiLFxyXG4gICAgICAgIFwiYXBwcnVubmVyOkRpc2Fzc29jaWF0ZUN1c3RvbURvbWFpblwiLFxyXG4gICAgICAgIFwiYXBwcnVubmVyOkRlc2NyaWJlQ3VzdG9tRG9tYWluc1wiLFxyXG4gICAgICBdLFxyXG4gICAgICByZXNvdXJjZXM6IFtgYXJuOmF3czphcHBydW5uZXI6JHt0aGlzLnJlZ2lvbn06JHt0aGlzLmFjY291bnR9OnNlcnZpY2UvJHtQUk9KRUNUX05BTUV9LWFwaS8qYF0sXHJcbiAgICB9KSk7XHJcblxyXG4gICAgLy8gPT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PVxyXG4gICAgLy8gT1VUUFVUU1xyXG4gICAgLy8gPT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PVxyXG4gICAgbmV3IGNkay5DZm5PdXRwdXQodGhpcywgXCJFY3JSZXBvVXJpXCIsIHtcclxuICAgICAgdmFsdWU6IGVjclJlcG8ucmVwb3NpdG9yeVVyaSxcclxuICAgICAgZGVzY3JpcHRpb246IFwiRUNSIFJlcG9zaXRvcnkgVVJJXCIsXHJcbiAgICB9KTtcclxuXHJcbiAgICBuZXcgY2RrLkNmbk91dHB1dCh0aGlzLCBcIkVjckxvZ2luQ29tbWFuZFwiLCB7XHJcbiAgICAgIHZhbHVlOiBgYXdzIGVjciBnZXQtbG9naW4tcGFzc3dvcmQgLS1yZWdpb24gJHt0aGlzLnJlZ2lvbn0gfCBkb2NrZXIgbG9naW4gLS11c2VybmFtZSBBV1MgLS1wYXNzd29yZC1zdGRpbiAke3RoaXMuYWNjb3VudH0uZGtyLmVjci4ke3RoaXMucmVnaW9ufS5hbWF6b25hd3MuY29tYCxcclxuICAgICAgZGVzY3JpcHRpb246IFwiRUNSIGxvZ2luIGNvbW1hbmRcIixcclxuICAgIH0pO1xyXG5cclxuICAgIGlmIChhcGlTZXJ2aWNlKSB7XHJcbiAgICAgIG5ldyBjZGsuQ2ZuT3V0cHV0KHRoaXMsIFwiQXBpVXJsXCIsIHtcclxuICAgICAgICB2YWx1ZTogYGh0dHBzOi8vJHtBUElfRE9NQUlOfWAsXHJcbiAgICAgICAgZGVzY3JpcHRpb246IFwiQVBJIFVSTCAoY3VzdG9tIGRvbWFpbilcIixcclxuICAgICAgfSk7XHJcblxyXG4gICAgICBuZXcgY2RrLkNmbk91dHB1dCh0aGlzLCBcIkFwaVNlcnZpY2VVcmxcIiwge1xyXG4gICAgICAgIHZhbHVlOiBhcGlTZXJ2aWNlLnNlcnZpY2VVcmwsXHJcbiAgICAgICAgZGVzY3JpcHRpb246IFwiQVBJIFVSTCAoQXBwIFJ1bm5lciBkZWZhdWx0KVwiLFxyXG4gICAgICB9KTtcclxuXHJcbiAgICAgIG5ldyBjZGsuQ2ZuT3V0cHV0KHRoaXMsIFwiQXBpU2VydmljZUFyblwiLCB7XHJcbiAgICAgICAgdmFsdWU6IGFwaVNlcnZpY2Uuc2VydmljZUFybixcclxuICAgICAgICBkZXNjcmlwdGlvbjogXCJBcHAgUnVubmVyIFNlcnZpY2UgQVJOXCIsXHJcbiAgICAgIH0pO1xyXG4gICAgfVxyXG5cclxuICAgIG5ldyBjZGsuQ2ZuT3V0cHV0KHRoaXMsIFwiRnJvbnRlbmRVcmxcIiwge1xyXG4gICAgICB2YWx1ZTogYGh0dHBzOi8vJHtGUk9OVEVORF9ET01BSU59YCxcclxuICAgICAgZGVzY3JpcHRpb246IFwiRnJvbnRlbmQgVVJMIChjdXN0b20gZG9tYWluKVwiLFxyXG4gICAgfSk7XHJcblxyXG4gICAgbmV3IGNkay5DZm5PdXRwdXQodGhpcywgXCJGcm9udGVuZENsb3VkRnJvbnRVcmxcIiwge1xyXG4gICAgICB2YWx1ZTogYGh0dHBzOi8vJHtkaXN0cmlidXRpb24uZGlzdHJpYnV0aW9uRG9tYWluTmFtZX1gLFxyXG4gICAgICBkZXNjcmlwdGlvbjogXCJGcm9udGVuZCBVUkwgKENsb3VkRnJvbnQgZGVmYXVsdClcIixcclxuICAgIH0pO1xyXG5cclxuICAgIG5ldyBjZGsuQ2ZuT3V0cHV0KHRoaXMsIFwiV2ViQnVja2V0TmFtZVwiLCB7XHJcbiAgICAgIHZhbHVlOiB3ZWJzaXRlQnVja2V0LmJ1Y2tldE5hbWUsXHJcbiAgICAgIGRlc2NyaXB0aW9uOiBcIlMzIGJ1Y2tldCBmb3IgZnJvbnRlbmRcIixcclxuICAgIH0pO1xyXG5cclxuICAgIG5ldyBjZGsuQ2ZuT3V0cHV0KHRoaXMsIFwiRGlzdHJpYnV0aW9uSWRcIiwge1xyXG4gICAgICB2YWx1ZTogZGlzdHJpYnV0aW9uLmRpc3RyaWJ1dGlvbklkLFxyXG4gICAgICBkZXNjcmlwdGlvbjogXCJDbG91ZEZyb250IERpc3RyaWJ1dGlvbiBJRFwiLFxyXG4gICAgfSk7XHJcblxyXG4gICAgbmV3IGNkay5DZm5PdXRwdXQodGhpcywgXCJTZWNyZXRzQXJuXCIsIHtcclxuICAgICAgdmFsdWU6IGFwaVNlY3JldHMuc2VjcmV0QXJuLFxyXG4gICAgICBkZXNjcmlwdGlvbjogXCJTZWNyZXRzIE1hbmFnZXIgQVJOIC0gdXBkYXRlIEFQSV9LRVkgaGVyZVwiLFxyXG4gICAgfSk7XHJcblxyXG4gICAgbmV3IGNkay5DZm5PdXRwdXQodGhpcywgXCJHaXRodWJBY3Rpb25zUm9sZUFyblwiLCB7XHJcbiAgICAgIHZhbHVlOiBnaXRodWJBY3Rpb25zUm9sZS5yb2xlQXJuLFxyXG4gICAgICBkZXNjcmlwdGlvbjogXCJJQU0gUm9sZSBBUk4gZm9yIEdpdEh1YiBBY3Rpb25zXCIsXHJcbiAgICB9KTtcclxuICB9XHJcbn1cclxuIl19