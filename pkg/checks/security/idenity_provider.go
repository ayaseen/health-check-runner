/*
Author: Amjad Yaseen
Email: ayaseen@redhat.com
Date: 2023-03-06
Modified: 2025-04-15

This file implements health checks for identity provider configuration. It:

- Verifies if a secure central identity provider (like LDAP) is configured
- Examines OAuth configuration for proper integration
- Checks for secure connection settings to identity sources
- Provides recommendations for proper authentication configuration
- Helps ensure secure user access to the cluster

This check helps maintain proper authentication and identity management for cluster users, a critical security component.
*/

package security

import (
	"encoding/json"
	"fmt"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"strings"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// IdentityProviderConfig represents the structure of the identity provider configuration
type IdentityProviderConfig struct {
	Spec struct {
		IdentityProviders []struct {
			Name          string `json:"name"`
			Type          string `json:"type"`
			MappingMethod string `json:"mappingMethod"`
			LDAP          *struct {
				URL          string `json:"url"`
				BindDN       string `json:"bindDN"`
				BindPassword string `json:"bindPassword"`
				Insecure     bool   `json:"insecure"`
				Attributes   struct {
					ID                []string `json:"id"`
					PreferredUsername []string `json:"preferredUsername"`
					Name              []string `json:"name"`
					Email             []string `json:"email"`
				} `json:"attributes"`
			} `json:"ldap,omitempty"`
		} `json:"identityProviders"`
	} `json:"spec"`
}

// IdentityProviderCheck checks if a secure central identity provider (LDAP) is configured
type IdentityProviderCheck struct {
	healthcheck.BaseCheck
}

// NewIdentityProviderCheck creates a new identity provider check
func NewIdentityProviderCheck() *IdentityProviderCheck {
	return &IdentityProviderCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"identity-provider",
			"Identity Provider Configuration",
			"Checks if a central identity provider (LDAP) is properly configured and secure",
			types.CategorySecurity,
		),
	}
}

// Run executes the health check
func (c *IdentityProviderCheck) Run() (healthcheck.Result, error) {
	// Get the OAuth configuration
	out, err := utils.RunCommand("oc", "get", "oauth", "cluster", "-o", "json")
	if err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to get OAuth configuration",
			types.ResultKeyRequired,
		), fmt.Errorf("error getting OAuth configuration: %v", err)
	}

	// Parse the JSON output
	var idpConfig IdentityProviderConfig
	if err := json.Unmarshal([]byte(out), &idpConfig); err != nil {
		return healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"Failed to parse OAuth configuration",
			types.ResultKeyRequired,
		), fmt.Errorf("error parsing OAuth configuration: %v", err)
	}

	// Get detailed information for the report
	detailedOut, err := utils.RunCommand("oc", "get", "oauth", "cluster", "-o", "yaml")
	if err != nil {
		// Non-critical error, we can continue without detailed output
		detailedOut = "Failed to get detailed OAuth configuration"
	}

	// Get OpenShift version for documentation links
	version, verErr := utils.GetOpenShiftMajorMinorVersion()
	if verErr != nil {
		version = "4.10" // Default to a known version if we can't determine
	}

	// Check if any identity providers are configured
	if len(idpConfig.Spec.IdentityProviders) == 0 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusCritical,
			"No identity providers are configured",
			types.ResultKeyRequired,
		)
		result.AddRecommendation("Configure at least one identity provider for user authentication")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/authentication_and_authorization/index#understanding-identity-provider", version))
		result.Detail = detailedOut
		return result, nil
	}

	// Check if LDAP is configured
	var ldapProviders []string
	for _, provider := range idpConfig.Spec.IdentityProviders {
		if provider.Type == "LDAP" {
			ldapProviders = append(ldapProviders, provider.Name)
		}
	}

	if len(ldapProviders) == 0 {
		// No LDAP providers, but other providers exist
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			fmt.Sprintf("Identity providers are configured (%s), but no LDAP provider found", getProviderTypes(idpConfig.Spec.IdentityProviders)),
			types.ResultKeyRecommended,
		)
		result.AddRecommendation("Configure a central identity provider (LDAP) for better integration with existing identity management systems")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/authentication_and_authorization/index#configuring-ldap-identity-provider", version))
		result.Detail = detailedOut
		return result, nil
	}

	// LDAP is configured, check for secure connection and proper search filters
	var insecureProviders []string
	var noSearchFilterProviders []string

	for _, provider := range idpConfig.Spec.IdentityProviders {
		if provider.Type == "LDAP" && provider.LDAP != nil {
			// Check for secure connection (URL starts with ldaps:// or insecure is false)
			if !strings.HasPrefix(provider.LDAP.URL, "ldaps://") && provider.LDAP.Insecure {
				insecureProviders = append(insecureProviders, provider.Name)
			}

			// Check for search filters (URL contains a filter like "(objectClass=*)")
			if !strings.Contains(provider.LDAP.URL, "(") || !strings.Contains(provider.LDAP.URL, ")") {
				noSearchFilterProviders = append(noSearchFilterProviders, provider.Name)
			}
		}
	}

	// If any LDAP providers are insecure
	if len(insecureProviders) > 0 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			fmt.Sprintf("LDAP providers are configured but some are using insecure connections: %s", strings.Join(insecureProviders, ", ")),
			types.ResultKeyRecommended,
		)
		result.AddRecommendation("Configure LDAP providers to use secure connections (ldaps:// or set insecure to false)")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/authentication_and_authorization/index#configuring-ldap-identity-provider", version))
		result.Detail = detailedOut
		return result, nil
	}

	// If any LDAP providers are missing search filters
	if len(noSearchFilterProviders) > 0 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusWarning,
			fmt.Sprintf("LDAP providers are configured but some are missing search filters: %s", strings.Join(noSearchFilterProviders, ", ")),
			types.ResultKeyAdvisory,
		)
		result.AddRecommendation("Configure LDAP providers with appropriate search filters to limit the scope of user searches")
		result.AddRecommendation(fmt.Sprintf("Refer to https://access.redhat.com/documentation/en-us/openshift_container_platform/%s/html-single/authentication_and_authorization/index#configuring-ldap-identity-provider", version))
		result.Detail = detailedOut
		return result, nil
	}

	// All checks passed
	result := healthcheck.NewResult(
		c.ID(),
		types.StatusOK,
		fmt.Sprintf("LDAP identity providers are properly configured: %s", strings.Join(ldapProviders, ", ")),
		types.ResultKeyNoChange,
	)
	result.Detail = detailedOut
	return result, nil
}

// getProviderTypes returns a comma-separated list of identity provider types
func getProviderTypes(providers []struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	MappingMethod string `json:"mappingMethod"`
	LDAP          *struct {
		URL          string `json:"url"`
		BindDN       string `json:"bindDN"`
		BindPassword string `json:"bindPassword"`
		Insecure     bool   `json:"insecure"`
		Attributes   struct {
			ID                []string `json:"id"`
			PreferredUsername []string `json:"preferredUsername"`
			Name              []string `json:"name"`
			Email             []string `json:"email"`
		} `json:"attributes"`
	} `json:"ldap,omitempty"`
}) string {
	var types []string
	typesMap := make(map[string]bool)

	for _, provider := range providers {
		if !typesMap[provider.Type] {
			types = append(types, provider.Type)
			typesMap[provider.Type] = true
		}
	}

	return strings.Join(types, ", ")
}
