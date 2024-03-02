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
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// var infraNodesExist bool
var infraNodeCount int

// Check cluster operators status
func checkInfraNode() {

	config, err := getClusterConfig()
	if err != nil {
		fmt.Println("Error getting cluster config:", err)
		os.Exit(1)
	}

	// Create a Kubernetes client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("Error creating Kubernetes client: %v\n", err)
		os.Exit(1)
	}

	// Get the list of nodes from the cluster
	nodes, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Error retrieving nodes: %v\n", err)
		os.Exit(1)
	}

	// Check if any nodes have the 'infra' label

	for _, node := range nodes.Items {
		labels := node.GetLabels()
		if _, ok := labels["node-role.kubernetes.io/infra"]; ok {
			infraNodeCount++
			//infraNodesExist = true
			//	fmt.Printf("Node %s has 'infra' label\n", node.Name)
		}
	}

	//if infraNodeCount > 0 {
	//	color.Green("Infra nodes are available\t\t\t\tPASSED")
	//} else {
	//	color.Red("Infra nodes are available\t\t\t\tFAILED")
	//}

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
		output.WriteString(InfraNodeResultsProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}

}

func getInfraNodes() (string, error) {

	// Get node status
	out, err := exec.Command("oc", "get", "-l", "node-role.kubernetes.io/infra").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func InfraNodeResultsProcess(line string) string {

	infraCount := strconv.Itoa(infraNodeCount)

	// To populate Key Summary section
	if strings.HasPrefix(line, "<<Infrastructure Nodes present>>") {
		if infraNodeCount > 0 {
			return line + "\n\n| Infra nodes are available, total: " + infraCount + "\n\n" + GetKeyChanges("nochange")
		} else {
			return line + "\n\n| Infra nodes not available, total: None\n\n" + GetKeyChanges("recommended")
		}
	}

	// To populate body section
	if strings.HasPrefix(line, "== Infrastructure Nodes present") {
		getInfra, _ := getInfraNodes()
		version, _ := getOpenShiftVersion()

		//Render the change status
		if infraNodeCount > 0 {
			return line + "\n\n" + GetChanges("nochange") + "\n\n[source, bash]\n----\n" + getInfra + "\n----\n" +
				"\n\n**Observation**\n\nInfrastructure nodes in place\n\n" +
				"**Recommendation**\n\nNone\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"\n\n* https://access.redhat.com/solutions/5034771[Infrastructure Nodes]"

		} else {
			return line + "\n\n" + GetChanges("recommended") + "\n\n[source, bash]\n----\n" + getInfra + "\n----\n" +
				"\n\n**Observation**\n\nInfrastructure nodes not configured. \n\n" +
				"**Recommendation**\n\nInfrastructure nodes allow customers to isolate infrastructure workloads for two primary purposes:\n\n" +
				" 1- to prevent incurring billing costs against subscription counts and\n" +
				" 2- to separate maintenance and management.\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"\n\n* https://access.redhat.com/solutions/5034771[Infrastructure Nodes]"
		}
	}
	return line + "\n"
}
