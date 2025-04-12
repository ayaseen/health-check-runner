/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06

This applications performs health checks on OpenShift to provide visibility into various functionalities. It verifies the following aspects:

- OpenShift configurations: Verify OpenShift configuration meets the standard and best practices.
- Security: It examines the security measures in place, such as authentication and authorization configurations.
- Application Probes: It tests the health and readiness probes of deployed applications to ensure they are functioning correctly.
- Resource Usage: It monitors resource consumption of OpenShift clusters, including CPU, memory, and storage.

The purpose of this applications is to provide administrators and developers with an overview of OpenShift's health and functionality, helping them identify potential issues and ensure the smooth operation of their OpenShift environment.
*/

package utils

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

func CompareOpenShiftVersion() bool {
	version, err := getOpenShiftVersion()
	if err != nil {
		fmt.Println("Error:", err)
		return false
	}

	major, minor := extractVersionParts(version)

	if major > 4 || (major == 4 && minor >= 11) {
		return true
	}

	return false
}

func extractVersionParts(version string) (int, int) {
	parts := strings.Split(version, ".")
	var major, minor int

	if len(parts) >= 1 {
		major = parseInt(parts[0])
	}
	if len(parts) >= 2 {
		minor = parseInt(parts[1])
	}

	return major, minor
}

func parseInt(str string) int {
	var num int
	_, err := fmt.Sscanf(str, "%d", &num)
	if err != nil {
		// If conversion fails, return 0
		return 0
	}
	return num
}

func getOpenShiftVersion() (string, error) {
	cmd := exec.Command("oc", "get", "clusterversion")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "version") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1][:4], nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	if err := cmd.Wait(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("OpenShift version not found")
}
