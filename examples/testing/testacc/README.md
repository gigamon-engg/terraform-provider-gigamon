# Acceptance Test CI Groups

This folder contains Terraform provider acceptance tests for app resources, AMI/AFI coverage, and traffic map condition coverage.

## Prerequisites

Set these environment variables before running any acceptance tests:

```bash
export TF_ACC=1
export FM_ADDRESS="https://<fm-host>"
export FM_API_TOKEN="<token>"
export FM_MONITORING_SESSION_ID="<session-id>"
```

## CI Target Groups

Use Go `-run` patterns to execute focused test groups.

### apps

Runs all app acceptance tests (5gcloud + app viz + AMI + phase1 app table tests):

```bash
go test ./examples/testing/testacc -v -run 'TestAcc(App5GCloudResource|AppVizResource|AmiResource|Phase1Apps)_'
```

### afi

Runs AFI acceptance tests:

```bash
go test ./examples/testing/testacc -v -run 'TestAccTrafficMapAfiResource_'
```

### inline

Runs initial inline workflow resource tests:

```bash
go test ./examples/testing/testacc -v -run 'TestAccUnifiedMonitoringSession_'
```

### map-conditions

Runs phase2 traffic map condition tests:

```bash
go test ./examples/testing/testacc -v -run 'TestAccTrafficMapPhase2Conditions_'
```

## Common Local Commands

Run only validation/negative-path checks quickly:

```bash
go test ./examples/testing/testacc -v -run 'TestAcc(App5GCloudResource_RequiredFieldMissing|AppVizResource_RequiredFieldMissing|AmiResource_RequiredFieldMissing|UnifiedMonitoringSession_RequiredFields|Phase1Apps_RequiredMonitoringSessionID|TrafficMapAfiResource_Validation|TrafficMapPhase2Conditions_Validation)$'
```

Compile test package without executing tests:

```bash
go test -c ./examples/testing/testacc
```

## Test Naming Convention

Use this naming style for new tests:

-   `TestAcc<AppOrDomain><ResourceOrScope>_<Scenario>`
-   Keep scenario suffixes consistent, such as `Basic`, `Validation`, `RequiredFieldMissing`, or `BasicUpdateImport`.
-   Group selectors should remain stable so CI patterns do not need frequent updates.
