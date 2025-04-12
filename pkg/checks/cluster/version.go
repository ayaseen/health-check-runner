package cluster

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"gitlab.consulting.redhat.com/meta/health-check-runner/pkg/healthcheck"
)

// ClusterVersionCheck checks if the cluster is running the latest version
type ClusterVersionCheck struct {
	healthcheck.BaseCheck
	latestVersion string
}

// NewClusterVersionCheck creates a new cluster version check
func NewClusterVersionCheck(latestVersion string) *ClusterVersionCheck {
	return &ClusterVersionCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"cluster-version",
			"Cluster Version",
			"Checks if the cluster is running the latest version of OpenShift",
			healthcheck.CategoryCluster,
		),
		latestVersion: latestVersion,
	}
}

// Run executes the health check
func (c *ClusterVersionCheck) Run() (healthcheck.Result, error) {
	// Get the current version of OpenShift
	out, err := exec.Command("oc", "get", "clusterversion", "-o", "jsonpath={.items[].status.history[].version}").Output()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to get cluster version",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error getting cluster version: %v", err)
	}

	currentVersion := strings.TrimSpace(string(out))

	// If no current version is found, return an error
	if currentVersion == "" {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"No cluster version found",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("no cluster version found")
	}

	// Get detailed cluster version information
	detailedOut, err := exec.Command("oc", "get", "clusterversion", "-o", "yaml").Output()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to get detailed cluster version",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error getting detailed cluster version: %v", err)
	}

	// Compare versions
	result, err := compareVersions(currentVersion, c.latestVersion)
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to compare versions",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error comparing versions: %v", err)
	}

	if result < 0 {
		// Current version is older than latest version
		checkResult := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusWarning,
			fmt.Sprintf("Cluster version %s is not the latest version (%s)", currentVersion, c.latestVersion),
			healthcheck.ResultKeyRecommended,
		)

		checkResult.AddRecommendation(fmt.Sprintf("Update to the latest version %s", c.latestVersion))
		checkResult.AddRecommendation("Follow the upgrade documentation at https://docs.openshift.com/container-platform/latest/updating/updating-cluster.html")
		checkResult.WithDetail(string(detailedOut))

		return checkResult, nil
	}

	// Current version is the latest version or newer
	checkResult := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusOK,
		fmt.Sprintf("Cluster version %s is up to date", currentVersion),
		healthcheck.ResultKeyNoChange,
	)

	checkResult.WithDetail(string(detailedOut))

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
