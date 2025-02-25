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
	"github.com/schollz/progressbar/v3"
	"log"
	"os"
	"time"
)

var provider string

// Define a struct to hold a check function and its name
type check struct {
	name string
	fn   func()
}

// Main function to call sub lists
var (
	functions = []check{
		{"All cluster operators are available", checkCO},
		{"Etcd Health", etcdHealth},
		//{"Cluster Version", clusterVersion},
		{"Cluster CNI Plugin", clusterCNIPlubin},
		{"Elevated Privileges", checkElevatedPrivileges},
		{"All nodes are Ready", nodeStatus},
		{"Logging Health", checkLoggingHealth},
		{"VMware Version", vmwareVersion},
		{"ETCD Encryption Type", etcdEncryption},
		//{"vCenter Highly Available", vmwareHA},
		{"Installation Type", installationType},
		{"EmptyDir Volumes not in use", emptyDirVolume},
		{"Kubeadmin user is absent", defaultOpenShiftUser},
		{"LimitRange is configured", resourceLimit},
		{"Infra nodes are available", checkInfraNode},
		{"Default Node Schedule", defaultNodeSchedule},
		{"ResourceQuota is configured", resourceQuota},
		{"Self Provisioners", selfProvisioners},
		{"Control Node Schedule", controlNodeSchedule},
		{"Project template is configured", defaultProject},
		{"ServiceMonitors is configured", serviceMonitor},
		{"OpenShift Proxy setting is set", proxySettings},
		{"Default Ingress Certificate", defaultIngressCertificate},
		{"Infrastructure Provider", infrastructureProvider},
		{"Network Policy", networkPolicy},
		{"Node Usage", nodeUsage},
		//{"VMware Network Type", vmwareNetworkType},
		//{"VMware Storage Type", vmwareStorageType},
		{"Infra Config Pool", checkInfraConfigPool},
		{"Physical Hypervisor Topology", providerTopology},
		{"User Workload Monitoring is enabled", monitoringUserWorkload},
		{"Logging is installed and configured", checkLogging},
		{"Ingress Controller Placement", ingressControllerPlacement},
		{"Liveness and Readiness are configured", applicationProbes},
		//{"VMware Nodes Placement", vmwareNodePlacement},
		//{"VMware Datastore Multitenancy", vmwareDatastoreMultitenancy},
		{"OpenShift Monitoring Storage", monitoringStack},
		{"Identity Provider", identityProvider},
		{"Default Ingress Controller Replica", ingressControllerReplica},
		{"ElasticSearch Health", CheckElasticSearch},
		{"Logging Forwarders OPS", checkLoggingForwardersOPS},
		{"Logging Forwarders Configuration", checkLoggingForwarders},
		{"Internal registry is functioning", internalRegistry},
		{"Default OpenShift ingress controller Type", IngressControllerType},
		{"Cluster Default SCC", ClusterDefaultSCC},
		{"Infra Taints Configuration", InfraTaints},
		{"Kubelet Configuration (Garbage Collection)", KubeletConfig},
		{"OpenShift alerts are forwarded to an external system", alerts},
		{"Check Logging Placement", checkLoggingPlacement},
		//{"Pod Image Pull Status", podImagePullStatus},
		//{"Etcd Backup", etcdBackup},
		//{"API Server Certificate", apiServerCertificate},
	}
)

func CheckLists() {

	// Get Infrastructure provider
	provider, _ = CheckInfrastructureProvider()

	// Create source folder if it doesn't exist
	err := CreateDirResources(DestPath)
	if err != nil {
		log.Fatal(err)
	}

	_, err = getClusterConfig()
	if err != nil {
		os.Exit(1)

	}

	err = checkOCExist()
	if err != nil {
		fmt.Println("Error: OpenShift cluster is not reachable through API")
		os.Exit(1)
	}

	ocpAccess := verifyOpenShiftAPIAccess()

	if ocpAccess == true {
		fmt.Println("Error: OpenShift cluster is not reachable through API")
		os.Exit(1)
	}

	fmt.Println("OpenShift Health Check in Progress ...")
	fmt.Println() // Ensure separation from the first check message.

	// Initialize the progress bar for the total number of checks
	bar := progressbar.NewOptions(len(functions),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetWidth(50),
		progressbar.OptionSetDescription("Initializing..."),
		progressbar.OptionShowCount(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerPadding: " ",
			BarStart:      "|",
			BarEnd:        "|",
		}),
	)

	// Render the initial state of the progress bar
	_ = bar.RenderBlank()

	for _, check := range functions {
		// Update the progress bar's description to include the current check name
		bar.Describe(fmt.Sprintf("[green]\033[1m%s\033[22m[reset] check in Progress ...", check.name))

		// Refresh the progress bar to show the updated description immediately
		_ = bar.RenderBlank()

		check.fn() // Execute the check function

		_ = bar.Add(1) // Increment the progress bar by one for each check

		time.Sleep(100 * time.Millisecond) // Simulate check duration
	}

	fmt.Println("\nAll checks completed!") // Final message after all checks

	// Compress resource folder and protect a random password
	Compress(DestPath, CompressFile)

}
