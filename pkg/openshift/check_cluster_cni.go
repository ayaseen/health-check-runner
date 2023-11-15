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

var clusterCNI string

func clusterCNIPlubin() {

	// Get the encryption type of the etcd server
	out, err := exec.Command("oc", "get", "network.config", "-o", "jsonpath={.items[*].spec.networkType}").Output()
	if err != nil {
		fmt.Printf("Error getting cluster version: %v", err)

	}
	clusterCNI = strings.TrimSpace(string(out))

	if clusterCNI != "" {
		color.HiCyan("Cluster CNI Type \t\t\t\t\tCHECKED")

	}

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
		output.WriteString(ClusterCNITypeProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}

}

func checkClusterCNIType() (string, error) {

	// Get the CNI Type
	out, err := exec.Command("oc", "get", "network.config", "-o", "yaml").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func ClusterCNITypeProcess(line string) string {

	if strings.HasPrefix(line, "<<CNI Network Plugin>>") {
		// Render the change status
		if clusterCNI != "OVNKubernetes" {
			return line + "\n\n| " + clusterCNI + "\n\n" + GetKeyChanges("recommended")
		} else {
			return line + "\n\n| " + clusterCNI + "\n\n" + GetKeyChanges("nochange")
		}
	}

	if strings.HasPrefix(line, "== CNI Network Plugin") {
		clusterCNINetwork, _ := checkClusterCNIType()

		version, _ := getOpenShiftVersion()

		if clusterCNI != "OVNKubernetes" {
			return line + "\n\n" + GetChanges("recommended") + "\n\n[source, yaml]\n----\n" + clusterCNINetwork + "\n----\n" +
				"\n\n**Observation**\n\nOVN Kubernetes is the default plugin at latest releases." +
				" Recommend to migrate to OVN Kubernetes.\n\n" +
				"**Recommendation**\n\nOpenShift CNI Network Plugin is \n\n" + clusterCNI + ", it recommended to utilize OVN CNI based for more features utilization.\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/networking/index#openshift-sdn-default-cni-network-provider[CNI Network Provider]\n"
		} else {
			return line + "\n\n" + GetChanges("nochange") + "\n\n[source, yaml]\n----\n" + clusterCNINetwork + "\n----\n" +
				"\n\n**Observation**\n\n OpenShift CNI Network Plugin is " + clusterCNI + "\n\n" +
				"**Recommendation**\n\nNone. \n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/networking/index#openshift-sdn-default-cni-network-provider[CNI Network Provider]\n"
		}
	}
	return line + "\n"
}
