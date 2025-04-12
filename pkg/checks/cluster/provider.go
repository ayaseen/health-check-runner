package cluster

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
)

// GetChecks returns all cluster-related health checks
func GetChecks() []healthcheck.Check {
	// Retrieve the latest OpenShift version
	latestVersion, err := getLatestOpenShiftVersion()
	if err != nil {
		// Fall back to a hardcoded version if there's an error
		latestVersion = "4.14.0"
	}

	var checks []healthcheck.Check

	// Add cluster version check
	checks = append(checks, NewClusterVersionCheck(latestVersion))

	// Add cluster operators check
	checks = append(checks, NewClusterOperatorsCheck())

	// Add nodes check
	checks = append(checks, NewNodeStatusCheck())

	// Add node usage check
	checks = append(checks, NewNodeUsageCheck())

	// Add control node schedulable check
	checks = append(checks, NewControlNodeSchedulableCheck())

	// Add infrastructure nodes check - from infra_nodes.go
	checks = append(checks, NewInfrastructureNodesCheck())

	// Add infrastructure machine config pool check - from infra_config_pool.go
	checks = append(checks, NewInfraMachineConfigPoolCheck())

	// Cluster default SCC check moved to security package

	// Add infrastructure provider check - from infrastructure_provider.go
	checks = append(checks, NewInfrastructureProviderCheck())

	// Add installation type check
	checks = append(checks, NewInstallationTypeCheck())

	// Add workload off infra nodes check
	checks = append(checks, NewWorkloadOffInfraNodesCheck())

	// Add proxy settings check
	checks = append(checks, NewProxySettingsCheck())

	// Note: DefaultProjectTemplateCheck and KubeadminUserCheck moved to security package

	return checks
}

// VersionInfo represents the version information from the Red Hat API
type VersionInfo struct {
	LatestReleases struct {
		Stable string `json:"stable"`
	} `json:"latest_releases"`
}

// getLatestOpenShiftVersion attempts to get the latest OpenShift version from Red Hat
func getLatestOpenShiftVersion() (string, error) {
	// Define API URL
	apiUrl := "https://api.openshift.com/api/upgrades_info/v1/graph"

	// Create a client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Make the request
	resp, err := client.Get(apiUrl)
	if err != nil {
		return "", fmt.Errorf("failed to get latest version: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	// Parse the JSON response
	var versionInfo VersionInfo
	if err := json.Unmarshal(body, &versionInfo); err != nil {
		return "", fmt.Errorf("failed to parse version info: %v", err)
	}

	// Extract and validate the version
	version := versionInfo.LatestReleases.Stable
	if version == "" {
		return "", fmt.Errorf("empty version received from API")
	}

	// Clean the version string (remove leading 'v' if present)
	version = strings.TrimPrefix(version, "v")

	return version, nil
}
