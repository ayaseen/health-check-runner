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
	"context"
	"fmt"
	"github.com/fatih/color"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var workerNodesLabeled bool

func providerTopology() {
	config, err := getClusterConfig()
	if err != nil {
		fmt.Println("Error getting cluster config:", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Println("Error creating client:", err)
		os.Exit(1)
	}

	// Get the list of nodes
	nodes, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Error retrieving node list: %v", err)
	}

	// Check if not all worker nodes have the label

	for _, node := range nodes.Items {
		if node.Labels["node-role.kubernetes.io/worker"] != "" {
			if _, ok := node.Labels["topology.kubernetes.io/zone"]; ok {
				workerNodesLabeled = true
			} else {
				workerNodesLabeled = false
				break
			}
		}
	}

	// Print the result
	if provider == "VSphere" {
		if workerNodesLabeled {
			color.Green("Physical Hypervisor Topology is set\t\t\tPASSED")
		} else {
			color.Red("Physical Hypervisor Topology is set\t\t\tFAILED")
		}
	} else {
		color.Red("Physical Hypervisor Topology is set\t\t\tFAILED")
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
	scanner := bufio.NewScanner(inputFile)

	// Create a buffer to hold the output
	var output strings.Builder

	// Process each line of the input file
	for scanner.Scan() {
		line := scanner.Text()
		output.WriteString(providerTopologyProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}

}

func providerTopologyProcess(line string) string {
	// To populate Key Summary section
	if strings.HasPrefix(line, "<<Physical Hypervisor Topology>>") {
		// Render the change status
		if workerNodesLabeled && provider == "VSphere" {
			return line + "\n\n| Physical Hypervisor Topology is set \n\n" + GetKeyChanges("nochange")
		} else {
			return line + "\n\n| Physical Hypervisor Topology is not set \n\n" + GetKeyChanges("recommended")
		}
	}

	if strings.HasPrefix(line, "== Physical Hypervisor Topology") {

		// Render the change status
		if provider == "VSphere" {
			if workerNodesLabeled {
				return line + "\n\n" + GetChanges("nochange") +
					"\n\n**Observation**\n\nAll nodes are Ready. \n\n" +
					"**Recommendation**\n\nNone \n\n"

			} else {
				return line + "\n\n" + GetChanges("recommended") +
					"\n\n**Observation**\n\nESXi hosts are not equally distributed across three failure domains. To ensure a better resilience in case of a datacenter partial disaster recovery," +
					" cusomter should ensure an equal distribution of the ESXi hosts in the failure domains. \n\n" +
					"**Recommendation**\n\nTroubleshoot why node is not Ready. \n\n" +
					"*Reference Link(s)*\n\n* https://cloud.redhat.com/blog/a-guide-to-implementing-failure-domains-with-openshift-workloads-on-vmware[Implementing Failure Domains with OpenShift Workloads on VMware]"

			}
		} else {
			return line + "\n\n" + GetChanges("na") +
				"\n\n**Observation**\n\n Not Applicable\n\n" +
				"**Recommendation**\n\nNone\n\n"
		}

	}
	return line + "\n"
}
