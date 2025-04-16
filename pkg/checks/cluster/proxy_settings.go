/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for OpenShift proxy settings. It:

- Examines if proxy configuration is present for clusters requiring it
- Verifies completeness of proxy configuration (HTTP, HTTPS, NoProxy)
- Checks if important domains are included in NoProxy settings
- Provides recommendations for optimal proxy configuration
- Helps ensure proper network connectivity in proxied environments

This check helps maintain proper external connectivity for clusters behind corporate proxies.
*/

package cluster

import (
	"encoding/json"
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// ProxyConfig represents the structure of the proxy configuration
type ProxyConfig struct {
	Spec struct {
		HTTPProxy  string `json:"httpProxy"`
		HTTPSProxy string `json:"httpsProxy"`
		NoProxy    string `json:"noProxy"`
	} `json:"spec"`
}

// ProxySettingsCheck checks the cluster's proxy settings
type ProxySettingsCheck struct {
	healthcheck.BaseCheck
}

// NewProxySettingsCheck creates a new proxy settings check
func NewProxySettingsCheck() *ProxySettingsCheck {
	return &ProxySettingsCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"proxy-settings",
			"OpenShift Proxy Settings",
			"Checks the proxy configuration for the OpenShift cluster",
			types.CategoryClusterConfig,
		),
	}
}

// Run executes the health check
func (c *ProxySettingsCheck) Run() (healthcheck.Result, error) {
	// Get the proxy configuration
	out, err := utils.RunCommand("oc", "get", "proxy/cluster", "-o", "json")
	if err != nil {
		// If there's an error, it likely means the proxy is not configured
		// This is not necessarily a failure, so we'll return a result indicating
		// that the proxy is not configured

		// Format the detailed output with proper AsciiDoc formatting
		var formattedDetailedOut strings.Builder
		formattedDetailedOut.WriteString("=== Proxy Configuration Analysis ===\n\n")
		formattedDetailedOut.WriteString("No proxy configuration found in the cluster.\n\n")

		return healthcheck.NewResult(
			c.ID(),
			types.StatusNotApplicable,
			"OpenShift Proxy is not configured",
			types.ResultKeyNotApplicable,
		), nil
	}

	// Parse the JSON output
	var proxyConfig ProxyConfig
	if err := json.Unmarshal([]byte(out), &proxyConfig); err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to parse proxy configuration",
			types.ResultKeyRequired,
		), fmt.Errorf("error parsing proxy configuration: %v", err)
	}

	// Format the detailed output with proper AsciiDoc formatting
	var formattedDetailedOut strings.Builder
	formattedDetailedOut.WriteString("=== Proxy Configuration Analysis ===\n\n")

	// Add proxy configuration with proper formatting
	if strings.TrimSpace(out) != "" {
		formattedDetailedOut.WriteString("Proxy Configuration:\n[source, json]\n----\n")
		formattedDetailedOut.WriteString(out)
		formattedDetailedOut.WriteString("\n----\n\n")
	} else {
		formattedDetailedOut.WriteString("Proxy Configuration: No information available\n\n")
	}

	// Add human-readable summary
	formattedDetailedOut.WriteString("=== Proxy Settings Summary ===\n\n")
	formattedDetailedOut.WriteString(fmt.Sprintf("HTTP Proxy: %s\n", proxyConfig.Spec.HTTPProxy))
	formattedDetailedOut.WriteString(fmt.Sprintf("HTTPS Proxy: %s\n", proxyConfig.Spec.HTTPSProxy))
	formattedDetailedOut.WriteString(fmt.Sprintf("No Proxy: %s\n\n", proxyConfig.Spec.NoProxy))

	// Check if proxy is configured
	if proxyConfig.Spec.HTTPProxy == "" && proxyConfig.Spec.HTTPSProxy == "" && proxyConfig.Spec.NoProxy == "" {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusNotApplicable,
			"OpenShift Proxy is not configured",
			types.ResultKeyNotApplicable,
		)
		result.Detail = formattedDetailedOut.String()
		return result, nil
	}

	// If proxy is configured, check if both HTTP and HTTPS proxies are set
	isComplete := proxyConfig.Spec.HTTPProxy != "" && proxyConfig.Spec.HTTPSProxy != "" && proxyConfig.Spec.NoProxy != ""

	if !isComplete {
		// Add analysis of missing components
		formattedDetailedOut.WriteString("=== Configuration Issues ===\n\n")
		if proxyConfig.Spec.HTTPProxy == "" {
			formattedDetailedOut.WriteString("- HTTP Proxy is not configured\n")
		}
		if proxyConfig.Spec.HTTPSProxy == "" {
			formattedDetailedOut.WriteString("- HTTPS Proxy is not configured\n")
		}
		if proxyConfig.Spec.NoProxy == "" {
			formattedDetailedOut.WriteString("- No Proxy list is not configured\n")
		}
		formattedDetailedOut.WriteString("\n")

		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"OpenShift Proxy configuration is incomplete",
			types.ResultKeyAdvisory,
		)
		result.AddRecommendation("Configure both HTTP and HTTPS proxies, and set appropriate NoProxy values")
		result.Detail = formattedDetailedOut.String()
		return result, nil
	}

	// Check if NoProxy includes important OpenShift domains
	importantDomains := []string{
		".cluster.local",
		".svc",
		"localhost",
		"127.0.0.1",
	}

	missingDomains := []string{}
	for _, domain := range importantDomains {
		if !strings.Contains(proxyConfig.Spec.NoProxy, domain) {
			missingDomains = append(missingDomains, domain)
		}
	}

	if len(missingDomains) > 0 {
		// Add analysis of missing domains
		formattedDetailedOut.WriteString("=== NoProxy Configuration Issues ===\n\n")
		formattedDetailedOut.WriteString("The following important domains are missing from the NoProxy configuration:\n")
		for _, domain := range missingDomains {
			formattedDetailedOut.WriteString(fmt.Sprintf("- %s\n", domain))
		}
		formattedDetailedOut.WriteString("\nThese domains should be included to ensure proper internal cluster communication.\n\n")

		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			fmt.Sprintf("OpenShift Proxy is configured but NoProxy is missing important domains: %s", strings.Join(missingDomains, ", ")),
			types.ResultKeyAdvisory,
		)
		result.AddRecommendation(fmt.Sprintf("Add these domains to NoProxy: %s", strings.Join(missingDomains, ", ")))
		result.Detail = formattedDetailedOut.String()
		return result, nil
	}

	// All checks passed
	formattedDetailedOut.WriteString("=== Configuration Analysis ===\n\n")
	formattedDetailedOut.WriteString("The proxy configuration is complete and properly configured.\n")
	formattedDetailedOut.WriteString("- Both HTTP and HTTPS proxies are defined\n")
	formattedDetailedOut.WriteString("- NoProxy list includes all important internal domains\n\n")

	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		"OpenShift Proxy is properly configured",
		types.ResultKeyNoChange,
	)
	result.Detail = formattedDetailedOut.String()
	return result, nil
}
