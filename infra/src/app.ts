#!/usr/bin/env node
import "source-map-support/register";
import * as cdk from "aws-cdk-lib";
import { GoInsuranceStack } from "./go-insurance-stack";

const app = new cdk.App();

new GoInsuranceStack(app, "GoInsurance", {
  env: {
    account: process.env.CDK_DEFAULT_ACCOUNT,
    region: process.env.CDK_DEFAULT_REGION || "eu-west-2",
  },
  description: "Go Insurance API - Demo",
});

app.synth();
