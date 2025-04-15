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

	// Following the order in the PDF:

	// Infrastructure Provider - first in Infra category
	checks = append(checks, NewInfrastructureProviderCheck())

	// Installation Type
	checks = append(checks, NewInstallationTypeCheck())

	// UPI with MachineSets check
	checks = append(checks, NewUPIMachineSetCheck())

	// Node Status
	checks = append(checks, NewNodeStatusCheck())

	// Node Usage
	checks = append(checks, NewNodeUsageCheck())

	// Cluster Version - first in Cluster Config
	checks = append(checks, NewClusterVersionCheck(latestVersion))

	// Cluster Operators
	checks = append(checks, NewClusterOperatorsCheck())

	// Control Nodes Schedulable
	checks = append(checks, NewControlNodeSchedulableCheck())

	// Infrastructure Nodes
	checks = append(checks, NewInfrastructureNodesCheck())

	// Workload off Infra Nodes
	checks = append(checks, NewWorkloadOffInfraNodesCheck())

	// Default Project Template - comes from security in PDF

	// Default Node Schedule
	checks = append(checks, NewDefaultNodeScheduleCheck()) // Fixed function name here

	// Infra machine config pool
	checks = append(checks, NewInfraMachineConfigPoolCheck())

	// Kubelet Configuration
	checks = append(checks, NewKubeletGarbageCollectionCheck())

	// ETCD health and performance checks
	checks = append(checks, NewEtcdHealthCheck())
	checks = append(checks, NewEtcdPerformanceCheck())

	// Openshift Proxy Settings
	checks = append(checks, NewProxySettingsCheck())

	// Infrastructure node taints
	checks = append(checks, NewInfraTaintsCheck())

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
