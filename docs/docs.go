// Package docs provides Swagger documentation for the Go Insurance API.
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "swagger": "2.0",
    "info": {
        "title": "Go Insurance API",
        "description": "Life Insurance Quote and Policy Management API.\n\nThis API implements a complete insurance workflow:\n1. **Products** - Browse available insurance products\n2. **Quotes** - Get pricing for coverage options\n3. **Applications** - Submit application with applicant info\n4. **Underwriting** - Automatic/manual risk assessment\n5. **Offers** - Accept or decline approved offers\n6. **Policies** - Issued policies after offer acceptance",
        "contact": {
            "name": "API Support",
            "url": "https://github.com/MrKriegler/go-insurance"
        },
        "license": {
            "name": "MIT"
        },
        "version": "1.0.0"
    },
    "host": "localhost:8080",
    "basePath": "/api/v1",
    "schemes": ["http", "https"],
    "consumes": ["application/json"],
    "produces": ["application/json"],
    "paths": {
        "/products": {
            "get": {
                "tags": ["Products"],
                "summary": "List all products",
                "description": "Returns all available insurance products",
                "operationId": "listProducts",
                "responses": {
                    "200": {
                        "description": "Successful response",
                        "schema": {
                            "type": "array",
                            "items": {"$ref": "#/definitions/Product"}
                        }
                    },
                    "500": {
                        "description": "Internal server error",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    }
                }
            }
        },
        "/products/{product_slug}": {
            "get": {
                "tags": ["Products"],
                "summary": "Get a product by slug",
                "description": "Returns a single insurance product",
                "operationId": "getProduct",
                "parameters": [
                    {
                        "name": "product_slug",
                        "in": "path",
                        "required": true,
                        "type": "string",
                        "description": "Product slug (e.g., term-life-10)"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Successful response",
                        "schema": {"$ref": "#/definitions/Product"}
                    },
                    "404": {
                        "description": "Product not found",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    }
                }
            }
        },
        "/quotes": {
            "post": {
                "tags": ["Quotes"],
                "summary": "Create a quote",
                "description": "Prices coverage and creates a quote valid for 24 hours",
                "operationId": "createQuote",
                "parameters": [
                    {
                        "name": "body",
                        "in": "body",
                        "required": true,
                        "schema": {"$ref": "#/definitions/QuoteInput"}
                    }
                ],
                "responses": {
                    "201": {
                        "description": "Quote created",
                        "schema": {"$ref": "#/definitions/Quote"}
                    },
                    "400": {
                        "description": "Validation error",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    },
                    "404": {
                        "description": "Product not found",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    }
                }
            }
        },
        "/quotes/{quote_id}": {
            "get": {
                "tags": ["Quotes"],
                "summary": "Get a quote by ID",
                "description": "Returns a single quote",
                "operationId": "getQuote",
                "parameters": [
                    {
                        "name": "quote_id",
                        "in": "path",
                        "required": true,
                        "type": "string",
                        "description": "Quote ID (ULID)"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Successful response",
                        "schema": {"$ref": "#/definitions/Quote"}
                    },
                    "404": {
                        "description": "Quote not found",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    }
                }
            }
        },
        "/applications": {
            "post": {
                "tags": ["Applications"],
                "summary": "Create an application",
                "description": "Creates a draft application from a quote",
                "operationId": "createApplication",
                "parameters": [
                    {
                        "name": "body",
                        "in": "body",
                        "required": true,
                        "schema": {"$ref": "#/definitions/ApplicationInput"}
                    }
                ],
                "responses": {
                    "201": {
                        "description": "Application created",
                        "schema": {"$ref": "#/definitions/Application"}
                    },
                    "400": {
                        "description": "Validation error",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    },
                    "404": {
                        "description": "Quote not found",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    },
                    "409": {
                        "description": "Quote already used",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    }
                }
            }
        },
        "/applications/{application_id}": {
            "get": {
                "tags": ["Applications"],
                "summary": "Get an application",
                "description": "Returns a single application",
                "operationId": "getApplication",
                "parameters": [
                    {
                        "name": "application_id",
                        "in": "path",
                        "required": true,
                        "type": "string",
                        "description": "Application ID (ULID)"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Successful response",
                        "schema": {"$ref": "#/definitions/Application"}
                    },
                    "404": {
                        "description": "Application not found",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    }
                }
            },
            "patch": {
                "tags": ["Applications"],
                "summary": "Update an application",
                "description": "Updates applicant information (only in draft status)",
                "operationId": "patchApplication",
                "parameters": [
                    {
                        "name": "application_id",
                        "in": "path",
                        "required": true,
                        "type": "string"
                    },
                    {
                        "name": "body",
                        "in": "body",
                        "required": true,
                        "schema": {"$ref": "#/definitions/ApplicationPatch"}
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Application updated",
                        "schema": {"$ref": "#/definitions/Application"}
                    },
                    "400": {
                        "description": "Validation error",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    },
                    "404": {
                        "description": "Application not found",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    },
                    "409": {
                        "description": "Application not in draft status",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    }
                }
            }
        },
        "/applications/{application_id}:submit": {
            "post": {
                "tags": ["Applications"],
                "summary": "Submit an application",
                "description": "Submits the application for underwriting review",
                "operationId": "submitApplication",
                "parameters": [
                    {
                        "name": "application_id",
                        "in": "path",
                        "required": true,
                        "type": "string"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Application submitted",
                        "schema": {"$ref": "#/definitions/Application"}
                    },
                    "400": {
                        "description": "Incomplete application",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    },
                    "404": {
                        "description": "Application not found",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    },
                    "409": {
                        "description": "Already submitted",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    }
                }
            }
        },
        "/underwriting/cases": {
            "get": {
                "tags": ["Underwriting"],
                "summary": "List referred cases",
                "description": "Returns underwriting cases awaiting manual review",
                "operationId": "listReferredCases",
                "responses": {
                    "200": {
                        "description": "Successful response",
                        "schema": {
                            "type": "array",
                            "items": {"$ref": "#/definitions/UnderwritingCase"}
                        }
                    },
                    "500": {
                        "description": "Internal server error",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    }
                }
            }
        },
        "/underwriting/cases/{case_id}": {
            "get": {
                "tags": ["Underwriting"],
                "summary": "Get an underwriting case",
                "description": "Returns details of an underwriting case",
                "operationId": "getUnderwritingCase",
                "parameters": [
                    {
                        "name": "case_id",
                        "in": "path",
                        "required": true,
                        "type": "string"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Successful response",
                        "schema": {"$ref": "#/definitions/UnderwritingCase"}
                    },
                    "404": {
                        "description": "Case not found",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    }
                }
            }
        },
        "/underwriting/cases/{case_id}:decide": {
            "post": {
                "tags": ["Underwriting"],
                "summary": "Make a decision",
                "description": "Manually approve or decline a referred case",
                "operationId": "decideCase",
                "parameters": [
                    {
                        "name": "case_id",
                        "in": "path",
                        "required": true,
                        "type": "string"
                    },
                    {
                        "name": "body",
                        "in": "body",
                        "required": true,
                        "schema": {"$ref": "#/definitions/UWDecisionInput"}
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Decision recorded",
                        "schema": {"$ref": "#/definitions/UnderwritingCase"}
                    },
                    "400": {
                        "description": "Validation error",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    },
                    "404": {
                        "description": "Case not found",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    },
                    "409": {
                        "description": "Already decided",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    }
                }
            }
        },
        "/applications/{application_id}/offers": {
            "post": {
                "tags": ["Offers"],
                "summary": "Generate an offer",
                "description": "Creates an offer from an approved application",
                "operationId": "createOffer",
                "parameters": [
                    {
                        "name": "application_id",
                        "in": "path",
                        "required": true,
                        "type": "string"
                    }
                ],
                "responses": {
                    "201": {
                        "description": "Offer created",
                        "schema": {"$ref": "#/definitions/Offer"}
                    },
                    "404": {
                        "description": "Application not found",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    },
                    "409": {
                        "description": "Application not approved or offer exists",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    }
                }
            }
        },
        "/offers/{offer_id}": {
            "get": {
                "tags": ["Offers"],
                "summary": "Get an offer",
                "description": "Returns a single offer",
                "operationId": "getOffer",
                "parameters": [
                    {
                        "name": "offer_id",
                        "in": "path",
                        "required": true,
                        "type": "string"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Successful response",
                        "schema": {"$ref": "#/definitions/Offer"}
                    },
                    "404": {
                        "description": "Offer not found",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    }
                }
            }
        },
        "/offers/{offer_id}:accept": {
            "post": {
                "tags": ["Offers"],
                "summary": "Accept an offer",
                "description": "Accepts the offer, triggering policy issuance",
                "operationId": "acceptOffer",
                "parameters": [
                    {
                        "name": "offer_id",
                        "in": "path",
                        "required": true,
                        "type": "string"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Offer accepted",
                        "schema": {"$ref": "#/definitions/Offer"}
                    },
                    "404": {
                        "description": "Offer not found",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    },
                    "409": {
                        "description": "Offer expired or not pending",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    }
                }
            }
        },
        "/offers/{offer_id}:decline": {
            "post": {
                "tags": ["Offers"],
                "summary": "Decline an offer",
                "description": "Declines the offer",
                "operationId": "declineOffer",
                "parameters": [
                    {
                        "name": "offer_id",
                        "in": "path",
                        "required": true,
                        "type": "string"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Offer declined",
                        "schema": {"$ref": "#/definitions/Offer"}
                    },
                    "404": {
                        "description": "Offer not found",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    },
                    "409": {
                        "description": "Offer not pending",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    }
                }
            }
        },
        "/policies": {
            "get": {
                "tags": ["Policies"],
                "summary": "List policies",
                "description": "Returns policies with optional filtering and pagination",
                "operationId": "listPolicies",
                "parameters": [
                    {
                        "name": "application_id",
                        "in": "query",
                        "type": "string",
                        "description": "Filter by application ID"
                    },
                    {
                        "name": "status",
                        "in": "query",
                        "type": "string",
                        "enum": ["active", "lapsed", "cancelled", "expired"],
                        "description": "Filter by status"
                    },
                    {
                        "name": "limit",
                        "in": "query",
                        "type": "integer",
                        "default": 20,
                        "description": "Page size"
                    },
                    {
                        "name": "offset",
                        "in": "query",
                        "type": "integer",
                        "default": 0,
                        "description": "Page offset"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Successful response",
                        "schema": {"$ref": "#/definitions/PolicyList"}
                    },
                    "500": {
                        "description": "Internal server error",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    }
                }
            }
        },
        "/policies/{policy_number}": {
            "get": {
                "tags": ["Policies"],
                "summary": "Get a policy by number",
                "description": "Returns a single policy",
                "operationId": "getPolicy",
                "parameters": [
                    {
                        "name": "policy_number",
                        "in": "path",
                        "required": true,
                        "type": "string",
                        "description": "Policy number (e.g., POL-2025-000001)"
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Successful response",
                        "schema": {"$ref": "#/definitions/Policy"}
                    },
                    "404": {
                        "description": "Policy not found",
                        "schema": {"$ref": "#/definitions/ProblemDetails"}
                    }
                }
            }
        }
    },
    "definitions": {
        "Product": {
            "type": "object",
            "properties": {
                "id": {"type": "string", "example": "01HXYZ..."},
                "slug": {"type": "string", "example": "term-life-10"},
                "name": {"type": "string", "example": "10-Year Term Life"},
                "term_years": {"type": "integer", "example": 10},
                "min_coverage": {"type": "integer", "example": 50000},
                "max_coverage": {"type": "integer", "example": 1000000},
                "base_rate": {"type": "number", "example": 0.25}
            }
        },
        "QuoteInput": {
            "type": "object",
            "required": ["product_slug", "coverage_amount", "term_years", "age"],
            "properties": {
                "product_slug": {"type": "string", "example": "term-life-10"},
                "coverage_amount": {"type": "integer", "example": 150000},
                "term_years": {"type": "integer", "example": 10},
                "age": {"type": "integer", "example": 35},
                "smoker": {"type": "boolean", "example": false}
            }
        },
        "Quote": {
            "type": "object",
            "properties": {
                "id": {"type": "string", "example": "01HXYZ..."},
                "product_id": {"type": "string"},
                "product_slug": {"type": "string", "example": "term-life-10"},
                "coverage_amount": {"type": "integer", "example": 150000},
                "term_years": {"type": "integer", "example": 10},
                "monthly_premium": {"type": "number", "example": 37.50},
                "status": {"type": "string", "enum": ["new", "priced", "expired"]},
                "created_at": {"type": "string", "format": "date-time"},
                "expires_at": {"type": "string", "format": "date-time"}
            }
        },
        "Applicant": {
            "type": "object",
            "required": ["first_name", "last_name", "email", "date_of_birth", "age", "state"],
            "properties": {
                "first_name": {"type": "string", "example": "John"},
                "last_name": {"type": "string", "example": "Doe"},
                "email": {"type": "string", "format": "email", "example": "john@example.com"},
                "date_of_birth": {"type": "string", "example": "1989-06-15"},
                "age": {"type": "integer", "example": 35},
                "smoker": {"type": "boolean", "example": false},
                "state": {"type": "string", "example": "CA"}
            }
        },
        "ApplicationInput": {
            "type": "object",
            "required": ["quote_id", "applicant"],
            "properties": {
                "quote_id": {"type": "string", "example": "01HXYZ..."},
                "applicant": {"$ref": "#/definitions/Applicant"}
            }
        },
        "ApplicationPatch": {
            "type": "object",
            "properties": {
                "applicant": {"$ref": "#/definitions/Applicant"}
            }
        },
        "Application": {
            "type": "object",
            "properties": {
                "id": {"type": "string"},
                "quote_id": {"type": "string"},
                "product_id": {"type": "string"},
                "product_slug": {"type": "string"},
                "coverage_amount": {"type": "integer"},
                "term_years": {"type": "integer"},
                "monthly_premium": {"type": "number"},
                "applicant": {"$ref": "#/definitions/Applicant"},
                "status": {"type": "string", "enum": ["draft", "submitted", "under_review", "approved", "declined"]},
                "created_at": {"type": "string", "format": "date-time"},
                "updated_at": {"type": "string", "format": "date-time"},
                "submitted_at": {"type": "string", "format": "date-time"}
            }
        },
        "RiskFactors": {
            "type": "object",
            "properties": {
                "age": {"type": "integer"},
                "smoker": {"type": "boolean"},
                "coverage_amount": {"type": "integer"},
                "term_years": {"type": "integer"}
            }
        },
        "RiskScore": {
            "type": "object",
            "properties": {
                "score": {"type": "integer", "description": "0-100, higher = riskier"},
                "flags": {"type": "array", "items": {"type": "string"}},
                "recommended": {"type": "string", "enum": ["approved", "declined", "referred"]}
            }
        },
        "UnderwritingCase": {
            "type": "object",
            "properties": {
                "id": {"type": "string"},
                "application_id": {"type": "string"},
                "risk_factors": {"$ref": "#/definitions/RiskFactors"},
                "risk_score": {"$ref": "#/definitions/RiskScore"},
                "decision": {"type": "string", "enum": ["pending", "approved", "declined", "referred"]},
                "method": {"type": "string", "enum": ["auto", "manual"]},
                "decided_by": {"type": "string"},
                "reason": {"type": "string"},
                "created_at": {"type": "string", "format": "date-time"},
                "updated_at": {"type": "string", "format": "date-time"},
                "decided_at": {"type": "string", "format": "date-time"}
            }
        },
        "UWDecisionInput": {
            "type": "object",
            "required": ["decision", "reason"],
            "properties": {
                "decision": {"type": "string", "enum": ["approved", "declined"]},
                "reason": {"type": "string", "example": "Risk factors within acceptable limits"}
            }
        },
        "Offer": {
            "type": "object",
            "properties": {
                "id": {"type": "string"},
                "application_id": {"type": "string"},
                "product_slug": {"type": "string"},
                "coverage_amount": {"type": "integer"},
                "term_years": {"type": "integer"},
                "monthly_premium": {"type": "number"},
                "status": {"type": "string", "enum": ["pending", "accepted", "declined", "expired", "issued"]},
                "created_at": {"type": "string", "format": "date-time"},
                "expires_at": {"type": "string", "format": "date-time"},
                "accepted_at": {"type": "string", "format": "date-time"},
                "declined_at": {"type": "string", "format": "date-time"}
            }
        },
        "Policy": {
            "type": "object",
            "properties": {
                "id": {"type": "string"},
                "number": {"type": "string", "example": "POL-2025-000001"},
                "application_id": {"type": "string"},
                "offer_id": {"type": "string"},
                "product_slug": {"type": "string"},
                "coverage_amount": {"type": "integer"},
                "term_years": {"type": "integer"},
                "monthly_premium": {"type": "number"},
                "insured": {"$ref": "#/definitions/Applicant"},
                "status": {"type": "string", "enum": ["active", "lapsed", "cancelled", "expired"]},
                "effective_date": {"type": "string", "format": "date-time"},
                "expiry_date": {"type": "string", "format": "date-time"},
                "issued_at": {"type": "string", "format": "date-time"}
            }
        },
        "PolicyList": {
            "type": "object",
            "properties": {
                "items": {"type": "array", "items": {"$ref": "#/definitions/Policy"}},
                "total": {"type": "integer"},
                "limit": {"type": "integer"},
                "offset": {"type": "integer"}
            }
        },
        "ProblemDetails": {
            "type": "object",
            "description": "RFC 7807 Problem Details",
            "properties": {
                "type": {"type": "string", "example": "about:blank"},
                "title": {"type": "string", "example": "Not Found"},
                "status": {"type": "integer", "example": 404},
                "detail": {"type": "string", "example": "Resource not found"}
            }
        }
    },
    "tags": [
        {"name": "Products", "description": "Insurance product catalog"},
        {"name": "Quotes", "description": "Get pricing for coverage options"},
        {"name": "Applications", "description": "Manage insurance applications"},
        {"name": "Underwriting", "description": "Risk assessment and decisions"},
        {"name": "Offers", "description": "Accept or decline approved offers"},
        {"name": "Policies", "description": "Issued insurance policies"}
    ]
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.0.0",
	Host:             "localhost:8080",
	BasePath:         "/api/v1",
	Schemes:          []string{"http", "https"},
	Title:            "Go Insurance API",
	Description:      "Life Insurance Quote and Policy Management API",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
