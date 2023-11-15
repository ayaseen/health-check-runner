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
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/alexmullins/zip"
	"io"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var OpenShiftAccess bool

const (
	DestPath     = "resources"
	CompressFile = "resources.zip"
)

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE")
}

const tpl = `= 


**Observation**


**Recommendation**


**Reference Link(s)**

* https://access.redhat.com/documentation/en-us/openshift_container_platform/4.10/`

func getClusterConfig() (*rest.Config, error) {
	home := homeDir()
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		kubeconfigPath = home + "/.kube/config"
	}

	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		fmt.Println("Kubeconfig file not found. Please check if the file exists, or export KUBECONFIG=~/kubeconfig:", kubeconfigPath)
		os.Exit(1)
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, err
		}
	}
	return config, nil
}

func getClientSet(config *rest.Config) (*kubernetes.Clientset, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

func getOpenShiftVersion() (string, error) {
	// Run the command "oc get clusterVer"
	cmd := exec.Command("oc", "get", "clusterversion")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", err
	}

	// Read the output of the command
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "version") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				version := parts[1]
				return version[:4], nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	if err := cmd.Wait(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("OpenShift version not found")
}

func verifyOpenShiftAPIAccess() bool {
	cmd := exec.Command("oc", "whoami")
	out, err := cmd.CombinedOutput()
	if err != nil {
		OpenShiftAccess = strings.Contains(string(out), "Unable to connect to the server")
	}
	return OpenShiftAccess
}

// Compress compresses the source folder to the destination zip file with a random password and deletes the source folder.
func Compress(srcPath, destPath string) string {
	// Check that source folder exists
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		fmt.Errorf("source folder '%s' does not exist: %w", srcPath, err)
	}

	if !srcInfo.IsDir() {
		errors.New("source path is not a directory")
	}

	// Generate random password for zip file
	password := make([]byte, 16)
	if _, err := rand.Read(password); err != nil {
		fmt.Errorf("failed to generate random password: %w", err)
	}
	passwordStr := fmt.Sprintf("%x", password)

	// Create zip file
	zipFile, err := os.Create(destPath)
	if err != nil {
		fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	// Create a new zip archive
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Walk through source folder and add files to zip archive
	err = filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and hidden files
		if info.IsDir() || strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Open the file for reading
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		// Create a new file header
		header := &zip.FileHeader{
			Name:   path[len(srcPath)+1:],
			Method: zip.Deflate,
		}

		// Set the password for the file
		header.SetPassword(passwordStr)

		// Create a new zip file entry
		entry, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		// Write the file contents to the zip file entry
		_, err = io.Copy(entry, file)
		if err != nil {
			return err
		}

		//if err == nil {
		//
		//	// Delete resources after it compressed
		//	DeleteFolder(DestPath)
		//}

		return nil
	})
	if err != nil {
		fmt.Errorf("failed to add files to zip archive: %w", err)
	}

	log.Printf("Compressed folder '%s' to '%s' with password '%s'", srcPath, destPath, passwordStr)

	return passwordStr
}

func DeleteFolder(source string) error {
	err := os.RemoveAll(source)
	if err != nil {
		return err
	}

	return nil
}

func CreateDirResources(destPath string) error {
	// Check if source folder exists
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		// Create source folder if it doesn't exist
		err := os.Mkdir(destPath, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}

func checkOCExist() error {
	// Check if the "oc" command is available

	// Get node status
	_, err := exec.Command("oc", "version").Output()
	if err != nil {
		return err
	}
	return nil

}

// To check if OpenShift Logging is installed
func checkLoggingConfigure() (string, error) {

	// Get the Installation method type
	out, err := exec.Command("oc", "get", "clusterlogging", "-n", "openshift-logging").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// To check if OpenShift Logging forwarder is exits
func checkLoggingForwarder() (string, error) {

	// Get the Installation method type
	out, err := exec.Command("oc", "get", "clusterLogforwarder", "-n", "openshift-logging").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func writeToFile(filename string, message string) error {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = fmt.Fprintln(file, message)
	if err != nil {
		return err
	}

	return nil
}

func CheckInfrastructureProvider() (string, error) {

	// Get the Infrastructure provider type
	out, err := exec.Command("oc", "get", "Infrastructure", "cluster", "-o", "jsonpath={.spec.platformSpec.type}").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// Render the action is require for check list
func GetChanges(option string) string {

	options := map[string]string{
		"required": `[cols="^"]
|===
|
{set:cellbgcolor:#FF0000}
Changes Required
|===`,
		"recommended": `[cols="^"]
|===
|
{set:cellbgcolor:#FEFE20}
Changes Recommended
|===`,
		"nochange": `[cols="^"]
|===
|
{set:cellbgcolor:#00FF00}
No Change
|===`,
		"advisory": `[cols="^"]
|===
|
{set:cellbgcolor:#80E5FF}
Advisory
|===`,
		"toevaluated": `[cols="^"]
|===
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated
|===`,

		"na": `[cols="^"]
|===
|
{set:cellbgcolor:#FFFFFF}
Not Applicable
|===`,
	}
	result := options[option]
	return result
}

// Render the action is require for check list
func GetKeyChanges(option string) string {

	options := map[string]string{
		"required": `|
{set:cellbgcolor:#FF0000}
Changes Required
`,
		"recommended": `|
{set:cellbgcolor:#FEFE20}
Changes Recommended
`,
		"nochange": `|
{set:cellbgcolor:#00FF00}
No Change
`,

		"advisory": `|
{set:cellbgcolor:#80E5FF}
Advisory
`,
		"na": `|
{set:cellbgcolor:#FFFFFF}
Not Applicable
`,
		"eval": `|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated
`,
	}
	result := options[option]
	return result
}

// HC Template

const docTPL = `
= Key

[cols="1,3", options=header]
|===
|Value
|Description

|
{set:cellbgcolor:#FF0000}
Changes Required
|
{set:cellbgcolor!}
Indicates Changes Required for system stability, subscription compliance, or other reason.

|
{set:cellbgcolor:#FEFE20}
Changes Recommended
|
{set:cellbgcolor!}
Indicates Changes Recommended to align with recommended practices, but not urgently required

|
{set:cellbgcolor:#A6B9BF}
N/A
|
{set:cellbgcolor!}
No advise given on line item.  For line items which are data-only to provide context.

|
{set:cellbgcolor:#80E5FF}
Advisory
|
{set:cellbgcolor!}
No change required or recommended, but additional information provided.

|
{set:cellbgcolor:#00FF00}
No Change
|
{set:cellbgcolor!}
No change required.  In alignment with recommended practices.

|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated
|
{set:cellbgcolor!}
Not yet evaluated.  Will appear only in draft copies.
|===

= Summary


[cols="1,2,2,3", options=header]
|===
|*Category*
|*Item Evaluated*
|*Observed Result*
|*Recommendation*


// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1010_compute-provider.item

// Category
|
{set:cellbgcolor!}
Infra

// Item Evaluated
a|
<<Infrastructure Provider(s)>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1020_installation-type.item

// Category
|
{set:cellbgcolor!}
Infra

// Item Evaluated
a|
<<Installation Type>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1030_if-upi-is-customer-using-additional-machinesets.item

// Category
|
{set:cellbgcolor!}
Infra

// Item Evaluated
a|
<<Check if user-provisioned infrastructure (UPI) using additional MachineSets>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1070_if-upi-what-kind-of-openshift-provisioning-automation-exists.item

// Category
|
{set:cellbgcolor!}
Infra

// Item Evaluated
a|
<<Check if user-provisioned infrastructure (UPI) OpenShift Provisioning Automation exists>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1190_node_cpu_mem_storage.item

// Category
|
{set:cellbgcolor!}
Infra

// Item Evaluated
a|
<<OpenShift Node Status>>


// ------------------------ITEM END


// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1190_node_cpu_mem_storage.item

// Category
|
{set:cellbgcolor!}
Infra

// Item Evaluated
a|
<<OpenShift Node Usage>>


// ------------------------ITEM END



// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1040_vmware-version.item

// Category
|
{set:cellbgcolor!}
vSphere

// Item Evaluated
a|
VMware vSphere Version



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1050_vcenter-highly-available.item

// Category
|
{set:cellbgcolor!}
vSphere

// Item Evaluated
a|
<<vCenter Highly Available>>


// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1060_vcenter-service-account.item

// Category
|
{set:cellbgcolor!}
vSphere

// Item Evaluated
a|
vCenter Service Account

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1080_physical-hypervisor-topology.item

// Category
|
{set:cellbgcolor!}
vSphere

// Item Evaluated
a|
<<Physical Hypervisor Topology>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1090_node-placement-on-hosts.item

// Category
|
{set:cellbgcolor!}
vSphere

// Item Evaluated
a|
<<Node Placement on Hosts>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1100_vmware-datastore-storage-provider.item

// Category
|
{set:cellbgcolor!}
vSphere

// Item Evaluated
a|
Check to see if the OpenShift Cluster is configured to use VMware datastore

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1110_vmware-ha-enabled.item

// Category
|
{set:cellbgcolor!}
vSphere

// Item Evaluated
a|
<<vSphere HA Enabled>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1120_datastore-storage-provider.item

// Category
|
{set:cellbgcolor!}
vSphere

// Item Evaluated
a|
Datastore Storage Provider



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1130_datastore-multitenancy.item

// Category
|
{set:cellbgcolor!}
vSphere

// Item Evaluated
a|
Datastore Multitenancy



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1140_vmware-networking-type.item

// Category
|
{set:cellbgcolor!}
vSphere

// Item Evaluated
a|
VMware Networking Type



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1150_vmware-backup.item

// Category
|
{set:cellbgcolor!}
vSphere

// Item Evaluated
a|
VM Backup

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1160_vmware-time-synchronization-enabled.item

// Category
|
{set:cellbgcolor!}
vSphere

// Item Evaluated
a|
VMware Time Synchronization Enabled

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1170_node-network-ranges.item

// Category
|
{set:cellbgcolor!}
Network

// Item Evaluated
a|
Node Network Ranges

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/2010_node-network-topology.item

// Category
|
{set:cellbgcolor!}
Network

// Item Evaluated
a|
Node Network Topology

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/2020_mtu-sizes.item

// Category
|
{set:cellbgcolor!}
Network

// Item Evaluated
a|
Cluster Network MTU Size

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/2030_load-balancer-type.item

// Category
|
{set:cellbgcolor!}
Network

// Item Evaluated
a|
<<Load Balancer Type>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/2040_load-balancer-health-checks-enabled.item

// Category
|
{set:cellbgcolor!}
Network

// Item Evaluated
a|
<<Load Balancer Health Checks Enabled>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/2050_load-balancer-balancing-algorithm.item

// Category
|
{set:cellbgcolor!}
Network

// Item Evaluated
a|
<<Load Balancer Balancing Algorithm>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/2051_load-balancer-ssl-passthrough.item

// Category
|
{set:cellbgcolor!}
Network

// Item Evaluated
a|
<<Load Balancer SSL Settings>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/2052_load-balancer-vips_consistently_configured.item

// Category
|
{set:cellbgcolor!}
Network

// Item Evaluated
a|
<<Load Balancer VIPs Consistently Configured>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/2060_ingress-controller-type.item

// Category
|
{set:cellbgcolor!}
Network

// Item Evaluated
a|
<<Ingress Controller Type>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/2070_ingress-controller-placement.item

// Category
|
{set:cellbgcolor!}
Network

// Item Evaluated
a|
<<Ingress Controller Placement>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/2080_ingress-controller-replica-count.item

// Category
|
{set:cellbgcolor!}
Network

// Item Evaluated
a|
<<Ingress Controller Replica Count>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/2080_ingress-controller-replica-count.item

// Category
|
{set:cellbgcolor!}
Network

// Item Evaluated
a|
<<Ingress Controller Certificate>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/2090_openshift-sdn-plugin.item

// Category
|
{set:cellbgcolor!}
Network

// Item Evaluated
a|
<<CNI Network Plugin>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/3110_provider.item

// Category
|
{set:cellbgcolor!}
Storage

// Item Evaluated
a|
Provider 1

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/3120_technology.item

// Category
|
{set:cellbgcolor!}
Storage

// Item Evaluated
a|
Technology 1

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/3130_block-or-file.item

// Category
|
{set:cellbgcolor!}
Storage

// Item Evaluated
a|
Block or File

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/3210_provider.item

// Category
|
{set:cellbgcolor!}
Storage

// Item Evaluated
a|
Provider 2

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/3220_technology.item

// Category
|
{set:cellbgcolor!}
Storage

// Item Evaluated
a|
Technology 2

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/3230_block-or-file.item

// Category
|
{set:cellbgcolor!}
Storage

// Item Evaluated
a|
Block or File

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/3310_provider.item

// Category
|
{set:cellbgcolor!}
Storage

// Item Evaluated
a|
Provider 3

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/3320_technology.item

// Category
|
{set:cellbgcolor!}
Storage

// Item Evaluated
a|
Technology 3

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/3330_block-or-file.item

// Category
|
{set:cellbgcolor!}
Storage

// Item Evaluated
a|
Block or File

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/3340_multipath-enabled.item

// Category
|
{set:cellbgcolor!}
Storage

// Item Evaluated
a|
Multipath Enabled

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/3350_IQN_Set_Correctly.item

// Category
|
{set:cellbgcolor!}
Storage

// Item Evaluated
a|
iSCSI Initiator Name

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4000_openshift_version.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Cluster Version>>



// ------------------------ITEM END
// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4000_openshift_version.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|

<<Cluster Operators>>



// ------------------------ITEM END
// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4001_masters_schedulable.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Control Nodes Schedulable>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4005_infra-nodes.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Infrastructure Nodes present>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4009_default_scc.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Default Security Context Constraint>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4010_adequate-measures-in-place-to-keep-customer-workloads-off-infra-nodes.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Adequate Measures in place to keep customer workloads off Infra Nodes>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4011_default_project_template_set.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Default Project Template Set>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4012_default_node_selector_set.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Default Node Selector Set>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4013_self_provisioner_enabled.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Self Provisioner Enabled>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4019_kubeadmin_enabled.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Kubeadmin user disabled>>



// ------------------------ITEM END


// ------------------------ITEM START


// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Network Policy>>



// ------------------------ITEM END


// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4020_identity-provider-type.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Identity Provider Type>>


// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4030_identity-provider-search-url.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Identity Provider Search URL>>


// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4040_ldap-encrypted-connection.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<LDAP Encrypted Connection>>


// ------------------------ITEM END



// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4051_image-registry-internal.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Openshift internal registry is functioning and running>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4160_user-workload-monitoring.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<User Workload Monitoring>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4170_openshift-logging-installed.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<OpenShift Logging>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4171_alternative_log_aggregation.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Alternative Log Aggregation>>



// ------------------------ITEM END



// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4250_etcd-backup-defined.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
etcd backup defined

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4251_etcd-encryption.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<ETCD Encryption Enabled>>


// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4252_etcd_disk_performance.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Etcd Performance>>



// ------------------------ITEM END



// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4260_infra-machine-configs-defined.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Infra machine config pool defined>>



// ------------------------ITEM END


// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4280_kubelet-config-overridden.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Kubelet Configuration Overridden>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4290_emptydir-volumes-in-use.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<EmptyDir Volumes in use>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4310_proxy-config.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Openshift Proxy Settings>>



// ------------------------ITEM END



// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/5010_readiness-and-liveness-probes.item

// Category
|
{set:cellbgcolor!}
App Dev

// Item Evaluated
a|
<<Readiness and Liveness Probes>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/5020_elevated-privileges.item

// Category
|
{set:cellbgcolor!}
App Dev

// Item Evaluated
a|
<<Elevated Privileges>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/5030_limit-request-configured.item

// Category
|
{set:cellbgcolor!}
App Dev

// Item Evaluated
a|
<<Resource Quotas Defined>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/5050_deployment-iac-items-versioned.item

// Category
|
{set:cellbgcolor!}
App Dev

// Item Evaluated
a|
Deployment IaC Items Versioned

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/5070_deployment-iac-type.item

// Category
|
{set:cellbgcolor!}
App Dev

// Item Evaluated
a|
Deployment IaC Type

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/5080_primary-ci-cd-tool.item

// Category
|
{set:cellbgcolor!}
App Dev

// Item Evaluated
a|
Primary CI CD Tool

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/5090_version-control.item

// Category
|
{set:cellbgcolor!}
App Dev

// Item Evaluated
a|
Version Control

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/5100_openshift-integration-method.item

// Category
|
{set:cellbgcolor!}
App Dev

// Item Evaluated
a|
OpenShift Integration Method

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/5110_container-registry-integration-method.item

// Category
|
{set:cellbgcolor!}
App Dev

// Item Evaluated
a|
Container Registry Integration Method

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END







// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/5150_container-base_images.item

// Category
|
{set:cellbgcolor!}
App Dev

// Item Evaluated
a|
Container Base Images

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/5160_disaster_recovery_deployments.item

// Category
|
{set:cellbgcolor!}
App Dev

// Item Evaluated
a|
Disaster Recovery Deployments

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END







// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/6070_rbac_is_enabled.item

// Category
|
{set:cellbgcolor!}
Security

// Item Evaluated
a|
Verify that RBAC is enabled

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END


// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/6110_garbage_collection.item

// Category
|
{set:cellbgcolor!}
Security

// Item Evaluated
a|
<<Ensure that garbage collection is configured as appropriate>>



// ------------------------ITEM END


// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/6260_client_cert_authentication.item

// Category
|
{set:cellbgcolor!}
Security

// Item Evaluated
a|
Client certificate authentication should not be used for users

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END



// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/6310_external_secret_storage.item

// Category
|
{set:cellbgcolor!}
Security

// Item Evaluated
a|
Consider external secret storage

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END



// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/6540_CICD-scanning.item

// Category
|
{set:cellbgcolor!}
Security

// Item Evaluated
a|
CI/CD integration with Security Scanning

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END




// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/7010_logging-forward-audit.item

// Category
|
{set:cellbgcolor!}
Op-Ready

// Item Evaluated
a|
<<Forward OpenShift audit logs to external logging system>>


// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/7020_logging-forward-app.item

// Category
|
{set:cellbgcolor!}
Op-Ready

// Item Evaluated
a|
<<Forward OpenShift application logs to external logging system>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/7030_logging-forward-infra.item

// Category
|
{set:cellbgcolor!}
Op-Ready

// Item Evaluated
a|
<<Forward OpenShift infrastructure logs to external logging system>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/7050_logging-healthy.item

// Category
|
{set:cellbgcolor!}
Op-Ready

// Item Evaluated
a|
<<OpenShift logging deployment is functioning and healthy>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/7060_logging-resource-contention.item

// Category
|
{set:cellbgcolor!}
Op-Ready

// Item Evaluated
a|
<<OpenShift logging Elasticsearch pods are scheduled on appropriate nodes>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/7070_logging-elastic-log-size.item

// Category
|
{set:cellbgcolor!}
Op-Ready

// Item Evaluated
a|
<<OpenShift Logging Elasticsearch storage has sufficient space>>



// ------------------------ITEM END




// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/7130_monitoring-user-apps.item

// Category
|
{set:cellbgcolor!}
Op-Ready

// Item Evaluated
a|
<<Application specific metrics are monitored on OpenShift>>



// ------------------------ITEM END



// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/7150_monitoring-alert-notify-external.item

// Category
|
{set:cellbgcolor!}
Op-Ready

// Item Evaluated
a|
<<Ensure OpenShift alerts are forwarded to an external system that is monitored>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/7160_monitoring-persistent-storage.item

// Category
|
{set:cellbgcolor!}
Op-Ready

// Item Evaluated
a|
<<Monitoring components need high performance/local persistent storage to maintain consistent state after a pod restart>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/7170_team-skills.item

// Category
|
{set:cellbgcolor!}
Op-Ready

// Item Evaluated
a|
Team Skills Operating Openshift

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END
|===

<<<

{set:cellbgcolor!}

# Infrastructure


[cols="1,2,2,3", options=header]
|===
|*Category*
|*Item Evaluated*
|*Observed Result*
|*Recommendation*


// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1010_compute-provider.item

// Category
|
{set:cellbgcolor!}
Infra

// Item Evaluated
a|
<<Infrastructure Provider(s)>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1020_installation-type.item

// Category
|
{set:cellbgcolor!}
Infra

// Item Evaluated
a|
<<Installation Type>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1030_if-upi-is-customer-using-additional-machinesets.item

// Category
|
{set:cellbgcolor!}
Infra

// Item Evaluated
a|
<<Check if user-provisioned infrastructure (UPI) using additional MachineSets>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1070_if-upi-what-kind-of-openshift-provisioning-automation-exists.item

// Category
|
{set:cellbgcolor!}
Infra

// Item Evaluated
a|
<<Check if user-provisioned infrastructure (UPI) OpenShift Provisioning Automation exists>>



// ------------------------ITEM END
// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1190_node_cpu_mem_storage.item

// Category
|
{set:cellbgcolor!}
Infra

// Item Evaluated
a|
<<OpenShift Node Status>>



// ------------------------ITEM END
// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1190_node_cpu_mem_storage.item

// Category
|
{set:cellbgcolor!}
Infra

// Item Evaluated
a|
<<OpenShift Node Usage>>



// ------------------------ITEM END

|===

<<<

== Infrastructure Provider(s)


== Installation Type


== Check if user-provisioned infrastructure (UPI) using additional MachineSets


== Check if user-provisioned infrastructure (UPI) OpenShift Provisioning Automation exists


== OpenShift Node Status


== OpenShift Node Usage


<<<

{set:cellbgcolor!}

# vSphere


[cols="1,2,2,3", options=header]
|===
|*Category*
|*Item Evaluated*
|*Observed Result*
|*Recommendation*


// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1040_vmware-version.item

// Category
|
{set:cellbgcolor!}
vSphere

// Item Evaluated
a|
VMware vSphere Version



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1050_vcenter-highly-available.item

// Category
|
{set:cellbgcolor!}
vSphere

// Item Evaluated
a|
<<vCenter Highly Available>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1060_vcenter-service-account.item

// Category
|
{set:cellbgcolor!}
vSphere

// Item Evaluated
a|
vCenter Service Account

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1080_physical-hypervisor-topology.item

// Category
|
{set:cellbgcolor!}
vSphere

// Item Evaluated
a|
<<Physical Hypervisor Topology>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1090_node-placement-on-hosts.item

// Category
|
{set:cellbgcolor!}
vSphere

// Item Evaluated
a|
<<Node Placement on Hosts>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1100_vmware-datastore-storage-provider.item

// Category
|
{set:cellbgcolor!}
vSphere

// Item Evaluated
a|
Check to see if the OpenShift Cluster is configured to use VMware datastore

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1110_vmware-ha-enabled.item

// Category
|
{set:cellbgcolor!}
vSphere

// Item Evaluated
a|
<<vSphere HA Enabled>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1120_datastore-storage-provider.item

// Category
|
{set:cellbgcolor!}
vSphere

// Item Evaluated
a|
Datastore Storage Provider



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1130_datastore-multitenancy.item

// Category
|
{set:cellbgcolor!}
vSphere

// Item Evaluated
a|
Datastore Multitenancy


// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1140_vmware-networking-type.item

// Category
|
{set:cellbgcolor!}
vSphere

// Item Evaluated
a|
VMware Networking Type



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1150_vmware-backup.item

// Category
|
{set:cellbgcolor!}
vSphere

// Item Evaluated
a|
VM Backup

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1160_vmware-time-synchronization-enabled.item

// Category
|
{set:cellbgcolor!}
vSphere

// Item Evaluated
a|
VMware Time Synchronization Enabled

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END
|===

<<<


== vCenter Highly Available


== Physical Hypervisor Topology


== Node Placement on Hosts


== vSphere HA Enabled


== Datastore Multitenancy







<<<

{set:cellbgcolor!}


# Network


[cols="1,2,2,3", options=header]
|===
|*Category*
|*Item Evaluated*
|*Observed Result*
|*Recommendation*


// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/1170_node-network-ranges.item

// Category
|
{set:cellbgcolor!}
Network

// Item Evaluated
a|
Node Network Ranges

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/2010_node-network-topology.item

// Category
|
{set:cellbgcolor!}
Network

// Item Evaluated
a|
Node Network Topology

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/2020_mtu-sizes.item

// Category
|
{set:cellbgcolor!}
Network

// Item Evaluated
a|
Cluster Network MTU Size

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/2030_load-balancer-type.item

// Category
|
{set:cellbgcolor!}
Network

// Item Evaluated
a|
<<Load Balancer Type>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/2040_load-balancer-health-checks-enabled.item

// Category
|
{set:cellbgcolor!}
Network

// Item Evaluated
a|
<<Load Balancer Health Checks Enabled>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/2050_load-balancer-balancing-algorithm.item

// Category
|
{set:cellbgcolor!}
Network

// Item Evaluated
a|
<<Load Balancer Balancing Algorithm>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/2051_load-balancer-ssl-passthrough.item

// Category
|
{set:cellbgcolor!}
Network

// Item Evaluated
a|
<<Load Balancer SSL Settings>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/2052_load-balancer-vips_consistently_configured.item

// Category
|
{set:cellbgcolor!}
Network

// Item Evaluated
a|
<<Load Balancer VIPs Consistently Configured>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/2060_ingress-controller-type.item

// Category
|
{set:cellbgcolor!}
Network

// Item Evaluated
a|
<<Ingress Controller Type>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/2070_ingress-controller-placement.item

// Category
|
{set:cellbgcolor!}
Network

// Item Evaluated
a|
<<Ingress Controller Placement>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/2080_ingress-controller-replica-count.item

// Category
|
{set:cellbgcolor!}
Network

// Item Evaluated
a|
<<Ingress Controller Replica Count>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/2080_ingress-controller-replica-count.item

// Category
|
{set:cellbgcolor!}
Network

// Item Evaluated
a|
<<Ingress Controller Certificate>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/2090_openshift-sdn-plugin.item

// Category
|
{set:cellbgcolor!}
Network

// Item Evaluated
a|
<<CNI Network Plugin>>



// ------------------------ITEM END
|===

<<<


== Load Balancer Type


== Load Balancer Health Checks Enabled


== Load Balancer Balancing Algorithm


== Load Balancer SSL Settings


== Load Balancer VIPs Consistently Configured


== Ingress Controller Type


== Ingress Controller Placement


== Ingress Controller Replica Count


== Ingress Controller Certificate


== CNI Network Plugin


<<<

{set:cellbgcolor!}

# Storage


[cols="1,2,2,3", options=header]
|===
|*Category*
|*Item Evaluated*
|*Observed Result*
|*Recommendation*


// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/3110_provider.item

// Category
|
{set:cellbgcolor!}
Storage

// Item Evaluated
a|
Provider 1

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/3120_technology.item

// Category
|
{set:cellbgcolor!}
Storage

// Item Evaluated
a|
Technology 1

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/3130_block-or-file.item

// Category
|
{set:cellbgcolor!}
Storage

// Item Evaluated
a|
Block or File

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/3210_provider.item

// Category
|
{set:cellbgcolor!}
Storage

// Item Evaluated
a|
Provider 2

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/3220_technology.item

// Category
|
{set:cellbgcolor!}
Storage

// Item Evaluated
a|
Technology 2

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/3230_block-or-file.item

// Category
|
{set:cellbgcolor!}
Storage

// Item Evaluated
a|
Block or File

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/3310_provider.item

// Category
|
{set:cellbgcolor!}
Storage

// Item Evaluated
a|
Provider 3

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/3320_technology.item

// Category
|
{set:cellbgcolor!}
Storage

// Item Evaluated
a|
Technology 3

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/3330_block-or-file.item

// Category
|
{set:cellbgcolor!}
Storage

// Item Evaluated
a|
Block or File

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/3340_multipath-enabled.item

// Category
|
{set:cellbgcolor!}
Storage

// Item Evaluated
a|
Multipath Enabled

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/3350_IQN_Set_Correctly.item

// Category
|
{set:cellbgcolor!}
Storage

// Item Evaluated
a|
iSCSI Initiator Name

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END
|===

<<<

<<<

{set:cellbgcolor!}

# OpenShift Cluster Config


[cols="1,2,2,3", options=header]
|===
|*Category*
|*Item Evaluated*
|*Observed Result*
|*Recommendation*


// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4000_openshift_version.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Cluster Version>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4000_openshift_version.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Cluster Operators>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4001_masters_schedulable.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Control Nodes Schedulable>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4005_infra-nodes.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Infrastructure Nodes present>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4009_default_scc.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Default Security Context Constraint>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4010_adequate-measures-in-place-to-keep-customer-workloads-off-infra-nodes.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Adequate Measures in place to keep customer workloads off Infra Nodes>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4011_default_project_template_set.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Default Project Template Set>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4012_default_node_selector_set.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Default Node Selector Set>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4013_self_provisioner_enabled.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Self Provisioner Enabled>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4019_kubeadmin_enabled.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Kubeadmin user disabled>>



// ------------------------ITEM END

// ------------------------ITEM START


// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Network Policy>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4020_identity-provider-type.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Identity Provider Type>>


// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4030_identity-provider-search-url.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Identity Provider Search URL>>


// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4040_ldap-encrypted-connection.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<LDAP Encrypted Connection>>


// ------------------------ITEM END


// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4051_image-registry-internal.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Openshift internal registry is functioning and running>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4160_user-workload-monitoring.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<User Workload Monitoring>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4170_openshift-logging-installed.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<OpenShift Logging>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4171_alternative_log_aggregation.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Alternative Log Aggregation>>



// ------------------------ITEM END



// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4250_etcd-backup-defined.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
etcd backup defined

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4251_etcd-encryption.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<ETCD Encryption Enabled>>




// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4252_etcd_disk_performance.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Etcd Performance>>


// ------------------------ITEM END



// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4260_infra-machine-configs-defined.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Infra machine config pool defined>>



// ------------------------ITEM END


// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4280_kubelet-config-overridden.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Kubelet Configuration Overridden>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4290_emptydir-volumes-in-use.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<EmptyDir Volumes in use>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/4310_proxy-config.item

// Category
|
{set:cellbgcolor!}
Cluster Config

// Item Evaluated
a|
<<Openshift Proxy Settings>>



// ------------------------ITEM END
|===

<<<


== Cluster Version


== Cluster Operators


== Control Nodes Schedulable


== Infrastructure Nodes present


== Default Security Context Constraint


== Adequate Measures in place to keep customer workloads off Infra Nodes


== Default Project Template Set


== Default Node Selector Set


== Self Provisioner Enabled


== Kubeadmin user disabled


== Network Policy


== Identity Provider Type


== Identity Provider Search URL


== Openshift internal registry is functioning and running


== User Workload Monitoring


== OpenShift Logging


== Alternative Log Aggregation


== ETCD Encryption Enabled


== Etcd Performance


== Infra machine config pool defined


== Kubelet Configuration Overridden


== EmptyDir Volumes in use


== Openshift Proxy Settings



<<<

{set:cellbgcolor!}

# Application Development


[cols="1,2,2,3", options=header]
|===
|*Category*
|*Item Evaluated*
|*Observed Result*
|*Recommendation*




// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/5010_readiness-and-liveness-probes.item

// Category
|
{set:cellbgcolor!}
App Dev

// Item Evaluated
a|
<<Readiness and Liveness Probes>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/5020_elevated-privileges.item

// Category
|
{set:cellbgcolor!}
App Dev

// Item Evaluated
a|
<<Elevated Privileges>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/5030_limit-request-configured.item

// Category
|
{set:cellbgcolor!}
App Dev

// Item Evaluated
a|
<<Resource Quotas Defined>>


// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/5050_deployment-iac-items-versioned.item

// Category
|
{set:cellbgcolor!}
App Dev

// Item Evaluated
a|
Deployment IaC Items Versioned

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/5070_deployment-iac-type.item

// Category
|
{set:cellbgcolor!}
App Dev

// Item Evaluated
a|
Deployment IaC Type

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/5080_primary-ci-cd-tool.item

// Category
|
{set:cellbgcolor!}
App Dev

// Item Evaluated
a|
Primary CI CD Tool

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/5090_version-control.item

// Category
|
{set:cellbgcolor!}
App Dev

// Item Evaluated
a|
Version Control

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/5100_openshift-integration-method.item

// Category
|
{set:cellbgcolor!}
App Dev

// Item Evaluated
a|
OpenShift Integration Method

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/5110_container-registry-integration-method.item

// Category
|
{set:cellbgcolor!}
App Dev

// Item Evaluated
a|
Container Registry Integration Method

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END







// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/5150_container-base_images.item

// Category
|
{set:cellbgcolor!}
App Dev

// Item Evaluated
a|
Container Base Images

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/5160_disaster_recovery_deployments.item

// Category
|
{set:cellbgcolor!}
App Dev

// Item Evaluated
a|
Disaster Recovery Deployments

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END
|===

<<<

== Readiness and Liveness Probes


== Elevated Privileges


== Resource Quotas Defined



<<<
{set:cellbgcolor!}

# Security


[cols="1,2,2,3", options=header]
|===
|*Category*
|*Item Evaluated*
|*Observed Result*
|*Recommendation*



// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/6070_rbac_is_enabled.item

// Category
|
{set:cellbgcolor!}
Security

// Item Evaluated
a|
Verify that RBAC is enabled

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END


// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/6110_garbage_collection.item

// Category
|
{set:cellbgcolor!}
Security

// Item Evaluated
a|
<<Ensure that garbage collection is configured as appropriate>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/6260_client_cert_authentication.item

// Category
|
{set:cellbgcolor!}
Security

// Item Evaluated
a|
Client certificate authentication should not be used for users

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/6310_external_secret_storage.item

// Category
|
{set:cellbgcolor!}
Security

// Item Evaluated
a|
Consider external secret storage

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END


// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/6540_CICD-scanning.item

// Category
|
{set:cellbgcolor!}
Security

// Item Evaluated
a|
CI/CD integration with Security Scanning

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END



|===

<<<


== Ensure that garbage collection is configured as appropriate






{set:cellbgcolor!}

# Operational Readiness


[cols="1,2,2,3", options=header]
|===
|*Category*
|*Item Evaluated*
|*Observed Result*
|*Recommendation*


// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/7010_logging-forward-audit.item

// Category
|
{set:cellbgcolor!}
Op-Ready

// Item Evaluated
a|
<<Forward OpenShift audit logs to external logging system>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/7020_logging-forward-app.item

// Category
|
{set:cellbgcolor!}
Op-Ready

// Item Evaluated
a|
<<Forward OpenShift application logs to external logging system>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/7030_logging-forward-infra.item

// Category
|
{set:cellbgcolor!}
Op-Ready

// Item Evaluated
a|
<<Forward OpenShift infrastructure logs to external logging system>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/7050_logging-healthy.item

// Category
|
{set:cellbgcolor!}
Op-Ready

// Item Evaluated
a|
<<OpenShift logging deployment is functioning and healthy>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/7060_logging-resource-contention.item

// Category
|
{set:cellbgcolor!}
Op-Ready

// Item Evaluated
a|
<<OpenShift logging Elasticsearch pods are scheduled on appropriate nodes>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/7070_logging-elastic-log-size.item

// Category
|
{set:cellbgcolor!}
Op-Ready

// Item Evaluated
a|
<<OpenShift Logging Elasticsearch storage has sufficient space>>



// ------------------------ITEM END




// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/7130_monitoring-user-apps.item

// Category
|
{set:cellbgcolor!}
Op-Ready

// Item Evaluated
a|
<<Application specific metrics are monitored on OpenShift>>



// ------------------------ITEM END



// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/7150_monitoring-alert-notify-external.item

// Category
|
{set:cellbgcolor!}
Op-Ready

// Item Evaluated
a|
<<Ensure OpenShift alerts are forwarded to an external system that is monitored>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/7160_monitoring-persistent-storage.item

// Category
|
{set:cellbgcolor!}
Op-Ready

// Item Evaluated
a|
<<Monitoring components need high performance/local persistent storage to maintain consistent state after a pod restart>>



// ------------------------ITEM END

// ------------------------ITEM START
// ----ITEM SOURCE:  ./content/healthcheck-items/7170_team-skills.item

// Category
|
{set:cellbgcolor!}
Op-Ready

// Item Evaluated
a|
Team Skills Operating Openshift

// Result
|


// Recommendation
|
{set:cellbgcolor:#FFFFFF}
To Be Evaluated

// ------------------------ITEM END
|===

<<<

== Forward OpenShift audit logs to external logging system


== Forward OpenShift application logs to external logging system


== Forward OpenShift infrastructure logs to external logging system


== OpenShift logging deployment is functioning and healthy


== OpenShift logging Elasticsearch pods are scheduled on appropriate nodes


== OpenShift Logging Elasticsearch storage has sufficient space


== Application specific metrics are monitored on OpenShift


== Ensure OpenShift alerts are forwarded to an external system that is monitored


== Monitoring components need high performance/local persistent storage to maintain consistent state after a pod restart








// Reset bgcolor for future tables
[grid=none,frame=none]
|===
|{set:cellbgcolor!}
|===

`
