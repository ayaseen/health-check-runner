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
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/vim25/soap"
	"gitlab.consulting.redhat.com/meta/health-check-runner/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

var eachOnSeparateHost bool

func vmwareNodePlacement() {
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

	if provider == "VSphere" {
		// Get the OpenShift master nodes
		masterNodes, _ := GetOpenShiftMasterNodes(clientset)
		username, password, _ := utils.GetCredentials()
		vCenterURL, _ := utils.GetServer()
		dataCenter, _ := utils.GetDatacenter()

		// Check if each master node is on a separate ESXi host
		eachOnSeparateHost, _ = IsEachMasterNodeOnSeparateESXiHost(vCenterURL, username, password, dataCenter, masterNodes)

		if eachOnSeparateHost == true {
			color.Green("Master node is on a separate ESXi host\t\t\tPASSED")
		} else {
			color.Red("Master node is on a separate ESXi host\t\t\tFAILED")
		}
	} else {
		color.HiCyan("Master node is on a separate ESXi host\t\t\tSKIPPED")
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
		output.WriteString(vmwareNodePlacementProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}

}

func vmwareNodePlacementProcess(line string) string {
	// To populate Key Summary section
	if strings.HasPrefix(line, "<<Node Placement on Hosts>>") {
		// Render the change status
		if eachOnSeparateHost == true && provider == "VSphere" {
			return line + "\n\n| Master node is on a separate ESXi host \n\n" + GetKeyChanges("nochange")
		} else {
			return line + "\n\n| Master node is not on a separate ESXi host \n\n" + GetKeyChanges("recommended")
		}
	}

	if strings.HasPrefix(line, "== Node Placement on Hosts") {
		version, _ := getOpenShiftVersion()

		// Render the change status
		if provider == "VSphere" {
			if eachOnSeparateHost == true {
				return line + "\n\n" + GetChanges("nochange") +
					"\n\n**Observation**\n\nMaster node is on a separate ESXi host \n\n" +
					"**Recommendation**\n\nNone \n\n"

			} else {
				return line + "\n\n" + GetChanges("recommended") +
					"\n\n**Observation**\n\nMaster nodes are on the same ESXi hosts or on different ESXi hosts not distributed across failure domains\n\n" +
					"**Recommendation**\n\nDistributed OpenShift master nodes across failure domains. \n\n" +
					"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
					"html-single/nodes/index#nodes-scheduler-about[Controlling pod placement using node taints]\n"

			}
		} else {
			return line + "\n\n" + GetChanges("na") +
				"\n\n**Observation**\n\n Not Applicable\n\n" +
				"**Recommendation**\n\nNone\n\n"
		}

	}
	return line + "\n"
}

func GetOpenShiftMasterNodes(clientset *kubernetes.Clientset) ([]corev1.Node, error) {
	// Retrieve all nodes
	nodes, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve nodes: %v", err)
	}

	var masterNodes []corev1.Node

	// Filter the nodes to include only master nodes
	for _, node := range nodes.Items {
		for _, taint := range node.Spec.Taints {
			if taint.Key == "node-role.kubernetes.io/master" && taint.Value == "true" {
				masterNodes = append(masterNodes, node)
				break
			}
		}
	}

	return masterNodes, nil
}

func IsEachMasterNodeOnSeparateESXiHost(vcenterURL, username, password, datacenterName string, masterNodes []corev1.Node) (bool, error) {
	// Connect to vCenter
	ctx := context.TODO()
	u, err := soap.ParseURL(vcenterURL)
	if err != nil {
		return false, fmt.Errorf("failed to parse vCenter URL: %v", err)
	}

	u.User = url.UserPassword(username, password)
	c, err := govmomi.NewClient(ctx, u, true)
	if err != nil {
		return false, fmt.Errorf("failed to connect to vCenter: %v", err)
	}

	defer c.Logout(ctx)

	// Retrieve the datacenter
	finder := find.NewFinder(c.Client, false)
	dc, err := finder.Datacenter(ctx, datacenterName)
	if err != nil {
		return false, fmt.Errorf("failed to retrieve datacenter '%s': %v", datacenterName, err)
	}

	// Set the datacenter as the search root for the finder
	finder.SetDatacenter(dc)

	// Retrieve the ESXi hosts within the datacenter
	hosts, err := finder.HostSystemList(ctx, "*")
	if err != nil {
		return false, fmt.Errorf("failed to retrieve ESXi hosts: %v", err)
	}

	// Create a map to track the number of master nodes on each ESXi host
	masterNodeCounts := make(map[string]int)

	// Iterate over the master nodes and check their ESXi host
	for _, node := range masterNodes {
		nodeHost := node.ObjectMeta.Labels["kubernetes.io/hostname"]

		// Find the corresponding ESXi host in the list
		found := false
		for _, host := range hosts {
			if host.Name() == nodeHost {
				masterNodeCounts[host.Name()]++
				found = true
				break
			}
		}

		if !found {
			return false, fmt.Errorf("failed to find ESXi host for master node '%s'", node.Name)
		}
	}

	// Check if any ESXi host has more than one master node
	for _, count := range masterNodeCounts {
		if count > 1 {
			return false, fmt.Errorf("multiple master nodes found on the same ESXi host")
		}
	}

	return true, nil
}
