# AWS Asset Inventory

A CLI tool that collects AWS resources from AWS Config across specified regions and generates inventory reports.

## Features

- Collects all resources tracked by AWS Config
- Supports multiple AWS regions
- Outputs raw inventory as JSON
- Generates markdown summary reports with:
  - Resource counts by type
  - Resource counts by region
  - Detailed resource listings

## Prerequisites

- Go 1.23 or later
- [Task](https://taskfile.dev/) (optional, for build automation)
- AWS credentials configured with access to AWS Config
- AWS Config must be enabled in each target region you query

### Required IAM Permissions

At minimum, the caller needs the following AWS Config permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "config:GetDiscoveredResourceCounts",
        "config:ListDiscoveredResources",
        "config:BatchGetResourceConfig"
      ],
      "Resource": "*"
    }
  ]
}
```

Note: If you collect across multiple regions, these permissions must apply in each target region.

## Installation

```bash
# Clone the repository
git clone https://github.com/scottbrown/aws-asset-inventory.git
cd aws-asset-inventory

# Build the binary
task build
# or without Task:
go build -o .build/aws-asset-inventory ./cmd/aws-asset-inventory
```

## Usage

The CLI uses subcommands to separate collection from reporting, allowing you to collect once and generate reports multiple times.

### Collect Resources

Collect AWS resources and save to a JSON file:

```bash
# Basic collection - outputs JSON to stdout
aws-asset-inventory collect --regions us-east-1,us-west-2

# Save to file
aws-asset-inventory collect --regions us-east-1,us-west-2 --output inventory.json

# With explicit AWS profile
aws-asset-inventory collect --profile myprofile --regions us-east-1 --output inventory.json

# Verbose output for debugging
aws-asset-inventory collect --regions us-east-1 --output inventory.json --verbose

# Control concurrency
aws-asset-inventory collect --regions us-east-1,us-west-2,eu-west-1 --concurrency 3 --output inventory.json
```

### Generate Reports

Generate markdown reports from collected inventory:

```bash
# Basic report to stdout
aws-asset-inventory report --input inventory.json

# Save report to file
aws-asset-inventory report --input inventory.json --output report.md

# Include detailed resource listings
aws-asset-inventory report --input inventory.json --output report.md --include-details
```

### Other Commands

```bash
# Print version information
aws-asset-inventory version

# Print required AWS Config permissions (one per line)
aws-asset-inventory permissions
```

### Typical Workflow

```bash
# Step 1: Collect resources (slow operation, run once)
aws-asset-inventory collect --profile myprofile --regions us-east-1,us-west-2 --output inventory.json

# Step 2: Generate reports (fast, can run multiple times)
aws-asset-inventory report --input inventory.json --output summary.md
aws-asset-inventory report --input inventory.json --output detailed.md --include-details
```

## Subcommands

### collect

Collect AWS resources from AWS Config.

| Flag | Short | Required | Description |
|------|-------|----------|-------------|
| `--regions` | `-r` | Yes | Comma-separated list of AWS regions |
| `--profile` | `-p` | No | AWS profile name (uses default credential chain if omitted) |
| `--output` | `-o` | No | Output file path (default: stdout) |
| `--verbose` | `-v` | No | Show detailed progress during collection |
| `--concurrency` | | No | Max concurrent region collections (default 5) |

### report

Generate a markdown report from inventory JSON.

| Flag | Short | Required | Description |
|------|-------|----------|-------------|
| `--input` | `-i` | Yes | Input JSON inventory file |
| `--output` | `-o` | No | Output file path (default: stdout) |
| `--include-details` | | No | Include resource details in report |

### version

Print version information. No flags.

### permissions

Print required AWS IAM permissions. No flags.

## AWS Credential Chain

When `--profile` is omitted, the tool uses the [default AWS credential chain](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configuring-sdk.html), which checks (in order):
1. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
2. Shared credentials file (`~/.aws/credentials`)
3. IAM role (for EC2/ECS/Lambda)

## Development

### Running Tests

```bash
# Run tests with coverage
task test

# Run tests without coverage
task test:short

# View coverage report in browser
task cover
```

### Available Tasks

```bash
task --list
```

- `build` - Build the binary
- `test` - Run tests with coverage
- `test:short` - Run tests without coverage
- `cover` - Open coverage report in browser
- `lint` - Run linter
- `fmt` - Format code
- `vet` - Run go vet
- `tidy` - Tidy go modules
- `clean` - Remove build and test artifacts
- `all` - Run fmt, vet, lint, test, and build

## Output Formats

### JSON Inventory

The JSON output contains the raw inventory data in a structure compatible with AWS Config:

```json
{
  "collectedAt": "2026-01-07T15:30:00Z",
  "profile": "myprofile",
  "regions": ["us-east-1", "us-west-2"],
  "resources": [
    {
      "resourceType": "AWS::EC2::Instance",
      "resourceId": "i-12345",
      "resourceName": "my-instance",
      "awsRegion": "us-east-1",
      "accountId": "123456789012",
      "arn": "arn:aws:ec2:us-east-1:123456789012:instance/i-12345",
      "configuration": { ... }
    }
  ]
}
```

### Markdown Report

The markdown report includes:

1. **Header** - Collection timestamp, profile, and regions
2. **Summary** - Total resource counts by type
3. **By Region** - Resource counts broken down by region
4. **Resource Details** - Detailed listing of all resources (only with `--include-details`)

## Licence

MIT
