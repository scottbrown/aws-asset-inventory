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

```bash
# Ensure the caller has AWS Config permissions (see "Required IAM Permissions").

# Basic usage - uses default AWS credential chain
aws-asset-inventory --regions us-east-1,us-west-2

# With explicit profile
aws-asset-inventory --profile myprofile --regions us-east-1,us-west-2

# Using environment variables for credentials
export AWS_PROFILE=myprofile
aws-asset-inventory --regions us-east-1,us-west-2

# Save JSON inventory to file
aws-asset-inventory --regions us-east-1 --output inventory.json

# Save markdown report to file
aws-asset-inventory --regions us-east-1 --report report.md

# Combined - save both JSON and markdown
aws-asset-inventory --profile myprofile --regions us-east-1,us-west-2 \
  --output inventory.json --report report.md

# JSON to stdout (no report)
aws-asset-inventory --regions us-east-1 --output - --no-report

# JSON to stdout, report to file
aws-asset-inventory --regions us-east-1 --output - --report report.md

# Quick summary without resource details
aws-asset-inventory --regions us-east-1 --summary-only

# Verbose output for debugging
aws-asset-inventory --regions us-east-1,us-west-2 --verbose

# Print required AWS Config permissions (one per line)
aws-asset-inventory --permissions
```

### Flags

| Flag | Short | Required | Description |
|------|-------|----------|-------------|
| `--profile` | `-p` | No | AWS profile name (uses default credential chain if omitted) |
| `--regions` | `-r` | Yes | Comma-separated list of AWS regions |
| `--output` | `-o` | No | Path for JSON inventory output (use `-` for stdout) |
| `--report` | | No | Path for markdown report (use `-` for stdout) |
| `--no-report` | | No | Skip markdown report generation |
| `--summary-only` | | No | Generate summary report without resource details |
| `--verbose` | `-v` | No | Show detailed progress during collection |
| `--permissions` | | No | Print required AWS Config permissions and exit |

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
4. **Resource Details** - Detailed listing of all resources

## Licence

MIT
