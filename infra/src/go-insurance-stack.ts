import * as cdk from "aws-cdk-lib";
import * as s3 from "aws-cdk-lib/aws-s3";
import * as cloudfront from "aws-cdk-lib/aws-cloudfront";
import * as origins from "aws-cdk-lib/aws-cloudfront-origins";
import * as ecr from "aws-cdk-lib/aws-ecr";
import * as apprunner from "@aws-cdk/aws-apprunner-alpha";
import * as iam from "aws-cdk-lib/aws-iam";
import * as secretsmanager from "aws-cdk-lib/aws-secretsmanager";
import * as route53 from "aws-cdk-lib/aws-route53";
import * as route53Targets from "aws-cdk-lib/aws-route53-targets";
import * as acm from "aws-cdk-lib/aws-certificatemanager";
import { Construct } from "constructs";

const PROJECT_NAME = "go-insurance";

// Domain configuration
const HOSTED_ZONE_NAME = "labs.iron-pig.com";
const FRONTEND_DOMAIN = "insurance.labs.iron-pig.com";
const API_DOMAIN = "api.insurance.labs.iron-pig.com";

export class GoInsuranceStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
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
    const frontendCertificate = new acm.DnsValidatedCertificate(
      this,
      "FrontendCertificate",
      {
        domainName: FRONTEND_DOMAIN,
        hostedZone: hostedZone,
        region: "us-east-1", // Required for CloudFront
      }
    );

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
      target: route53.RecordTarget.fromAlias(
        new route53Targets.CloudFrontTarget(distribution)
      ),
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
    let apiService: apprunner.Service | undefined;
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
    const githubOidcProvider = iam.OpenIdConnectProvider.fromOpenIdConnectProviderArn(
      this,
      "GithubOidcProvider",
      `arn:aws:iam::${this.account}:oidc-provider/token.actions.githubusercontent.com`
    );

    const githubActionsRole = new iam.Role(this, "GithubActionsRole", {
      roleName: `github-actions-${PROJECT_NAME}`,
      assumedBy: new iam.FederatedPrincipal(
        githubOidcProvider.openIdConnectProviderArn,
        {
          StringEquals: {
            "token.actions.githubusercontent.com:aud": "sts.amazonaws.com",
          },
          StringLike: {
            "token.actions.githubusercontent.com:sub": `repo:${githubOrg}/${githubRepo}:*`,
          },
        },
        "sts:AssumeRoleWithWebIdentity"
      ),
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
