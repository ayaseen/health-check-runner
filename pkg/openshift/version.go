/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06

This application performs health checks on OpenShift to provide visibility into various functionalities. It verifies the following aspects:

- OpenShift configurations: Verify OpenShift configuration meets the standard and best practices.
- Security: It examines the security measures in place, such as authentication and authorization configurations.
- Application Probes: It tests the health and readiness probes of deployed applications to ensure they are functioning correctly.
- Resource Usage: It monitors resource consumption of OpenShift clusters, including CPU, memory, and storage.

The purpose of this application is to provide administrators and developers with an overview of OpenShift's health and functionality, helping them identify potential issues and ensure the smooth operation of their OpenShift environment.
*/

package openshift

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

var latestOpenShiftVersion, latestRelease string

func FetchLatestOpenShiftVersion() {
	url := "https://mirror.openshift.com/pub/openshift-v4/clients/ocp/latest/release.txt"

	// Send an HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Find the index of "Version:"
	index := strings.Index(string(body), "Version:")
	if index == -1 {
		fmt.Println("Version not found")
		return
	}

	// Find the index of the line break after "Version:"
	lineBreak := strings.Index(string(body)[index:], "\n")
	if lineBreak == -1 {
		fmt.Println("Version line not found")
		return
	}

	// Extract the version line
	versionLine := string(body)[index+len("Version:") : index+lineBreak]

	// Store the latest OpenShift version
	latestOpenShiftVersion = strings.TrimSpace(versionLine)

	latestRelease = latestOpenShiftVersion

}

func init() {
	// Fetch and store the latest OpenShift version during the build process
	FetchLatestOpenShiftVersion()

}

func compareVersions(version1, version2 string) (int, error) {
	// Directly compare if version1 is "4.14" or greater
	if isGreaterThanOrEqual(version1, "4.14") {
		return 0, nil
	}

	// Compare the two versions normally if version1 is less than "4.14"
	v1, v2, err := parseVersions(version1, version2)
	if err != nil {
		return 0, err
	}

	for i := 0; i < len(v1); i++ {
		if v1[i] < v2[i] {
			return -1, nil
		} else if v1[i] > v2[i] {
			return 1, nil
		}
	}
	return 0, nil // Versions are equal
}

// isGreaterThanOrEqual checks if a version is greater than or equal to a target version.
func isGreaterThanOrEqual(version, target string) bool {
	v1, v2, err := parseVersions(version, target)
	if err != nil {
		return false
	}

	for i := 0; i < len(v1); i++ {
		if v1[i] > v2[i] {
			return true
		} else if v1[i] < v2[i] {
			return false
		}
	}
	return true // Versions are equal
}

// parseVersions parses and converts the version strings into slices of integers.
func parseVersions(version1, version2 string) ([]int, []int, error) {
	v1Components := strings.Split(version1, ".")
	v2Components := strings.Split(version2, ".")

	v1 := make([]int, len(v1Components))
	v2 := make([]int, len(v2Components))

	var err error
	for i := 0; i < len(v1Components); i++ {
		v1[i], err = strconv.Atoi(v1Components[i])
		if err != nil {
			return nil, nil, fmt.Errorf("invalid version format: %s", version1)
		}
	}

	for i := 0; i < len(v2Components); i++ {
		v2[i], err = strconv.Atoi(v2Components[i])
		if err != nil {
			return nil, nil, fmt.Errorf("invalid version format: %s", version2)
		}
	}
	return v1, v2, nil
}
