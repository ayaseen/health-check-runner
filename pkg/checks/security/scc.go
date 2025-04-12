package security

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

const (
	defaultSCCName = "restricted"
	// This is a simplified version of the default SCC for comparison
	defaultSCCKeyFields = `allowHostDirVolumePlugin: false
allowHostIPC: false
allowHostNetwork: false
allowHostPID: false
allowHostPorts: false
allowPrivilegeEscalation: true
allowPrivilegedContainer: false
requiredDropCapabilities:
- KILL
- MKNOD
- SETUID
- SETGID`
)

// ClusterDefaultSCCCheck checks if the default security context constraint has been modified
type ClusterDefaultSCCCheck struct {
	healthcheck.BaseCheck
}

// NewClusterDefaultSCCCheck creates a new default SCC check
func NewClusterDefaultSCCCheck() *ClusterDefaultSCCCheck {
	return &ClusterDefaultSCCCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"cluster-default-scc",
			"Default Security Context Constraint",
			"Checks if the default security context constraint has been modified",
			healthcheck.CategorySecurity,
		),
	}
}

// Run executes the health check
func (c *ClusterDefaultSCCCheck) Run() (healthcheck.Result, error) {
	// Get Kubernetes config
	config, err := utils.GetClusterConfig()
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to get cluster config",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error getting cluster config: %v", err)
	}

	// Create dynamic client
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to create Kubernetes client",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error creating Kubernetes client: %v", err)
	}

	// Define the SCC resource
	sccGVR := schema.GroupVersionResource{
		Group:    "security.openshift.io",
		Version:  "v1",
		Resource: "securitycontextconstraints",
	}

	// Get the restricted SCC
	ctx := context.Background()
	scc, err := client.Resource(sccGVR).Get(ctx, defaultSCCName, metav1.GetOptions{})
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to retrieve restricted SCC",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error retrieving restricted SCC: %v", err)
	}

	// Get the SCC as YAML for detailed output
	detailedOut, err := utils.RunCommand("oc", "get", "scc", defaultSCCName, "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed SCC configuration"
	}

	// Get the important fields from the SCC to compare
	sccData := scc.Object

	// Check key fields that shouldn't be modified
	modified := false
	modifiedFields := []string{}

	// Check allowHostDirVolumePlugin
	if val, ok := sccData["allowHostDirVolumePlugin"].(bool); ok && val {
		modified = true
		modifiedFields = append(modifiedFields, "allowHostDirVolumePlugin")
	}

	// Check allowHostIPC
	if val, ok := sccData["allowHostIPC"].(bool); ok && val {
		modified = true
		modifiedFields = append(modifiedFields, "allowHostIPC")
	}

	// Check allowHostNetwork
	if val, ok := sccData["allowHostNetwork"].(bool); ok && val {
		modified = true
		modifiedFields = append(modifiedFields, "allowHostNetwork")
	}

	// Check allowHostPID
	if val, ok := sccData["allowHostPID"].(bool); ok && val {
		modified = true
		modifiedFields = append(modifiedFields, "allowHostPID")
	}

	// Check allowHostPorts
	if val, ok := sccData["allowHostPorts"].(bool); ok && val {
		modified = true
		modifiedFields = append(modifiedFields, "allowHostPorts")
	}

	// Check allowPrivilegedContainer
	if val, ok := sccData["allowPrivilegedContainer"].(bool); ok && val {
		modified = true
		modifiedFields = append(modifiedFields, "allowPrivilegedContainer")
	}

	// If not modified, return OK
	if !modified {
		result := healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusOK,
			"Default security context constraint (restricted) has not been modified",
			healthcheck.ResultKeyNoChange,
		)
		result.Detail = detailedOut
		return result, nil
	}

	// Create result with modified SCC information
	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusWarning,
		fmt.Sprintf("Default security context constraint (restricted) has been modified: %s", strings.Join(modifiedFields, ", ")),
		healthcheck.ResultKeyRecommended,
	)

	result.AddRecommendation("Do not modify the default SCCs, as they may be reset during cluster upgrades")
	result.AddRecommendation("Instead, create custom SCCs for specific needs")
	result.AddRecommendation("Follow the documentation at https://docs.openshift.com/container-platform/latest/authentication/managing-security-context-constraints.html")

	result.Detail = detailedOut

	return result, nil
}

// ElevatedPrivilegesCheck checks for workloads with elevated privileges
type ElevatedPrivilegesCheck struct {
	healthcheck.BaseCheck
}

// NewElevatedPrivilegesCheck creates a new elevated privileges check
func NewElevatedPrivilegesCheck() *ElevatedPrivilegesCheck {
	return &ElevatedPrivilegesCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"elevated-privileges",
			"Elevated Privileges",
			"Checks for workloads running with elevated privileges",
			healthcheck.CategorySecurity,
		),
	}
}

// Run executes the health check
func (c *ElevatedPrivilegesCheck) Run() (healthcheck.Result, error) {
	// Get the output of a complex command that finds pods with privileged containers
	out, err := utils.RunCommand("oc", "get", "pods", "--all-namespaces", "-o", "json")
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to get pod information",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error getting pod information: %v", err)
	}

	// This would normally involve a complex parsing of the JSON to find privileged containers
	// For simplicity, we'll use a grep command to find containers with privileged security context
	privilegedPodsOut, err := RunCommandWithPipe("echo", out, "grep", "-A 5", "privileged: true")
	if err != nil {
		// This might not be a critical error, as it could just mean no privileged pods exist
		if strings.Contains(err.Error(), "exit status 1") && privilegedPodsOut == "" {
			return healthcheck.NewResult(
				c.ID(),
				healthcheck.StatusOK,
				"No pods with elevated privileges found",
				healthcheck.ResultKeyNoChange,
			), nil
		}

		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusCritical,
			"Failed to find privileged pods",
			healthcheck.ResultKeyRequired,
		), fmt.Errorf("error finding privileged pods: %v", err)
	}

	// If there are no matches, the check passes
	if privilegedPodsOut == "" {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusOK,
			"No pods with elevated privileges found",
			healthcheck.ResultKeyNoChange,
		), nil
	}

	// Get additional information about the pods with elevated privileges
	namespacesCmd := "echo \"" + privilegedPodsOut + "\" | grep -B 5 'namespace' | grep -o 'namespace\": \"[^\"]*' | cut -d'\"' -f3 | sort | uniq"
	namespacesOut, err := RunCommandWithShell(namespacesCmd)
	if err != nil {
		// Non-critical error, we can continue
		namespacesOut = "Failed to get namespaces"
	}

	// Create table of affected namespaces
	namespaces := strings.Split(strings.TrimSpace(namespacesOut), "\n")
	var privilegedNamespaces []string
	for _, ns := range namespaces {
		if ns != "" && !strings.HasPrefix(ns, "openshift-") && ns != "kube-system" {
			privilegedNamespaces = append(privilegedNamespaces, ns)
		}
	}

	// If only system namespaces have privileged pods, the check passes
	if len(privilegedNamespaces) == 0 {
		return healthcheck.NewResult(
			c.ID(),
			healthcheck.StatusOK,
			"Only system pods have elevated privileges",
			healthcheck.ResultKeyNoChange,
		), nil
	}

	// Create result with elevated privileges information
	result := healthcheck.NewResult(
		c.ID(),
		healthcheck.StatusWarning,
		fmt.Sprintf("Found user workloads with elevated privileges in %d namespaces", len(privilegedNamespaces)),
		healthcheck.ResultKeyRecommended,
	)

	result.AddRecommendation("Review and remove elevated privileges from user workloads")
	result.AddRecommendation("Use restrictive SCCs for user workloads")
	result.AddRecommendation("Follow the principle of least privilege")

	// Add detailed information
	detail := fmt.Sprintf("Namespaces with privileged workloads:\n%s\n\nDetailed information:\n%s",
		strings.Join(privilegedNamespaces, "\n"),
		privilegedPodsOut)

	result.Detail = detail

	return result, nil
}

// GetSecurityChecks returns security-related health checks (renamed from GetChecks)
func GetSecurityChecks() []healthcheck.Check {
	var checks []healthcheck.Check

	// Add default SCC check
	checks = append(checks, NewClusterDefaultSCCCheck())

	// Add elevated privileges check
	checks = append(checks, NewElevatedPrivilegesCheck())

	// Add other security checks here
	checks = append(checks, NewEtcdEncryptionCheck())
	checks = append(checks, NewEtcdBackupCheck())
	checks = append(checks, NewEtcdHealthCheck())

	return checks
}

// RunCommandWithPipe runs a command with the output of another command as input
func RunCommandWithPipe(cmd1Name string, cmd1Input string, cmd2Name string, cmd2Args ...string) (string, error) {
	// Create a command with the input as stdin
	cmd1 := exec.Command(cmd1Name, cmd1Input)

	// Create the second command
	cmd2 := exec.Command(cmd2Name, cmd2Args...)

	// Connect the first command's output to the second command's input
	pipeReader, pipeWriter, err := os.Pipe()
	if err != nil {
		return "", fmt.Errorf("failed to create pipe: %v", err)
	}

	cmd1.Stdout = pipeWriter
	cmd2.Stdin = pipeReader

	// Prepare to capture the output of cmd2
	var output bytes.Buffer
	cmd2.Stdout = &output

	// Start the first command
	if err := cmd1.Start(); err != nil {
		return "", fmt.Errorf("failed to start first command: %v", err)
	}

	// Start the second command
	if err := cmd2.Start(); err != nil {
		return "", fmt.Errorf("failed to start second command: %v", err)
	}

	// Wait for both commands to complete
	if err := cmd1.Wait(); err != nil {
		return "", fmt.Errorf("first command failed: %v", err)
	}

	pipeWriter.Close()

	if err := cmd2.Wait(); err != nil {
		return "", fmt.Errorf("second command failed: %v", err)
	}

	return output.String(), nil
}

// RunCommandWithShell runs a command through the shell
func RunCommandWithShell(command string) (string, error) {
	// Run the command through the shell
	cmd := exec.Command("sh", "-c", command)

	// Prepare to capture the output
	var output bytes.Buffer
	cmd.Stdout = &output

	// Run the command
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("shell command failed: %v", err)
	}

	return output.String(), nil
}
