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
	"os/exec"
)

func Applicationchecklists() {
	getpods()
}

func getpods() {
	output, err := exec.Command("oc", "get", "pods", "-A").Output()
	if err != nil {
		fmt.Println("Error: Failed to retrieve cluster role binding information")
		return
	}

	if string(output) == "" {
		fmt.Println("no pods found!")
		return
	}

	fmt.Println("Total pods in the cluster: %d", string(output))
}
