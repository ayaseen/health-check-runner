/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for Security Context Constraints (SCCs). It:

- Verifies that the default 'restricted' SCC hasn't been modified
- Examines key security fields that impact container isolation
- Compares current SCC configuration against secure baseline values
- Provides recommendations for SCC management best practices
- Identifies potentially insecure SCC modifications

These checks help maintain a secure container runtime environment by ensuring proper security constraints are in place.
*/

package security

import (
	"context"
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/types"
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
			types.CategorySecurity,
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
			types.StatusCritical,
			"Failed to get cluster config",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting cluster config: %v", err)
	}

	// Create dynamic client
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to create Kubernetes client",
			types.ResultKeyRequired,
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
			types.StatusCritical,
			"Failed to retrieve restricted SCC",
			types.ResultKeyRequired,
		), fmt.Errorf("error retrieving restricted SCC: %v", err)
	}

	// Get the SCC as YAML for detailed output
	detailedOut, err := utils.RunCommand("oc", "get", "scc", defaultSCCName, "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed SCC configuration"
	}

	// Create the exact format for the detail output with proper spacing
	var formattedDetailOut strings.Builder
	formattedDetailOut.WriteString("=== Security Context Constraint Analysis ===\n\n")

	// Add SCC configuration with proper formatting
	if strings.TrimSpace(detailedOut) != "" {
		formattedDetailOut.WriteString("Restricted SCC Configuration:\n[source, yaml]\n----\n")
		formattedDetailOut.WriteString(detailedOut)
		formattedDetailOut.WriteString("\n----\n\n")
	} else {
		formattedDetailOut.WriteString("Restricted SCC Configuration: No information available\n\n")
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

	// Add SCC status information
	formattedDetailOut.WriteString("=== SCC Status ===\n\n")
	if !modified {
		formattedDetailOut.WriteString("The default 'restricted' SCC has not been modified.\n\n")
		formattedDetailOut.WriteString("This maintains the intended security posture of the OpenShift cluster.\n\n")
	} else {
		formattedDetailOut.WriteString("The default 'restricted' SCC has been modified with the following changes:\n\n")
		for _, field := range modifiedFields {
			formattedDetailOut.WriteString(fmt.Sprintf("- %s\n", field))
		}
		formattedDetailOut.WriteString("\nModifying default SCCs is not recommended as they may be reset during cluster upgrades.\n\n")
	}

	// If not modified, return OK
	if !modified {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			"Default security context constraint (restricted) has not been modified",
			types.ResultKeyNoChange,
		)
		result.Detail = formattedDetailOut.String()
		return result, nil
	}

	// Create result with modified SCC information
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusWarning,
		fmt.Sprintf("Default security context constraint (restricted) has been modified: %s", strings.Join(modifiedFields, ", ")),
		types.ResultKeyRecommended,
	)

	result.AddRecommendation("Do not modify the default SCCs, as they may be reset during cluster upgrades")
	result.AddRecommendation("Instead, create custom SCCs for specific needs")
	result.AddRecommendation("Follow the documentation at https://docs.openshift.com/container-platform/latest/authentication/managing-security-context-constraints.html")

	result.Detail = formattedDetailOut.String()

	return result, nil
}
