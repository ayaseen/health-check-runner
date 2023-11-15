package utils

import (
	"os/exec"
	"strings"
)

// Helper function to get the total count of OpenShift nodes
func GetOpenShiftNodesCount() (int, error) {
	cmd := exec.Command("oc", "get", "nodes", "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	// Extract the number of nodes from the output
	nodesCount := strings.Count(string(output), `"name":`)
	return nodesCount, nil
}
