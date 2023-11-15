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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Node struct {
	Name    string
	Status  string
	Role    string
	Age     string
	Version string
}

var role string
var ready string

func nodeStatus() {
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

	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Println("Error listing nodes:", err)

	}

	nodeList := make([]Node, 0)
	for _, node := range nodes.Items {

		for _, taint := range node.Spec.Taints {
			switch taint.Key {
			case "node-role.kubernetes.io/master":
				role = "master"
			case "infra":
				role = "infra,worker"
			default:
				role = "worker"

			}

		}

		status := string(node.Status.Conditions[len(node.Status.Conditions)-1].Status)
		ready = status

		nodeList = append(nodeList, Node{
			Name:    node.Name,
			Status:  status,
			Role:    role,
			Age:     time.Since(node.CreationTimestamp.Time).String(),
			Version: node.Status.NodeInfo.KubeletVersion,
		})

	}

	if ready == "True" {
		color.Green("All nodes are Ready\t\t\t\t\tPASSED")

	} else {
		color.Red("All nodes are Ready\t\t\t\t\tFAILED")

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
		output.WriteString(nodeStatusProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}

}

func checkNodeStatus() (string, error) {

	// Get node status
	out, err := exec.Command("oc", "get", "nodes").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func nodeStatusProcess(line string) string {
	// To populate Key Summary section
	if strings.HasPrefix(line, "<<OpenShift Node Status>>") {
		// Render the change status
		if ready == "True" {
			return line + "\n\n| None \n\n" + GetKeyChanges("nochange")
		} else {
			return line + "\n\n| Some node not Ready \n\n" + GetKeyChanges("advisory")
		}
	}

	if strings.HasPrefix(line, "== OpenShift Node Status") {
		nodeStatus, _ := checkNodeStatus()
		version, _ := getOpenShiftVersion()

		// Render the change status
		if ready == "True" {
			return line + "\n\n" + GetChanges("nochange") + "\n\n[source, bash]\n----\n" + nodeStatus + "\n----\n" +
				"\n\n**Observation**\n\nAll nodes are Ready. \n\n" +
				"**Recommendation**\n\nNone \n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version

		} else {
			return line + "\n\n" + GetChanges("advisory") + "\n\n[source, bash]\n----\n" + nodeStatus + "\n----\n" +
				"\n\n**Observation**\n\nSome node not Ready. \n\n" +
				"**Recommendation**\n\nTroubleshoot why node is not Ready. \n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version

		}

	}
	return line + "\n"
}
