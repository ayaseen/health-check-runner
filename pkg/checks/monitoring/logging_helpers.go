package monitoring

import (
	"github.com/ayaseen/health-check-runner/pkg/utils"
	"strings"
)

// LoggingType represents the type of logging configuration detected
type LoggingType string

const (
	// LoggingTypeNone indicates no logging is installed
	LoggingTypeNone LoggingType = "None"
	// LoggingTypeTraditional indicates traditional OpenShift logging with Elasticsearch
	LoggingTypeTraditional LoggingType = "Traditional"
	// LoggingTypeLoki indicates OpenShift logging with Loki
	LoggingTypeLoki LoggingType = "Loki"
)

// LoggingInfo holds information about the detected logging configuration
type LoggingInfo struct {
	// Type indicates the type of logging detected
	Type LoggingType
	// HasExternalForwarder indicates if logs are forwarded to an external system
	HasExternalForwarder bool
}

// DetectLoggingConfiguration checks which logging configuration is active
func DetectLoggingConfiguration() (LoggingInfo, error) {
	info := LoggingInfo{
		Type:                 LoggingTypeNone,
		HasExternalForwarder: false,
	}

	// First check for traditional logging
	_, traditionErr := utils.RunCommand("oc", "get", "crd", "clusterloggings.logging.openshift.io")

	// Then check for Loki-based logging
	_, lokiErr := utils.RunCommand("oc", "get", "crd", "clusterlogforwarders.observability.openshift.io")

	// Determine logging type based on CRD availability
	if traditionErr == nil {
		// Check if traditional logging is actually deployed
		clOut, clErr := utils.RunCommand("oc", "get", "clusterlogging", "-n", "openshift-logging")
		if clErr == nil && strings.Contains(clOut, "instance") {
			info.Type = LoggingTypeTraditional
		}
	}

	if lokiErr == nil {
		// Check if Loki forwarding is configured
		clfOut, clfErr := utils.RunCommand("oc", "get", "clusterlogforwarders.observability.openshift.io", "-n", "openshift-logging")
		if clfErr == nil && strings.Contains(clfOut, "instance") {
			// Loki takes precedence if both are installed
			info.Type = LoggingTypeLoki
		}
	}

	// If logging is configured, check for external forwarders
	if info.Type != LoggingTypeNone {
		// Check for external forwarding configuration
		if info.Type == LoggingTypeLoki {
			// For Loki, check if there are non-Loki outputs in the ClusterLogForwarder
			clfoOut, err := utils.RunCommand("oc", "get", "clusterlogforwarders.observability.openshift.io", "-n", "openshift-logging", "-o", "yaml")
			if err == nil {
				// Check for output types other than lokiStack
				if strings.Contains(clfoOut, "type: elasticsearch") ||
					strings.Contains(clfoOut, "type: fluentdForward") ||
					strings.Contains(clfoOut, "type: kafka") ||
					strings.Contains(clfoOut, "type: syslog") ||
					strings.Contains(clfoOut, "type: cloudwatch") {
					info.HasExternalForwarder = true
				}
			}
		} else {
			// For traditional logging, check for ClusterLogForwarder
			clfOut, err := utils.RunCommand("oc", "get", "clusterlogforwarder", "-n", "openshift-logging")
			if err == nil && strings.Contains(clfOut, "instance") {
				info.HasExternalForwarder = true
			}
		}
	}

	return info, nil
}

// Helper function for backward compatibility
func isLoggingInstalled() (bool, error) {
	info, err := DetectLoggingConfiguration()
	return info.Type != LoggingTypeNone, err
}
