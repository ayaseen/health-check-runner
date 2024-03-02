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
	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/client-go/config/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var clusterOperators, availability string

var allAvailable bool

var co configv1.ClusterOperator

// Check cluster operators status
func checkCO() {

	config, err := getClusterConfig()
	if err != nil {
		fmt.Println("Error getting cluster config:", err)
		os.Exit(1)
	}

	client, err := versioned.NewForConfig(config)
	if err != nil {
		fmt.Println("Error creating OpenShift client:", err)

	}

	// Get the list of cluster operators
	coClient := client.ConfigV1()
	cos, err := coClient.ClusterOperators().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Error retrieving cluster operators: %v\n", err)

	}

	// Check if all cluster operators are available
	allAvailable = true

	for _, co = range cos.Items {

		for _, condition := range co.Status.Conditions {
			if condition.Type == configv1.OperatorAvailable {
				availability = string(condition.Status)
				break
			}

		}
		if availability != "True" {
			allAvailable = false

			break
		}
	}

	// Print the result
	//if allAvailable {
	//	color.Green("All cluster operators are available\t\t\tPASSED")
	//} else {
	//	for _, co := range cos.Items {
	//
	//		for _, condition := range co.Status.Conditions {
	//			if condition.Type == configv1.OperatorAvailable {
	//				availability = string(condition.Status)
	//				break
	//			}
	//		}
	//		if availability == "False" {
	//
	//			clusterOperators += "\n\n|===\n|Operator Name | Available\n\n|" + co.Name + "\n|" + availability + "\n|===\n"
	//		}
	//	}
	//	color.Red("All cluster operators are available\t\t\tFAILED")
	//}

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return
	}
	//// Open the input file for reading
	//file, err := os.Open(filepath.Join(dir, "content/cluster_operators.item"))
	//if err != nil {
	//	fmt.Println("Error opening file:", err)
	//	return
	//}
	//defer file.Close()

	// Create the output file for writing
	outfile, err := os.Create(filepath.Join(dir, "resources/healthcheck-body.adoc"))
	if err != nil {
		fmt.Println("Error creating output file:", err)
		return
	}
	defer outfile.Close()

	// Read each line of the input file
	scanner := bufio.NewScanner(strings.NewReader(docTPL))
	for scanner.Scan() {
		line := scanner.Text()
		outfile.WriteString(coResultsProcess(line))
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error scanning file:", err)
		return
	}

}

func getCO() (string, error) {

	// Get node status
	out, err := exec.Command("oc", "get", "co").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func coResultsProcess(line string) string {

	//version, err := getOpenShiftVersion()
	//if err != nil {
	//	return line + " Error getting OpenShift version: " + err.Error() + "\n"
	//}

	// To populate Key Summary section
	if strings.HasPrefix(line, "<<Cluster Operators>>") {
		if allAvailable == false {
			return line + "\n\n| Some Operator required actions \n\n" + GetKeyChanges("advisory")
		} else {
			return line + "\n\n| All CO operators Ready \n\n" + GetKeyChanges("nochange")
		}
	}

	// To populate body section
	if strings.HasPrefix(line, "== Cluster Operators") {
		getOperators, _ := getCO()
		version, _ := getOpenShiftVersion()

		//Render the change status
		if allAvailable == false {
			return line + "\n\n" + GetChanges("advisory") + "\n\n[source, bash]\n----\n" + getOperators + "\n----\n" +
				"\n\n**Observation**\n\nCheck the reasons why Cluster Operators are failing. \n\n" +
				"**Recommendation**\n\nCheck the reasons why Cluster Operators are failing. \n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version

		} else {
			return line + "\n\n" + GetChanges("nochange") + "\n\n[source, bash]\n----\n" + getOperators + "\n----\n" +
				"\n\n**Observation**\n\nNone\n\n" +
				"**Recommendation**\n\nNone\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version

		}
	}
	return line + "\n"
}
