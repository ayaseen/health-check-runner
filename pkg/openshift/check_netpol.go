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
	"bufio"
	"fmt"
	"github.com/fatih/color"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Check network policies
var netpolicy []byte

func networkPolicy() {

	netpolicy, err := exec.Command("oc", "get", "netpol", "-A").Output()
	if err != nil {
		fmt.Println("Error: Failed to retrieve cluster role binding information")

	}

	if string(netpolicy) == "" {
		color.Red("Network Policies are configured\t\t\tFAILED")

	} else {
		color.Green("Network Policies are configured\t\t\tPASSED")
	}

	// Create the output file for writing
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return
	}

	inputFile, err := os.Open(filepath.Join(dir, "resources/healthcheck-body.adoc"))
	if err != nil {
		panic(err)
	}
	defer inputFile.Close()

	// Open the output file
	outputFile, err := os.OpenFile("resources/healthcheck-body.adoc", os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer outputFile.Close()

	// Create a scanner to read the input file
	fileScanner := bufio.NewScanner(inputFile)

	// Create a buffer to hold the output
	var output strings.Builder

	// Process each line of the input file
	for fileScanner.Scan() {
		line := fileScanner.Text()
		output.WriteString(networkPolicyProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}
}

func checkNetworkPolicy() (string, error) {

	out, err := exec.Command("oc", "get", "netpol", "-A").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func networkPolicyProcess(line string) string {
	if strings.HasPrefix(line, "<<Network Policy>>") {

		if string(netpolicy) == "" {
			return line + "\n\n|Network Policies not enabled\n\n" + GetKeyChanges("recommended") + "\n\n"
		} else {
			return line + "\n\n|Network Policies enabled\n\n" + GetKeyChanges("nochange") + "\n\n"
		}
	}

	if strings.HasPrefix(line, "== Network Policy") {
		netpol, _ := checkNetworkPolicy()
		version, _ := getOpenShiftVersion()

		if string(netpolicy) == "" {
			return line + "\n\n" + GetChanges("recommended") + "\n[source,bash]\n----\n# oc get netpol -A\nNo resources found\n----\n" +
				"\n\n**Observation**\n\nTo isolate one or more pods in a project, you can create networkPolicy objects in that project to indicate the allowed incoming connections. \n\n" +
				"**Recommendation**\n\nDefine network policies that restrict traffic to pods in your cluster.\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/networking/index#about-network-policy[About Network Policy]\n"

		} else {
			return line + "\n\n" + GetChanges("nochange") + "\n[source,bash]\n----\n" + netpol + "\n----\n" +
				"\n\n**Observation**\n\nNetwork Policies configure in project users. \n\n" +
				"**Recommendation**\n\nNone.\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/networking/index#about-network-policy[About Network Policy]\n"

		}
	}

	return line + "\n"
}
