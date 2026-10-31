# AWS Asset Inventory

A CLI tool that collects AWS resources from AWS Config across specified regions and generates inventory reports.

## Features

- Collects all resources tracked by AWS Config
- Supports multiple AWS regions
- Outputs raw inventory as JSON
- Records how resources relate to each other (for example, instance → subnet → VPC) and their tags
- Generates markdown summary reports with:
  - Resource counts by type
  - Resource counts by region
  - Detailed resource listings
- Serves an interactive 3D map of the inventory that you can fly through in the browser

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
        "config:BatchGetResourceConfig",
        "config:SelectResourceConfig"
      ],
      "Resource": "*"
    }
  ]
}
```

Note: If you collect across multiple regions, these permissions must apply in each target region.

`config:SelectResourceConfig` is used to get relationships and tags. Without it, collection still works, but the tool prints a warning for each region and the inventory has no relationships.

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

### Explore in 3D

Start a local web server that shows the inventory as a 3D map:

```bash
aws-asset-inventory serve --input inventory.json
# Serving 1234 resources at http://127.0.0.1:8080 (press Ctrl+C to stop)
```

Open the URL in a browser. Each AWS Config relationship is a line. Each resource is a 3D shape:

- **Colour** shows the service (EC2, S3, IAM, …).
- **Shape** shows the resource type. Common types have a shape that suggests what they are: instances are cubes, volumes and databases are cylinders, S3 buckets are buckets, security groups are rings. In one service, each other type gets a different shape.
- **Outline** shapes are containers that hold other resources, such as VPCs, subnets, clusters and stacks.
- **Size** shows the number of relationships.

The **Resource types** list in the sidebar is the legend. It shows the shape and colour of each type, grouped by service, and you can use it to show or hide types.

- **Fly** through the map: W/S forward and back, A/D left and right, R/F up and down, Q/E roll, arrow keys or drag to look. Use the Camera menu to change to orbit controls.
- **Click** a resource to fly to it, highlight its relationships, and see its tags, relationships, and configuration.
- **Find** a resource by name, ID, or type.
- **Filter** by region and service, and hide resources with no relationships.

Resources that are referred to by a relationship but are not in the inventory (for example, a resource type that AWS Config does not record) are shown as small grey shapes.

The server listens on `127.0.0.1` only by default, because the inventory can contain sensitive data. All assets are embedded in the binary, so the viewer works offline.

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

# Step 3: Explore the inventory in 3D
aws-asset-inventory serve --input inventory.json
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

### serve

Start a web server that shows inventory JSON as an interactive 3D map.

| Flag | Short | Required | Description |
|------|-------|----------|-------------|
| `--input` | `-i` | Yes | Input JSON inventory file |
| `--listen` | `-l` | No | Address to listen on (default `127.0.0.1:8080`). A non-loopback address makes the inventory readable by anyone who can reach it. |

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
- `vendor` - Rebuild the 3D viewer's JavaScript bundle from `tools/vendor` (needs Node.js)
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
      "configuration": { ... },
      "tags": { "Name": "my-instance" },
      "relationships": [
        {
          "resourceType": "AWS::EC2::VPC",
          "resourceId": "vpc-12345",
          "relationshipName": "Is contained in Vpc"
        }
      ]
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

The 3D viewer includes a bundle of [3d-force-graph](https://github.com/vasturiano/3d-force-graph), [three.js](https://threejs.org/) and their dependencies, all under permissive licences. See `awsassetinventory/web/vendor/graph.LICENSES.txt`. To change their versions, edit `tools/vendor/package.json` and run `task vendor`.
