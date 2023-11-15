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
	components1 := strings.Split(version1, ".")
	components2 := strings.Split(version2, ".")

	// Ensure that both versions have three components
	if len(components1) != 3 || len(components2) != 3 {
		return 0, fmt.Errorf("invalid version format")
	}

	for i := 0; i < 3; i++ {
		v1, err := strconv.Atoi(components1[i])
		if err != nil {
			return 0, fmt.Errorf("invalid version format: %s", version1)
		}

		v2, err := strconv.Atoi(components2[i])
		if err != nil {
			return 0, fmt.Errorf("invalid version format: %s", version2)
		}

		if v1 < v2 {
			return -1, nil
		} else if v1 > v2 {
			return 1, nil
		}
	}

	return 0, nil // Both versions are equal
}
