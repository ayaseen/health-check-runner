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

package utils

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
	"log"
	"net/url"
	"os/exec"
	"strings"
)

//func main() {
//	username, password, err := GetCredentials()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	server, err := GetServer()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	datacenter, err := GetDatacenter()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Use the retrieved values as needed
//	fmt.Println("Username:", username)
//	fmt.Println("Password:", password)
//	fmt.Println("Server:", server)
//	fmt.Println("Datacenter:", datacenter)
//
//	// Get the Kubernetes client
//	clientset, err := GetKubernetesClient()
//	if err != nil {
//		log.Fatalf("Failed to get Kubernetes client: %v", err)
//	}
//
//	// Get the OpenShift master nodes
//	masterNodes, err := GetOpenShiftMasterNodes(clientset)
//	if err != nil {
//		log.Fatalf("Failed to get OpenShift master nodes: %v", err)
//	}
//
//	// Check if each master node is on a separate ESXi host
//	eachOnSeparateHost, err := IsEachMasterNodeOnSeparateESXiHost(server, username, password, datacenter, masterNodes)
//	if err != nil {
//		log.Fatalf("Failed to check master nodes on separate ESXi hosts: %v", err)
//	}
//
//	fmt.Printf("Each OpenShift master node is on a separate ESXi host: %v\n", eachOnSeparateHost)
//
//}

func GetCredentials() (string, string, error) {
	// Run the 'oc' command to retrieve the secret in YAML format
	cmd := exec.Command("oc", "get", "secret", "vsphere-creds", "-o", "yaml", "-n", "kube-system")
	output, _ := cmd.Output()

	// Parse the YAML output and extract the username and password fields
	data := string(output)
	username := extractField(data, "username")
	password := extractField(data, "password")

	// Decode the base64-encoded username and password
	decodedUsername, err := base64.StdEncoding.DecodeString(username)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode username: %v", err)
	}
	decodedPassword, err := base64.StdEncoding.DecodeString(password)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode password: %v", err)
	}

	return string(decodedUsername), string(decodedPassword), nil
}

func GetServer() (string, error) {
	// Run the 'oc' command to retrieve the ConfigMap in YAML format
	cmd := exec.Command("oc", "get", "configmap", "cloud-provider-config", "-o", "yaml", "-n", "openshift-config")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run 'oc' command: %v", err)
	}

	// Parse the YAML output and extract the server field
	data := string(output)
	server := extractServerField(data)

	// Append the desired prefix to the server value
	prefix := "https://"
	server = fmt.Sprintf("%s%s/sdk", prefix, server)

	// Remove double quotes from the server value
	server = strings.ReplaceAll(server, `"`, "")

	return server, nil
}

func GetDatacenter() (string, error) {
	// Run the 'oc' command to retrieve the ConfigMap in YAML format
	cmd := exec.Command("oc", "get", "configmap", "cloud-provider-config", "-o", "yaml", "-n", "openshift-config")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run 'oc' command: %v", err)
	}

	// Parse the YAML output and extract the datacenter field
	data := string(output)
	datacenter := extractDatacenterField(data)

	// Remove double quotes from the server value
	datacenter = strings.ReplaceAll(datacenter, `"`, "")

	return datacenter, nil
}

func GetDataStore() (string, error) {
	// Run the 'oc' command to retrieve the ConfigMap in YAML format
	cmd := exec.Command("oc", "get", "configmap", "cloud-provider-config", "-o", "yaml", "-n", "openshift-config")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run 'oc' command: %v", err)
	}

	// Parse the YAML output and extract the datacenter field
	data := string(output)
	datacenter := extractDataStoreField(data)

	// Remove double quotes from the server value
	datacenter = strings.ReplaceAll(datacenter, `"`, "")

	return datacenter, nil
}

func extractDataStoreField(data string) string {
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		if strings.Contains(line, "default-datastore") {
			// Extract the value after the colon (:)
			parts := strings.Split(line, "=")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1])
			}
		}
	}

	//log.Fatalf("Failed to find datastore field in the data")
	return ""
}

func ProcessVirtualMachines(vcenterURL, username, password, datacenterName string, datastore string) int {
	ctx := context.TODO()
	u, err := soap.ParseURL(vcenterURL)
	handleError(err, "Failed to parse vCenter URL")

	u.User = url.UserPassword(username, password)
	c, err := govmomi.NewClient(ctx, u, true)
	handleError(err, "Failed to connect to vCenter")

	finder := find.NewFinder(c.Client, true)

	dc, err := finder.Datacenter(ctx, datacenterName)
	handleError(err, "Failed to find datacenter")

	finder.SetDatacenter(dc)

	datastoreObj, err := finder.Datastore(ctx, datastore)
	handleError(err, "Failed to find datastore")

	vms, err := finder.VirtualMachineList(ctx, "*")
	handleError(err, "Failed to retrieve virtual machine list")

	//fmt.Println("Virtual Machines in Datastore:", datastore)
	//fmt.Println("-------------------------------")

	totalMachines := 0

	for _, vm := range vms {
		vmObj := object.NewVirtualMachine(c.Client, vm.Reference())

		var moVM mo.VirtualMachine
		err := vmObj.Properties(ctx, vmObj.Reference(), []string{"datastore", "guest"}, &moVM)
		handleError(err, "Failed to retrieve virtual machine properties")

		if containsDatastore(moVM.Datastore, datastoreObj.Reference()) {
			//fmt.Println("Name:", vmObj.Name())
			if moVM.Guest != nil && moVM.Guest.HostName != "" {
				//fmt.Println("Host Name:", moVM.Guest.HostName)
				totalMachines++
			}
		}
	}

	//fmt.Println("Total Virtual Machines in Datastore:", totalMachines)
	return totalMachines
}

func handleError(err error, message string) {
	if err != nil {
		log.Fatalf("%s: %v", message, err)
	}
}

func extractDatacenterField(data string) string {
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		if strings.Contains(line, "datacenter") {
			// Extract the value after the colon (:)
			parts := strings.Split(line, "=")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1])
			}
		}
	}

	log.Fatalf("Failed to find datacenter field in the data")
	return ""
}

func extractField(data, field string) string {
	start := strings.Index(data, field+":")
	if start == -1 {
		log.Fatalf("Failed to find field '%s' in the YAML output", field)
	}

	end := strings.Index(data[start:], "\n")
	if end == -1 {
		log.Fatalf("Failed to extract value for field '%s'", field)
	}

	value := strings.TrimSpace(data[start+len(field)+1 : start+end])
	return value
}

func extractServerField(data string) string {
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		if strings.Contains(line, "server") {
			// Extract the value after the equal sign (=)
			parts := strings.Split(line, "=")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1])
			}
		}
	}

	log.Fatalf("Failed to find server field in the data")
	return ""
}

func GetClusterNameByDatacenterName(ctx context.Context, finder *find.Finder, datacenterName string) (string, error) {
	// Retrieve the datacenter
	dc, err := finder.Datacenter(ctx, datacenterName)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve datacenter '%s': %v", datacenterName, err)
	}

	// Set the datacenter as the search root for the finder
	finder.SetDatacenter(dc)

	// Retrieve the clusters within the datacenter
	clusters, err := finder.ClusterComputeResourceList(ctx, "*")
	if err != nil {
		return "", fmt.Errorf("failed to retrieve clusters in datacenter '%s': %v", datacenterName, err)
	}

	// Assume there is only one cluster in the datacenter
	if len(clusters) == 0 {
		return "", fmt.Errorf("no clusters found in datacenter '%s'", datacenterName)
	}

	cluster := clusters[0]
	return cluster.InventoryPath, nil
}

func IsHAClusterEnabled(vcenterURL, username, password, datacenterName string) (int, bool, error) {
	// Connect to vCenter
	ctx := context.TODO()
	u, err := soap.ParseURL(vcenterURL)
	if err != nil {
		return 0, false, fmt.Errorf("failed to parse vCenter URL: %v", err)
	}

	u.User = url.UserPassword(username, password)
	c, err := govmomi.NewClient(ctx, u, true)
	if err != nil {
		return 0, false, fmt.Errorf("failed to connect to vCenter: %v", err)
	}

	defer c.Logout(ctx)

	// Retrieve the datacenter
	finder := find.NewFinder(c.Client, false)
	clusterName, err := GetClusterNameByDatacenterName(ctx, finder, datacenterName)
	if err != nil {
		return 0, false, fmt.Errorf("failed to retrieve cluster name: %v", err)
	}

	// Retrieve the HA cluster
	cluster, err := finder.ClusterComputeResource(ctx, clusterName)
	if err != nil {
		return 0, false, fmt.Errorf("failed to retrieve cluster '%s': %v", clusterName, err)
	}

	// Check if HA is enabled
	hc, err := cluster.Configuration(ctx)
	if err != nil {
		return 0, false, fmt.Errorf("failed to retrieve HA cluster configuration: %v", err)
	}

	// Retrieve the ESXi hosts within the cluster
	hostList, err := cluster.Hosts(ctx)
	if err != nil {
		return 0, false, fmt.Errorf("failed to retrieve ESXi hosts: %v", err)
	}

	numHosts := len(hostList)

	return numHosts, *hc.DasConfig.Enabled, nil
}

func GetDatastoreStorageType(vcenterURL, username, password, datacenterName string) (string, error) {
	// Connect to vCenter
	ctx := context.TODO()
	u, err := soap.ParseURL(vcenterURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse vCenter URL: %v", err)
	}

	u.User = url.UserPassword(username, password)
	c, err := govmomi.NewClient(ctx, u, true)
	if err != nil {
		return "", fmt.Errorf("failed to connect to vCenter: %v", err)
	}

	finder := find.NewFinder(c.Client, true)

	// Find the specified datacenter
	datacenter, err := finder.Datacenter(ctx, datacenterName)
	if err != nil {
		return "", fmt.Errorf("failed to find datacenter: %v", err)
	}

	// Retrieve the datastores in the datacenter
	finder.SetDatacenter(datacenter)
	datastoreList, err := finder.DatastoreList(ctx, "*")
	if err != nil {
		return "", fmt.Errorf("failed to retrieve datastores: %v", err)
	}

	// Get the first datastore from the list
	if len(datastoreList) > 0 {
		datastore := datastoreList[0]
		storageType, _ := datastore.Type(ctx)
		return formatStorageType(string(storageType)), nil
	}

	return "", fmt.Errorf("no datastores found in the datacenter")
}

func formatStorageType(storageType string) string {
	// Convert the storage type to lowercase
	lowercaseType := strings.ToLower(storageType)

	// Capitalize the first letter
	firstLetter := strings.ToLower(string(lowercaseType[0]))

	// Combine the first letter with the remaining string in uppercase
	uppercaseType := strings.ToUpper(lowercaseType[1:])

	return firstLetter + uppercaseType
}

func CheckHostNetworkingType(vcenterURL, username, password, datacenterName string) (string, error) {
	ctx := context.TODO()
	u, err := soap.ParseURL(vcenterURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse vCenter URL: %v", err)
	}

	u.User = url.UserPassword(username, password)
	c, err := govmomi.NewClient(ctx, u, true)
	if err != nil {
		return "", fmt.Errorf("failed to connect to vCenter: %v", err)
	}

	finder := find.NewFinder(c.Client)

	// Use the finder to retrieve the datacenter
	_, err = finder.Datacenter(ctx, datacenterName)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve datacenter: %v", err)
	}

	// Retrieve the host system list under the datacenter
	hosts, err := finder.HostSystemList(ctx, "*")
	if err != nil {
		return "", fmt.Errorf("failed to retrieve hosts: %v", err)
	}

	pc := property.DefaultCollector(c.Client)

	for _, host := range hosts {
		var hostProps mo.HostSystem
		err = pc.RetrieveOne(ctx, host.Reference(), []string{"network"}, &hostProps)
		if err != nil {
			return "", fmt.Errorf("failed to retrieve host properties: %v", err)
		}

		for _, network := range hostProps.Network {
			var networkProps mo.Network
			err = pc.RetrieveOne(ctx, network, []string{"summary"}, &networkProps)
			if err != nil {
				return "", fmt.Errorf("failed to retrieve network properties: %v", err)
			}

			networkType := networkProps.Summary.GetNetworkSummary().Network
			if networkType != nil {
				networkTypeString := networkType.String()

				// Find the index of the first colon ":"
				colonIndex := strings.Index(networkTypeString, ":")
				if colonIndex != -1 {
					networkTypeString = strings.TrimSpace(networkTypeString[:colonIndex])
				} else {
					return "", fmt.Errorf("failed to extract network type")
				}

				return networkTypeString, nil
			}
		}
	}

	return "", fmt.Errorf("no network type found")
}

func GetVmwareVersion(vCenterURL, username, password string) (string, error) {
	// Connect to vCenter
	ctx := context.TODO()
	u, err := soap.ParseURL(vCenterURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse vCenter URL: %v", err)
	}

	u.User = url.UserPassword(username, password)
	c, err := govmomi.NewClient(ctx, u, true)
	if err != nil {
		return "", fmt.Errorf("failed to connect to vCenter: %v", err)
	}

	// Retrieve vSphere version
	about := c.ServiceContent.About
	ver := fmt.Sprintf("%s.%s", about.Version, about.Build)

	return ver, nil
}

// Helper function to check if a given datastore is in the list of datastores
func containsDatastore(datastores []types.ManagedObjectReference, target types.ManagedObjectReference) bool {
	for _, datastore := range datastores {
		if datastore == target {
			return true
		}
	}
	return false
}
