/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-17

This file implements health checks for cluster version. It:

- Compares the current OpenShift version against the latest available version
- Verifies if the cluster is running an up-to-date release
- Provides recommendations for updating outdated clusters
- Includes version comparison utilities
- Helps ensure clusters receive security updates and new features

This check helps administrators keep their OpenShift clusters current with the latest improvements and security fixes.
*/

package cluster

import (
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/version"
	"os/exec"
	"strconv"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
)

// ClusterVersionCheck checks if the cluster is running the latest version
type ClusterVersionCheck struct {
	healthcheck.BaseCheck
}

// NewClusterVersionCheck creates a new cluster version check
func NewClusterVersionCheck() *ClusterVersionCheck {
	return &ClusterVersionCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"cluster-version",
			"Cluster Version",
			"Checks if the cluster is running the latest version of OpenShift",
			types.CategoryClusterConfig,
		),
	}
}

// Run executes the health check
func (c *ClusterVersionCheck) Run() (healthcheck.Result, error) {
	// Get the current version of OpenShift
	out, err := exec.Command("oc", "get", "clusterversion", "-o", "jsonpath={.items[].status.history[0].version}").Output()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get cluster version",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting cluster version: %v", err)
	}

	currentVersion := strings.TrimSpace(string(out))

	// If no current version is found, return an error
	if currentVersion == "" {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"No cluster version found",
			types.ResultKeyRequired,
		), fmt.Errorf("no cluster version found")
	}

	// Get detailed cluster version information
	detailedOut, err := exec.Command("oc", "get", "clusterversion", "-o", "yaml").Output()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get detailed cluster version",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting detailed cluster version: %v", err)
	}

	// Get the latest version that was embedded during build
	latestVersion := version.LatestOpenShiftVersion

	// Format the detailed output with proper AsciiDoc formatting
	var formattedDetailedOut strings.Builder
	formattedDetailedOut.WriteString("=== Cluster Version Analysis ===\n\n")
	formattedDetailedOut.WriteString(fmt.Sprintf("Current Version: %s\n", currentVersion))
	formattedDetailedOut.WriteString(fmt.Sprintf("Latest Available Version: %s\n\n", latestVersion))

	// Add detailed cluster version information
	if len(detailedOut) > 0 {
		formattedDetailedOut.WriteString("Cluster Version Details:\n[source, yaml]\n----\n")
		formattedDetailedOut.WriteString(string(detailedOut))
		formattedDetailedOut.WriteString("\n----\n\n")
	} else {
		formattedDetailedOut.WriteString("Cluster Version Details: No information available\n\n")
	}

	// Try to get update history
	updateHistory, _ := exec.Command("oc", "get", "clusterversion", "-o", "jsonpath={.items[].status.history}").Output()
	if len(updateHistory) > 0 {
		formattedDetailedOut.WriteString("Update History:\n[source, json]\n----\n")
		formattedDetailedOut.WriteString(string(updateHistory))
		formattedDetailedOut.WriteString("\n----\n\n")
	}

	// Compare versions
	result, err := compareVersions(currentVersion, latestVersion)
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to compare versions",
			types.ResultKeyRequired,
		), fmt.Errorf("error comparing versions: %v", err)
	}

	// Add version comparison results
	formattedDetailedOut.WriteString("=== Version Comparison ===\n\n")
	if result < 0 {
		formattedDetailedOut.WriteString(fmt.Sprintf("The cluster version %s is older than the latest available version %s.\n\n",
			currentVersion, latestVersion))
	} else if result == 0 {
		formattedDetailedOut.WriteString(fmt.Sprintf("The cluster is running the latest version %s.\n\n", currentVersion))
	} else {
		formattedDetailedOut.WriteString(fmt.Sprintf("The cluster version %s is newer than our reference version %s.\n\n",
			currentVersion, latestVersion))
	}

	if result < 0 {
		// Current version is older than latest version
		checkResult := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			fmt.Sprintf("Cluster version %s is not the latest version (%s)", currentVersion, latestVersion),
			types.ResultKeyRequired,
		)

		checkResult.AddRecommendation(fmt.Sprintf("Update to the latest version %s", latestVersion))
		checkResult.AddRecommendation("Follow the upgrade documentation at https://docs.openshift.com/container-platform/latest/updating/updating-cluster.html")
		checkResult.Detail = formattedDetailedOut.String()

		return checkResult, nil
	}

	// Current version is the latest version or newer
	checkResult := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		fmt.Sprintf("Cluster version %s is up to date", currentVersion),
		types.ResultKeyNoChange,
	)

	checkResult.Detail = formattedDetailedOut.String()

	return checkResult, nil
}

// compareVersions compares two version strings and returns:
// -1 if version1 is older than version2
// 0 if version1 is equal to version2
// 1 if version1 is newer than version2
func compareVersions(version1, version2 string) (int, error) {
	v1 := parseVersion(version1)
	v2 := parseVersion(version2)

	// Compare major version
	if v1[0] != v2[0] {
		return compareInts(v1[0], v2[0])
	}

	// Compare minor version
	if v1[1] != v2[1] {
		return compareInts(v1[1], v2[1])
	}

	// Compare patch version
	if len(v1) > 2 && len(v2) > 2 {
		return compareInts(v1[2], v2[2])
	}

	// If one has a patch version and the other doesn't, the one with a patch version is newer
	if len(v1) > len(v2) {
		return 1, nil
	} else if len(v1) < len(v2) {
		return -1, nil
	}

	// Versions are equal
	return 0, nil
}

// parseVersion parses a version string (e.g., "4.10.0") into a slice of integers
func parseVersion(version string) []int {
	parts := strings.Split(version, ".")
	result := make([]int, 0, len(parts))

	for _, part := range parts {
		// Handle version parts like "4.10.0-rc.1" by taking only the number before the dash
		if strings.Contains(part, "-") {
			part = strings.Split(part, "-")[0]
		}

		// Convert to integer
		num, err := strconv.Atoi(part)
		if err != nil {
			// If we can't convert to integer, assume 0
			num = 0
		}

		result = append(result, num)
	}

	// Ensure we have at least major and minor versions
	for len(result) < 2 {
		result = append(result, 0)
	}

	return result
}

// compareInts compares two integers and returns:
// -1 if a < b
// 0 if a == b
// 1 if a > b
func compareInts(a, b int) (int, error) {
	if a < b {
		return -1, nil
	} else if a > b {
		return 1, nil
	}

	return 0, nil
}
