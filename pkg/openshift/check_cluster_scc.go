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
	"github.com/openshift/client-go/config/clientset/versioned"
	"os"
	"os/exec"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
)

var sccStatus bool

const (
	defaultSCCName = "restricted"
	sccDefault     = `allowHostDirVolumePlugin: false
allowHostIPC: false
allowHostNetwork: false
allowHostPID: false
allowHostPorts: false
allowPrivilegeEscalation: true
allowPrivilegedContainer: false
allowedCapabilities: null
apiVersion: security.openshift.io/v1
defaultAddCapabilities: null
fsGroup:
  type: MustRunAs
groups: []
kind: SecurityContextConstraints
metadata:
  name: restricted
priority: null
readOnlyRootFilesystem: false
requiredDropCapabilities:
- KILL
- MKNOD
- SETUID
- SETGID
runAsUser:
  type: MustRunAsRange
seLinuxContext:
  type: MustRunAs
supplementalGroups:
  type: RunAsAny
users: []
volumes:
- configMap
- csi
- downwardAPI
- emptyDir
- ephemeral
- persistentVolumeClaim
- projected
- secret`
)

// Custom type that includes all the fields in the SCC
type SecurityContextConstraints struct {
	AllowHostDirVolumePlugin bool              `yaml:"allowHostDirVolumePlugin"`
	AllowHostIPC             bool              `yaml:"allowHostIPC"`
	AllowHostNetwork         bool              `yaml:"allowHostNetwork"`
	AllowHostPID             bool              `yaml:"allowHostPID"`
	AllowHostPorts           bool              `yaml:"allowHostPorts"`
	AllowPrivilegeEscalation bool              `yaml:"allowPrivilegeEscalation"`
	AllowPrivilegedContainer bool              `yaml:"allowPrivilegedContainer"`
	AllowedCapabilities      []string          `yaml:"allowedCapabilities"`
	APIVersion               string            `yaml:"apiVersion"`
	DefaultAddCapabilities   []string          `yaml:"defaultAddCapabilities"`
	FSGroup                  map[string]string `yaml:"fsGroup"`
	Groups                   []string          `yaml:"groups"`
	Kind                     string            `yaml:"kind"`
	Metadata                 struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Priority                 string            `yaml:"priority"`
	ReadOnlyRootFilesystem   bool              `yaml:"readOnlyRootFilesystem"`
	RequiredDropCapabilities []string          `yaml:"requiredDropCapabilities"`
	RunAsUser                map[string]string `yaml:"runAsUser"`
	SELinuxContext           map[string]string `yaml:"seLinuxContext"`
	SupplementalGroups       map[string]string `yaml:"supplementalGroups"`
	Users                    []string          `yaml:"users"`
	Volumes                  []string          `yaml:"volumes"`
}

// Compare the SCC objects
func compareSCC(current, expected []byte) bool {
	var currentSCC, expectedSCC SecurityContextConstraints

	// Unmarshal the YAML into the structs
	if err := yaml.Unmarshal(current, &currentSCC); err != nil {
		fmt.Println("Failed to unmarshal current SCC:", err)
		return false
	}
	if err := yaml.Unmarshal(expected, &expectedSCC); err != nil {
		fmt.Println("Failed to unmarshal expected SCC:", err)
		return false
	}

	// Compare each field in the SCC structs
	if currentSCC.AllowHostDirVolumePlugin != expectedSCC.AllowHostDirVolumePlugin ||
		currentSCC.AllowHostIPC != expectedSCC.AllowHostIPC ||
		currentSCC.AllowHostNetwork != expectedSCC.AllowHostNetwork ||
		currentSCC.AllowHostPID != expectedSCC.AllowHostPID ||
		currentSCC.AllowHostPorts != expectedSCC.AllowHostPorts ||
		currentSCC.AllowPrivilegeEscalation != expectedSCC.AllowPrivilegeEscalation ||
		currentSCC.AllowPrivilegedContainer != expectedSCC.AllowPrivilegedContainer ||
		!compareStringSlices(currentSCC.AllowedCapabilities, expectedSCC.AllowedCapabilities) ||
		currentSCC.APIVersion != expectedSCC.APIVersion ||
		!compareStringSlices(currentSCC.DefaultAddCapabilities, expectedSCC.DefaultAddCapabilities) ||
		!compareStringMaps(currentSCC.FSGroup, expectedSCC.FSGroup) ||
		!compareStringSlices(currentSCC.Groups, expectedSCC.Groups) ||
		currentSCC.Kind != expectedSCC.Kind ||
		currentSCC.Metadata.Name != expectedSCC.Metadata.Name ||
		currentSCC.Priority != expectedSCC.Priority ||
		currentSCC.ReadOnlyRootFilesystem != expectedSCC.ReadOnlyRootFilesystem ||
		!compareStringSlices(currentSCC.RequiredDropCapabilities, expectedSCC.RequiredDropCapabilities) ||
		!compareStringMaps(currentSCC.RunAsUser, expectedSCC.RunAsUser) ||
		!compareStringMaps(currentSCC.SELinuxContext, expectedSCC.SELinuxContext) ||
		!compareStringMaps(currentSCC.SupplementalGroups, expectedSCC.SupplementalGroups) ||
		!compareStringSlices(currentSCC.Users, expectedSCC.Users) ||
		!compareStringSlices(currentSCC.Volumes, expectedSCC.Volumes) {
		return false
	}

	return true
}

// Compare string slices
func compareStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// Compare string maps
func compareStringMaps(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}

	for k, v := range a {
		if bVal, ok := b[k]; !ok || bVal != v {
			return false
		}
	}

	return true
}

func ClusterDefaultSCC() {

	config, err := getClusterConfig()
	if err != nil {
		fmt.Println("Error getting cluster config:", err)
		os.Exit(1)
	}

	client, err := versioned.NewForConfig(config)
	if err != nil {
		fmt.Println("Error creating OpenShift client:", err)

	}

	// Retrieve the current SCC
	currentSCC, err := client.RESTClient().Get().
		AbsPath("apis/security.openshift.io/v1/securitycontextconstraints/" + defaultSCCName).
		DoRaw(context.TODO())
	if err != nil {
		fmt.Println("Failed to retrieve current SCC:", err)
		return
	}

	// Compare the current SCC with the default SCC
	if compareSCC(currentSCC, []byte(sccDefault)) {
		color.Green("Default Security Context Constraint unchanged\t\tPASSED")
		sccStatus = true
	} else {
		color.Red("Default Security Context Constraint unchanged\t\tFAILED")
		sccStatus = false
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
		output.WriteString(clusterDefaultSCCProcess(line))
	}

	// Write the output to the output file
	_, err = outputFile.WriteString(output.String())
	if err != nil {
		panic(err)
	}

}

func checkClusterDefaultSCC() (string, error) {

	// Get the Default SCC restricted
	out, err := exec.Command("oc", "get", "scc", "restricted", "-o", "yaml").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func clusterDefaultSCCProcess(line string) string {

	if strings.HasPrefix(line, "<<Default Security Context Constraint>>") {
		// Render the change status
		if sccStatus != true {
			return line + "\n\n| SCC not modified \n\n" + GetKeyChanges("recommended")
		} else {
			return line + "\n\n| SCC has not been modified \n\n" + GetKeyChanges("nochange")
		}
	}

	if strings.HasPrefix(line, "== Default Security Context Constraint") {
		defaultSCC, _ := checkClusterDefaultSCC()

		version, _ := getOpenShiftVersion()

		if sccStatus != true {
			return line + "\n\n" + GetChanges("recommended") + "\n\n[source, yaml]\n----\n" + defaultSCC + "\n----\n" +
				"\n\n**Observation**\n\nOpenShift Default Security Context Constraint has been changed\n\n" +
				"**Recommendation**\n\nDo not modify the default SCCs. Customizing the default SCCs can lead to issues when some of the platform pods deploy or OpenShift Container Platform is upgraded." +
				"Additionally, the default SCC values are reset to the defaults during some cluster upgrades, which discards all customizations to those SCCs." +
				"\n\nInstead of modifying the default SCCs, create and modify your own SCCs as needed. For detailed steps, see Creating security context constraints.\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/authentication_and_authorization/index#managing-pod-security-policies[Default Security Context Constraint]\n"
		} else {
			return line + "\n\n" + GetChanges("nochange") + "\n\n[source, yaml]\n----\n" + defaultSCC + "\n----\n" +
				"\n\n**Observation**\n\nOpenShift Default Security Context Constraint has not been changed\n\n" +
				"**Recommendation**\n\nDo not modify the default SCCs. Customizing the default SCCs can lead to issues when some of the platform pods deploy or OpenShift Container Platform is upgraded." +
				" Additionally, the default SCC values are reset to the defaults during some cluster upgrades, which discards all customizations to those SCCs." +
				"\n\nInstead of modifying the default SCCs, create and modify your own SCCs as needed. For detailed steps, see Creating security context constraints.\n\n" +
				"*Reference Link(s)*\n\n* https://access.redhat.com/documentation/en-us/openshift_container_platform/" + version +
				"/html-single/authentication_and_authorization/index#managing-pod-security-policies[Default Security Context Constraint]\n"
		}
	}
	return line + "\n"
}
