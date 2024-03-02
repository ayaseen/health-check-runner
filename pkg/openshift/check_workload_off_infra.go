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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os"
	"path/filepath"
	"strings"
)

var hasTaint bool
var nodeName string

func InfraTaints() {
	config, err := getClusterConfig()
	if err != nil {
		fmt.Println("Error getting cluster config:", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("Failed to create clientset: %v\n", err)
		os.Exit(1)
	}

	nodeName = findInfraNode(clientset)
	if nodeName != "" {
		hasTaint = checkTaints(clientset, nodeName)
		if hasTaint {
			//color.Green("Infrastructure node with required taints found\t\tPASSED")
		}

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
		output.WriteString(infraTaintsProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}

}

func infraTaintsProcess(line string) string {
	// To populate Key Summary section
	if strings.HasPrefix(line, "<<Adequate Measures in place to keep customer workloads off Infra Nodes>>") {
		if nodeName != "" {
			// Render the change status
			if hasTaint {
				return line + "\n\n| Infrastructure node with required taints configured \n\n" + GetKeyChanges("nochange")
			} else {
				return line + "\n\n| Infrastructure node with required taints not configured \n\n" + GetKeyChanges("required")
			}
		} else {
			return line + "\n\n| Infrastructure node with required taints not configured \n\n" + GetKeyChanges("required")
		}

	}

	if strings.HasPrefix(line, "== Adequate Measures in place to keep customer workloads off Infra Nodes") {

		if nodeName != "" {
			// Render the change status
			if hasTaint {
				return line + "\n\n" + GetChanges("nochange") +
					"\n\n**Observation**\n\nInfrastructure node with required taints configured to keep workload off infra nodes.. \n\n" +
					"**Recommendation**\n\nNone \n\n" +
					"*Reference Link(s)*\n\n* https://access.redhat.com/solutions/5034771\n"

			} else {
				return line + "\n\n" + GetChanges("required") +
					"\n\n**Observation**\n\nNo configuration or incomplete configuration is done to exclude application workloads from being scheduled in Infrastructure Nodes." +
					" This could lead to breach to subscription contract. \n\n" +
					"**Recommendation**\n\nLabel and taint nodes designated for infrastructure." +
					" Ensure components such as the router, logging, and monitoring stacks have appropriate tolerances for these taints.\n\n" +
					"*Reference Link(s)*\n\n* https://access.redhat.com/solutions/5034771\n"
			}
		} else {
			return line + "\n\n" + GetChanges("required") +
				"\n\n**Observation**\n\nNo configuration or incomplete configuration is done to exclude application workloads from being scheduled in Infrastructure Nodes." +
				" This could lead to breach to subscription contract. \n\n" +
				"**Recommendation**\n\nLabel and taint nodes designated for infrastructure." +
				" Ensure components such as the router, logging, and monitoring stacks have appropriate tolerances for these taints.\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/solutions/5034771\n"
		}

	}
	return line + "\n"
}

func findInfraNode(clientset *kubernetes.Clientset) string {
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Failed to list nodes: %v\n", err)
		os.Exit(1)
	}

	for _, node := range nodes.Items {
		labels := node.GetLabels()
		if _, ok := labels["node-role.kubernetes.io/infra"]; ok {
			return node.Name
		}
	}

	return ""
}

func checkTaints(clientset *kubernetes.Clientset, nodeName string) bool {
	node, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	if err != nil {
		fmt.Printf("Failed to get node: %v\n", err)
		os.Exit(1)
	}

	for _, taint := range node.Spec.Taints {
		if taint.Key == "infra" && taint.Value == "reserved" {
			return true
		}
	}

	return false
}
