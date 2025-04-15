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

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "proxy/cluster", "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed proxy configuration"
	}

	// Check if proxy is configured
	if proxyConfig.Spec.HTTPProxy == "" && proxyConfig.Spec.HTTPSProxy == "" && proxyConfig.Spec.NoProxy == "" {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusNotApplicable,
			"OpenShift Proxy is not configured",
			types.ResultKeyNotApplicable,
		)
		result.Detail = detailedOut
		return result, nil
	}

	// If proxy is configured, check if both HTTP and HTTPS proxies are set
	isComplete := proxyConfig.Spec.HTTPProxy != "" && proxyConfig.Spec.HTTPSProxy != "" && proxyConfig.Spec.NoProxy != ""

	if !isComplete {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			"OpenShift Proxy configuration is incomplete",
			types.ResultKeyAdvisory,
		)
		result.AddRecommendation("Configure both HTTP and HTTPS proxies, and set appropriate NoProxy values")
		result.Detail = detailedOut
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
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			fmt.Sprintf("OpenShift Proxy is configured but NoProxy is missing important domains: %s", strings.Join(missingDomains, ", ")),
			types.ResultKeyAdvisory,
		)
		result.AddRecommendation(fmt.Sprintf("Add these domains to NoProxy: %s", strings.Join(missingDomains, ", ")))
		result.Detail = detailedOut
		return result, nil
	}

	// All checks passed
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		"OpenShift Proxy is properly configured",
		types.ResultKeyNoChange,
	)
	result.Detail = detailedOut
	return result, nil
}
