# OpenShift Health Check Runner

A comprehensive tool for evaluating OpenShift cluster configurations against best practices and providing actionable recommendations.

![health-check-demo](images/hc.gif)

## Overview

This Go application performs thorough health checks on OpenShift clusters to identify potential issues and suggest improvements. The tool runs more than 40 specialized checks across multiple categories:

- **Cluster Configuration**: Node status, operator health, version compliance, etc.
- **Networking**: Ingress controller, CNI plugins, network policies, etc.
- **Security**: SCC configurations, authentication providers, etcd encryption, etc.
- **Applications**: Probe configurations, resource quotas, volume usage, etc.
- **Storage**: Storage classes, persistent volumes, storage performance, etc.
- **Operational Readiness**: Monitoring, logging, alert forwarding, etc.

Each check evaluates a specific aspect of the cluster and provides detailed analysis, observations, and recommendations.

## Prerequisites

- Go 1.20 or higher
- Access to an OpenShift cluster (version 4.14 or later)
- Valid `KUBECONFIG` or authenticated `oc login` session

## Installation

Clone the repository and build the application:

```bash
git clone https://github.com/yourusername/health-check-runner.git
cd health-check-runner
./build.sh
```

The `build.sh` script will create a binary named `health-check-runner` in the current directory.

## Usage

Run the health check against your OpenShift cluster:

```bash
./health-check-runner --output-dir ./health-reports --parallel --verbose
```

### Command-line Options

- `--output-dir <directory>`: Directory where reports will be saved (default: "./reports")
- `--parallel`: Run checks in parallel for faster execution
- `--verbose`: Display detailed output during check execution
- `--timeout <duration>`: Maximum time allowed for a check (e.g., "60s", "5m")
- `--format <format>`: Output format (asciidoc, html, json, summary)
- `--no-progress`: Disable progress bar display


## Reports

The application generates comprehensive reports in AsciiDoc format by default. The reports include:

- Summary tables with color-coded status indicators
- Detailed check results organized by category
- Specific recommendations for addressing issues
- Links to official documentation for further guidance

The generated reports are saved in the specified output directory (default: "./reports"). The main report file is protected with a password for security.

## Report Types

The tool can generate reports in the following formats:

- **AsciiDoc**: Comprehensive, formatted reports with color-coded sections
- **HTML**: Web-friendly reports that can be viewed in browsers
- **JSON**: Machine-readable format for integration with other tools
- **Summary**: Brief text summary of issues found

## Contributing

Contributions are welcome! If you'd like to add new checks, improve existing ones, or enhance the tool's functionality:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/new-check`)
3. Commit your changes (`git commit -am 'Add new check for XYZ'`)
4. Push to the branch (`git push origin feature/new-check`)
5. Create a pull request

## License

[Add license information here]

## Disclaimer

This is a diagnostic tool that provides recommendations based on best practices. Always review suggestions carefully before implementing changes in production environments.

> **Note**: This application is still under development and might produce unexpected results in some environments.