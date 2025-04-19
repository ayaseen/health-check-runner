// pkg/utils/executive_summary/generator.go

package executive_summary

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/ayaseen/health-check-runner/pkg/checks"
	"github.com/ayaseen/health-check-runner/pkg/types"
)

// CategoryInfo represents information about a check category
type CategoryInfo struct {
	Name  string
	Score int
}

// CheckSummary represents the summary of health check results
type CheckSummary struct {
	ClusterName        string
	CustomerName       string
	Total              int
	NoChange           int
	Advisory           int
	ChangesRecommended int
	ChangesRequired    int
	NotApplicable      int
	ItemsRequired      []string
	ItemsRecommended   []string
	ItemsAdvisory      []string

	// Scores for specific category groupings (for the template)
	ScoreInfra         int
	ScoreGovernance    int
	ScoreCompliance    int
	ScoreMonitoring    int
	ScoreBuildSecurity int

	// Dynamic descriptions for each category based on actual results
	InfraDescription         string
	GovernanceDescription    string
	ComplianceDescription    string
	MonitoringDescription    string
	BuildSecurityDescription string

	// Scores for each category (string keys for the template)
	CategoryResults map[string][]string

	// Category info with names and scores
	Categories []CategoryInfo
}

// CheckMetadata stores the mapping of check IDs to their metadata
type CheckMetadata map[string]struct {
	Name     string
	Category types.Category
}

// GenerateExecutiveSummary generates an executive summary based on the health check report
func GenerateExecutiveSummary(reportPath string, clusterName string, customerName string, outputDir string) error {
	// Parse the health check report
	lines, err := parseHealthCheckFile(reportPath)
	if err != nil {
		return fmt.Errorf("error reading report file: %v", err)
	}

	// Get metadata for all checks from the checks package
	checkMetadata := buildCheckMetadata()

	// Summarize the checks
	summary := summarizeChecks(lines, checkMetadata)
	summary.ClusterName = clusterName
	summary.CustomerName = customerName

	// Calculate category scores
	calculateCategoryScores(&summary)
	overallScore := calculateOverallScore(summary)

	// Render the executive summary
	output, err := renderExecutiveSummary(summary, overallScore)
	if err != nil {
		return fmt.Errorf("render error: %v", err)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Write the executive summary
	outputPath := filepath.Join(outputDir, "070_executive-summary.adoc")
	if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
		return fmt.Errorf("write error: %v", err)
	}

	fmt.Printf("✅ Executive summary generated: %s (score: %.2f%%)\n", outputPath, overallScore)
	return nil
}

// parseHealthCheckFile reads the input adoc file and returns its lines
func parseHealthCheckFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// buildCheckMetadata creates a mapping of check IDs to their metadata
func buildCheckMetadata() CheckMetadata {
	metadata := make(CheckMetadata)

	// Get all checks from the checks package
	allChecks := checks.GetAllChecks()

	// Build metadata for each check
	for _, check := range allChecks {
		metadata[check.ID()] = struct {
			Name     string
			Category types.Category
		}{
			Name:     check.Name(),
			Category: check.Category(),
		}
	}

	return metadata
}

// cleanItemText removes the leading "<<" and trailing ">>" (if present)
// and trims any extra whitespace
func cleanItemText(item string) string {
	item = strings.TrimSpace(item)
	if strings.HasPrefix(item, "<<") && strings.HasSuffix(item, ">>") {
		item = strings.TrimSpace(item[2 : len(item)-2])
	}
	return item
}

// getCategoryString returns a string version of the category
func getCategoryString(category types.Category) string {
	// Use variable to convert to string
	categoryStr := ""

	switch category {
	case types.CategoryClusterConfig:
		categoryStr = "Cluster Config"
	case types.CategoryNetworking:
		categoryStr = "Networking"
	case types.CategoryApplications:
		categoryStr = "Applications"
	case types.CategoryOpReady:
		categoryStr = "Op-Ready"
	case types.CategorySecurity:
		categoryStr = "Security"
	case types.CategoryStorage:
		categoryStr = "Storage"
	case types.CategoryPerformance:
		categoryStr = "Performance"
	default:
		categoryStr = "Unknown"
	}

	return categoryStr
}

// summarizeChecks processes the adoc lines and builds the CheckSummary
func summarizeChecks(lines []string, checkMetadata CheckMetadata) CheckSummary {
	var summary CheckSummary
	summary.CategoryResults = make(map[string][]string)

	// Initialize categories
	for _, category := range []types.Category{
		types.CategoryClusterConfig,
		types.CategoryNetworking,
		types.CategoryApplications,
		types.CategoryOpReady,
		types.CategorySecurity,
		types.CategoryStorage,
		types.CategoryPerformance,
	} {
		catStr := getCategoryString(category)
		summary.CategoryResults[catStr] = []string{}
	}

	// Track items for each category for generating descriptions
	itemsByCategory := make(map[string][]string)
	for _, category := range []string{
		"Cluster Config", "Networking", "Applications", "Op-Ready",
		"Security", "Storage", "Performance",
	} {
		itemsByCategory[category] = []string{}
	}

	inSummary := false
	inBlock := false
	foundColor := false
	currentItem := ""
	currentCheckID := ""

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Stop if we reach the "# Infrastructure" section or any category section
		if strings.HasPrefix(line, "# ") {
			break
		}

		// Remove leading pipe from table rows
		if strings.HasPrefix(line, "|") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "|"))
		}

		if line == "= Summary" {
			inSummary = true
			continue
		}

		if !inSummary {
			continue
		}

		// Detect ITEM START/END markers
		if strings.Contains(line, "// ------------------------ITEM START") {
			inBlock = true
			foundColor = false
			currentItem = ""
			currentCheckID = ""
			continue
		}

		if strings.Contains(line, "// ------------------------ITEM END") {
			inBlock = false
			continue
		}

		if inBlock {
			// Check for the item source line which contains the check ID
			if strings.Contains(line, "// ----ITEM SOURCE") {
				parts := strings.Split(line, "/")
				if len(parts) > 0 {
					lastPart := parts[len(parts)-1]
					if strings.Contains(lastPart, ".item") {
						currentCheckID = strings.TrimSuffix(lastPart, ".item")
					}
				}
				continue
			}

			// If a line contains <<...>>, use that as the item text
			if strings.Contains(line, "<<") && strings.Contains(line, ">>") {
				currentItem = cleanItemText(line)
				continue
			}

			// If the line starts with "a|", try to capture the item text
			if strings.HasPrefix(line, "a|") && currentItem == "" {
				// Remove the "a|" prefix
				tmp := strings.TrimSpace(strings.TrimPrefix(line, "a|"))
				// If nothing remains, try to use the next line
				if tmp == "" && i+1 < len(lines) {
					i++
					tmp = strings.TrimSpace(lines[i])
				}
				if tmp != "" {
					currentItem = cleanItemText(tmp)
				}
				continue
			}

			// A color marker indicates that the next line is the check status
			if strings.HasPrefix(line, "{set:cellbgcolor:#") {
				foundColor = true
				continue
			}

			// Once found, read the status and process the item
			if foundColor {
				status := strings.ToLower(strings.TrimSpace(line))

				// Determine category
				var categoryStr string

				if currentCheckID != "" {
					if meta, exists := checkMetadata[currentCheckID]; exists {
						// Use metadata if available
						categoryStr = getCategoryString(meta.Category)
					} else {
						// Fallback to using the item text
						categoryStr = getCategoryStringFromItemText(currentItem)
					}
				} else {
					// No check ID found, use item text for categorization
					categoryStr = getCategoryStringFromItemText(currentItem)
				}

				// Save the status under the determined category
				summary.CategoryResults[categoryStr] = append(summary.CategoryResults[categoryStr], status)

				// Store the item text for this category (for description generation)
				if currentItem != "" {
					itemsByCategory[categoryStr] = append(itemsByCategory[categoryStr], currentItem)
				}

				// Tally counts based on the status
				switch {
				case strings.HasPrefix(status, "no change"):
					summary.NoChange++
				case strings.HasPrefix(status, "advisory"):
					summary.Advisory++
					summary.ItemsAdvisory = append(summary.ItemsAdvisory, currentItem)
				case strings.HasPrefix(status, "changes recommended"):
					summary.ChangesRecommended++
					summary.ItemsRecommended = append(summary.ItemsRecommended, currentItem)
				case strings.HasPrefix(status, "changes required"):
					summary.ChangesRequired++
					summary.ItemsRequired = append(summary.ItemsRequired, currentItem)
				case strings.HasPrefix(status, "not applicable"):
					summary.NotApplicable++
				}

				foundColor = false
			}
		}
	}

	summary.Total = summary.NoChange + summary.Advisory + summary.ChangesRecommended + summary.ChangesRequired + summary.NotApplicable
	return summary
}

// getCategoryStringFromItemText is a fallback method that determines a category based on item text
func getCategoryStringFromItemText(item string) string {
	itemLower := strings.ToLower(strings.TrimSpace(item))

	// Try to match with keywords associated with each category

	// Cluster Config / Infrastructure Setup
	if strings.Contains(itemLower, "cluster") || strings.Contains(itemLower, "infrastructure") ||
		strings.Contains(itemLower, "node") || strings.Contains(itemLower, "etcd") ||
		strings.Contains(itemLower, "installation type") || strings.Contains(itemLower, "provider") ||
		strings.Contains(itemLower, "machine") || strings.Contains(itemLower, "version") {
		return "Cluster Config"
	}

	// Networking
	if strings.Contains(itemLower, "network") || strings.Contains(itemLower, "ingress") ||
		strings.Contains(itemLower, "cni") || strings.Contains(itemLower, "dns") ||
		strings.Contains(itemLower, "controller") {
		return "Networking"
	}

	// Applications
	if strings.Contains(itemLower, "probe") || strings.Contains(itemLower, "resource quota") ||
		strings.Contains(itemLower, "emptydir") || strings.Contains(itemLower, "limit") ||
		strings.Contains(itemLower, "readiness") || strings.Contains(itemLower, "liveness") ||
		strings.Contains(itemLower, "registry") {
		return "Applications"
	}

	// Op-Ready (Monitoring and Logging)
	if strings.Contains(itemLower, "monitor") || strings.Contains(itemLower, "logging") ||
		strings.Contains(itemLower, "alert") || strings.Contains(itemLower, "metric") ||
		strings.Contains(itemLower, "forward") || strings.Contains(itemLower, "elasticsearch") ||
		strings.Contains(itemLower, "backup") {
		return "Op-Ready"
	}

	// Security
	if strings.Contains(itemLower, "security") || strings.Contains(itemLower, "privilege") ||
		strings.Contains(itemLower, "scc") || strings.Contains(itemLower, "rbac") ||
		strings.Contains(itemLower, "identity") || strings.Contains(itemLower, "ldap") ||
		strings.Contains(itemLower, "encrypt") || strings.Contains(itemLower, "kubeadmin") ||
		strings.Contains(itemLower, "policy") {
		return "Security"
	}

	// Storage
	if strings.Contains(itemLower, "storage") || strings.Contains(itemLower, "volume") ||
		strings.Contains(itemLower, "persistent") || strings.Contains(itemLower, "pv") {
		return "Storage"
	}

	// Performance
	if strings.Contains(itemLower, "performance") || strings.Contains(itemLower, "throughput") ||
		strings.Contains(itemLower, "latency") || strings.Contains(itemLower, "speed") {
		return "Performance"
	}

	// Default fallback
	return "Cluster Config"
}

// calculateCategoryScores calculates the score for each category
func calculateCategoryScores(summary *CheckSummary) {
	// Initialize Categories slice
	summary.Categories = make([]CategoryInfo, 0)

	// Map to store category scores for special groupings
	infraScores := []float64{}
	governanceScores := []float64{}
	complianceScores := []float64{}
	monitoringScores := []float64{}
	buildSecurityScores := []float64{}

	// Store items by category for generating descriptions
	infraItems := []string{}
	governanceItems := []string{}
	complianceItems := []string{}
	monitoringItems := []string{}
	buildSecurityItems := []string{}

	// Store status by category for determining descriptions
	infraStatus := map[string]int{"ok": 0, "warning": 0, "critical": 0, "na": 0}
	governanceStatus := map[string]int{"ok": 0, "warning": 0, "critical": 0, "na": 0}
	complianceStatus := map[string]int{"ok": 0, "warning": 0, "critical": 0, "na": 0}
	monitoringStatus := map[string]int{"ok": 0, "warning": 0, "critical": 0, "na": 0}
	buildSecurityStatus := map[string]int{"ok": 0, "warning": 0, "critical": 0, "na": 0}

	// Calculate scores for each category
	for categoryStr, results := range summary.CategoryResults {
		total := len(results)
		if total == 0 {
			fmt.Printf("⚠️  %s: No items found\n", categoryStr)
			summary.Categories = append(summary.Categories, CategoryInfo{
				Name:  categoryStr,
				Score: 0,
			})
			continue
		}

		scoreSum := 0.0
		for _, r := range results {
			scoreSum += resultScore(r)
		}

		finalScore := (scoreSum / float64(total)) * 100

		// Add to Categories slice
		summary.Categories = append(summary.Categories, CategoryInfo{
			Name:  categoryStr,
			Score: int(finalScore),
		})

		// Map category to special groupings based on your desired template
		// This mapping is based on the paste.txt categories
		switch categoryStr {
		case "Cluster Config":
			infraScores = append(infraScores, finalScore)
			infraItems = append(infraItems, getItemsForCategory(summary, categoryStr)...)
			updateStatusCounts(infraStatus, results)
		case "Networking":
			infraScores = append(infraScores, finalScore)
			infraItems = append(infraItems, getItemsForCategory(summary, categoryStr)...)
			updateStatusCounts(infraStatus, results)
		case "Security":
			governanceScores = append(governanceScores, finalScore)
			governanceItems = append(governanceItems, getItemsForCategory(summary, categoryStr)...)
			updateStatusCounts(governanceStatus, results)

			// Security items could go in multiple categories
			for _, itemText := range getItemsForCategory(summary, categoryStr) {
				itemLower := strings.ToLower(itemText)
				if strings.Contains(itemLower, "authentication") ||
					strings.Contains(itemLower, "encrypt") ||
					strings.Contains(itemLower, "ldap") ||
					strings.Contains(itemLower, "identity") {
					complianceItems = append(complianceItems, itemText)
					complianceScores = append(complianceScores, finalScore)
				}
				if strings.Contains(itemLower, "elevated") ||
					strings.Contains(itemLower, "privilege") ||
					strings.Contains(itemLower, "scc") {
					buildSecurityItems = append(buildSecurityItems, itemText)
					buildSecurityScores = append(buildSecurityScores, finalScore)
				}
			}
		case "Op-Ready":
			monitoringScores = append(monitoringScores, finalScore)
			monitoringItems = append(monitoringItems, getItemsForCategory(summary, categoryStr)...)
			updateStatusCounts(monitoringStatus, results)
		case "Applications":
			buildSecurityScores = append(buildSecurityScores, finalScore)
			buildSecurityItems = append(buildSecurityItems, getItemsForCategory(summary, categoryStr)...)
			updateStatusCounts(buildSecurityStatus, results)
		case "Storage":
			infraScores = append(infraScores, finalScore)
			infraItems = append(infraItems, getItemsForCategory(summary, categoryStr)...)
			updateStatusCounts(infraStatus, results)
		case "Performance":
			infraScores = append(infraScores, finalScore)
			infraItems = append(infraItems, getItemsForCategory(summary, categoryStr)...)
			updateStatusCounts(infraStatus, results)
		}

		fmt.Printf("📊 %s: %d items → avg %.2f%%\n", categoryStr, total, finalScore)
	}

	// Calculate average scores for each special grouping
	summary.ScoreInfra = calculateAverageScore(infraScores)
	summary.ScoreGovernance = calculateAverageScore(governanceScores)
	summary.ScoreCompliance = calculateAverageScore(complianceScores)
	summary.ScoreMonitoring = calculateAverageScore(monitoringScores)
	summary.ScoreBuildSecurity = calculateAverageScore(buildSecurityScores)

	// If any score is still 0, use a default score based on overall health
	overallAvg := calculateAverageScore([]float64{
		float64(summary.ScoreInfra),
		float64(summary.ScoreGovernance),
		float64(summary.ScoreCompliance),
		float64(summary.ScoreMonitoring),
		float64(summary.ScoreBuildSecurity),
	})

	if summary.ScoreInfra == 0 {
		summary.ScoreInfra = overallAvg
	}
	if summary.ScoreGovernance == 0 {
		summary.ScoreGovernance = overallAvg
	}
	if summary.ScoreCompliance == 0 {
		summary.ScoreCompliance = overallAvg
	}
	if summary.ScoreMonitoring == 0 {
		summary.ScoreMonitoring = overallAvg
	}
	if summary.ScoreBuildSecurity == 0 {
		summary.ScoreBuildSecurity = overallAvg
	}

	// Generate dynamic descriptions based on actual findings
	summary.InfraDescription = generateSpecificDescriptions("Infrastructure", summary.ScoreInfra, infraItems, infraStatus)
	summary.GovernanceDescription = generateSpecificDescriptions("Policy Governance", summary.ScoreGovernance, governanceItems, governanceStatus)
	summary.ComplianceDescription = generateSpecificDescriptions("Compliance", summary.ScoreCompliance, complianceItems, complianceStatus)
	summary.MonitoringDescription = generateSpecificDescriptions("Monitoring and Logging", summary.ScoreMonitoring, monitoringItems, monitoringStatus)
	summary.BuildSecurityDescription = generateSpecificDescriptions("Build/Deploy Security", summary.ScoreBuildSecurity, buildSecurityItems, buildSecurityStatus)
}

// getItemsForCategory returns the item names/texts for a specific category
func getItemsForCategory(summary *CheckSummary, category string) []string {
	// For now, this is a placeholder. In a real implementation, we'd track
	// which category each item belongs to during the initial parsing.
	// This would be populated from the itemsByCategory map defined in summarizeChecks.

	var items []string

	// As a fallback, look for category-related keywords in all item lists
	allItems := append(summary.ItemsRequired, append(summary.ItemsRecommended, summary.ItemsAdvisory...)...)

	for _, item := range allItems {
		itemLower := strings.ToLower(item)

		switch category {
		case "Cluster Config":
			if strings.Contains(itemLower, "cluster") ||
				strings.Contains(itemLower, "node") ||
				strings.Contains(itemLower, "infrastructure") ||
				strings.Contains(itemLower, "version") ||
				strings.Contains(itemLower, "etcd") ||
				strings.Contains(itemLower, "machine") {
				items = append(items, item)
			}
		case "Networking":
			if strings.Contains(itemLower, "network") ||
				strings.Contains(itemLower, "ingress") ||
				strings.Contains(itemLower, "dns") ||
				strings.Contains(itemLower, "cni") ||
				strings.Contains(itemLower, "route") {
				items = append(items, item)
			}
		case "Applications":
			if strings.Contains(itemLower, "probe") ||
				strings.Contains(itemLower, "quota") ||
				strings.Contains(itemLower, "limit") ||
				strings.Contains(itemLower, "emptydir") ||
				strings.Contains(itemLower, "registry") {
				items = append(items, item)
			}
		case "Op-Ready":
			if strings.Contains(itemLower, "monitor") ||
				strings.Contains(itemLower, "logging") ||
				strings.Contains(itemLower, "alert") ||
				strings.Contains(itemLower, "metric") ||
				strings.Contains(itemLower, "elasticsearch") ||
				strings.Contains(itemLower, "backup") {
				items = append(items, item)
			}
		case "Security":
			if strings.Contains(itemLower, "security") ||
				strings.Contains(itemLower, "privilege") ||
				strings.Contains(itemLower, "rbac") ||
				strings.Contains(itemLower, "scc") ||
				strings.Contains(itemLower, "authentication") ||
				strings.Contains(itemLower, "identity") ||
				strings.Contains(itemLower, "encrypt") ||
				strings.Contains(itemLower, "policy") {
				items = append(items, item)
			}
		case "Storage":
			if strings.Contains(itemLower, "storage") ||
				strings.Contains(itemLower, "volume") ||
				strings.Contains(itemLower, "persistent") ||
				strings.Contains(itemLower, "pv") {
				items = append(items, item)
			}
		case "Performance":
			if strings.Contains(itemLower, "performance") ||
				strings.Contains(itemLower, "throughput") ||
				strings.Contains(itemLower, "latency") ||
				strings.Contains(itemLower, "speed") {
				items = append(items, item)
			}
		}
	}

	return items
}

// updateStatusCounts updates the count of status types in a category
func updateStatusCounts(statusMap map[string]int, results []string) {
	for _, status := range results {
		statusLower := strings.ToLower(status)
		if strings.Contains(statusLower, "no change") {
			statusMap["ok"]++
		} else if strings.Contains(statusLower, "changes recommended") {
			statusMap["warning"]++
		} else if strings.Contains(statusLower, "changes required") {
			statusMap["critical"]++
		} else if strings.Contains(statusLower, "not applicable") {
			statusMap["na"]++
		}
	}
}

// getCommonStrengths extracts common strength keywords from items
func getCommonStrengths(items []string) []string {
	strengths := make(map[string]int)

	// Look for positive indicators in the item texts
	for _, item := range items {
		itemLower := strings.ToLower(item)

		// Check for common strength indicators
		if strings.Contains(itemLower, "configured") {
			strengths["configuration"] += 1
		}
		if strings.Contains(itemLower, "enabled") {
			strengths["enabled features"] += 1
		}
		if strings.Contains(itemLower, "status") && !strings.Contains(itemLower, "critical") {
			strengths["status"] += 1
		}
		if strings.Contains(itemLower, "monitor") {
			strengths["monitoring"] += 1
		}
		if strings.Contains(itemLower, "log") {
			strengths["logging"] += 1
		}
		if strings.Contains(itemLower, "backup") {
			strengths["backups"] += 1
		}
		if strings.Contains(itemLower, "network") {
			strengths["networking"] += 1
		}
		if strings.Contains(itemLower, "security") {
			strengths["security"] += 1
		}
		if strings.Contains(itemLower, "node") {
			strengths["node configuration"] += 1
		}
		if strings.Contains(itemLower, "version") {
			strengths["versioning"] += 1
		}
	}

	// Extract top strengths
	var result []string
	for k, v := range strengths {
		if v >= 2 {
			result = append(result, k)
		}
	}

	return result
}

// getCommonGaps extracts common gap keywords from items
func getCommonGaps(items []string) []string {
	gaps := make(map[string]int)

	// Look for warning/critical indicators in the item texts
	for _, item := range items {
		itemLower := strings.ToLower(item)

		if strings.Contains(itemLower, "missing") {
			gaps["missing configurations"] += 1
		}
		if strings.Contains(itemLower, "disabled") {
			gaps["disabled features"] += 1
		}
		if strings.Contains(itemLower, "alert") {
			gaps["alerts"] += 1
		}
		if strings.Contains(itemLower, "storage") {
			gaps["storage"] += 1
		}
		if strings.Contains(itemLower, "encryption") {
			gaps["encryption"] += 1
		}
		if strings.Contains(itemLower, "quota") {
			gaps["resource quotas"] += 1
		}
		if strings.Contains(itemLower, "privilege") {
			gaps["privilege management"] += 1
		}
		if strings.Contains(itemLower, "access") {
			gaps["access control"] += 1
		}
		if strings.Contains(itemLower, "forward") {
			gaps["forwarding"] += 1
		}
		if strings.Contains(itemLower, "audit") {
			gaps["audit"] += 1
		}
	}

	// Extract top gaps
	var result []string
	for k, v := range gaps {
		if v >= 1 {
			result = append(result, k)
		}
	}

	return result
}

// improvedGenerateDescription creates a more specific dynamic description based on actual check results
func improvedGenerateDescription(categoryName string, score int, items []string, statusCounts map[string]int) string {
	// Get strengths and gaps from items
	strengths := getCommonStrengths(items)
	gaps := getCommonGaps(items)

	// Calculate percentages of different status types
	total := statusCounts["ok"] + statusCounts["warning"] + statusCounts["critical"] + statusCounts["na"]
	if total == 0 {
		total = 1 // Avoid division by zero
	}

	okPercent := float64(statusCounts["ok"]) / float64(total) * 100
	warnPercent := float64(statusCounts["warning"]) / float64(total) * 100
	criticalPercent := float64(statusCounts["critical"]) / float64(total) * 100

	// Generate descriptive phrases based on actual findings
	var strengthPhrase, gapPhrase string

	// Handle strengths
	if len(strengths) > 0 {
		if len(strengths) == 1 {
			strengthPhrase = fmt.Sprintf("%s is well configured", strengths[0])
		} else if len(strengths) == 2 {
			strengthPhrase = fmt.Sprintf("both %s and %s are well established", strengths[0], strengths[1])
		} else {
			strengthPhrase = fmt.Sprintf("key areas like %s and %s are properly configured",
				strings.Join(strengths[:len(strengths)-1], ", "),
				strengths[len(strengths)-1])
		}
	} else if okPercent > 70 {
		strengthPhrase = "most configurations align with best practices"
	} else if okPercent > 50 {
		strengthPhrase = "some key configurations are properly set up"
	} else {
		strengthPhrase = "there are some properly configured elements"
	}

	// Handle gaps
	if len(gaps) > 0 {
		if criticalPercent > 20 {
			if len(gaps) == 1 {
				gapPhrase = fmt.Sprintf("%s requires immediate attention", gaps[0])
			} else if len(gaps) == 2 {
				gapPhrase = fmt.Sprintf("both %s and %s need critical attention", gaps[0], gaps[1])
			} else {
				gapPhrase = fmt.Sprintf("critical issues exist with %s and %s",
					strings.Join(gaps[:len(gaps)-1], ", "),
					gaps[len(gaps)-1])
			}
		} else if warnPercent > 20 {
			if len(gaps) == 1 {
				gapPhrase = fmt.Sprintf("%s could use improvement", gaps[0])
			} else if len(gaps) == 2 {
				gapPhrase = fmt.Sprintf("improvements in %s and %s are recommended", gaps[0], gaps[1])
			} else {
				gapPhrase = fmt.Sprintf("recommended improvements for %s and %s",
					strings.Join(gaps[:len(gaps)-1], ", "),
					gaps[len(gaps)-1])
			}
		} else {
			if len(gaps) == 1 {
				gapPhrase = fmt.Sprintf("minor adjustments to %s would be beneficial", gaps[0])
			} else {
				gapPhrase = fmt.Sprintf("minor adjustments to %s would optimize performance",
					strings.Join(gaps, " and "))
			}
		}
	} else if criticalPercent > 0 {
		gapPhrase = "some critical items require attention"
	} else if warnPercent > 0 {
		gapPhrase = "a few improvements are recommended"
	} else {
		gapPhrase = "with minimal gaps identified"
	}

	// Build the full description
	if score >= 90 {
		return fmt.Sprintf("%s is excellent; %s %s.", categoryName, strengthPhrase, gapPhrase)
	} else if score >= 80 {
		return fmt.Sprintf("%s is well-configured; %s %s.", categoryName, strengthPhrase, gapPhrase)
	} else if score >= 70 {
		return fmt.Sprintf("%s meets most requirements; %s %s.", categoryName, strengthPhrase, gapPhrase)
	} else if score >= 60 {
		return fmt.Sprintf("%s needs attention; %s %s.", categoryName, strengthPhrase, gapPhrase)
	} else {
		return fmt.Sprintf("%s requires significant improvements; %s %s.", categoryName, strengthPhrase, gapPhrase)
	}
}

// generateSpecificDescriptions creates category-specific descriptions
func generateSpecificDescriptions(category string, score int, items []string, statusCounts map[string]int) string {
	// Base description that's dynamically generated
	baseDesc := improvedGenerateDescription(category, score, items, statusCounts)

	// Add category-specific context based on the category name
	switch category {
	case "Infrastructure":
		if score >= 80 {
			return baseDesc + " The cluster infrastructure provides a solid foundation for workloads."
		} else if score >= 60 {
			return baseDesc + " Some infrastructure components need attention to ensure long-term stability."
		} else {
			return baseDesc + " Infrastructure improvements are needed to support production workloads."
		}

	case "Policy Governance":
		if score >= 80 {
			return baseDesc + " Security policies provide good protection for cluster resources."
		} else if score >= 60 {
			return baseDesc + " Policy enhancements would better secure the environment."
		} else {
			return baseDesc + " Significant policy improvements are needed to protect cluster resources."
		}

	case "Compliance":
		if score >= 80 {
			return baseDesc + " The cluster largely meets compliance requirements."
		} else if score >= 60 {
			return baseDesc + " Several compliance requirements need addressing."
		} else {
			return baseDesc + " Major compliance gaps exist that should be prioritized."
		}

	case "Monitoring and Logging":
		if score >= 80 {
			return baseDesc + " Observability systems provide good visibility into cluster operations."
		} else if score >= 60 {
			return baseDesc + " Improved observability would help with operational management."
		} else {
			return baseDesc + " Significant observability gaps exist that limit operational insight."
		}

	case "Build/Deploy Security":
		if score >= 80 {
			return baseDesc + " Application deployments are well-protected by security controls."
		} else if score >= 60 {
			return baseDesc + " Additional deployment security controls would improve workload protection."
		} else {
			return baseDesc + " Workload security requires significant hardening."
		}

	default:
		return baseDesc
	}
}

// calculateAverageScore calculates the average score from a slice of scores
func calculateAverageScore(scores []float64) int {
	if len(scores) == 0 {
		return 0
	}

	sum := 0.0
	for _, score := range scores {
		sum += score
	}

	return int(sum / float64(len(scores)))
}

// calculateOverallScore calculates the overall score
func calculateOverallScore(summary CheckSummary) float64 {
	if summary.Total == 0 {
		return 0.0
	}

	score := 0.0
	total := 0

	for _, results := range summary.CategoryResults {
		for _, r := range results {
			score += resultScore(r)
			total++
		}
	}

	return (score / float64(total)) * 100
}

// resultScore assigns a score for each status
func resultScore(status string) float64 {
	status = strings.ToLower(strings.TrimSpace(status))
	switch {
	case strings.HasPrefix(status, "no change"):
		return 1.00
	case strings.HasPrefix(status, "not applicable"):
		return 0.90
	case strings.HasPrefix(status, "advisory"):
		return 0.80
	case strings.HasPrefix(status, "changes recommended"):
		return 0.70
	case strings.HasPrefix(status, "changes required"):
		return 0.00
	default:
		return 0.00
	}
}

// renderExecutiveSummary renders the executive summary template
func renderExecutiveSummary(summary CheckSummary, score float64) (string, error) {
	// Define the template as a string
	tmplStr := `Red Hat Consulting conducted a comprehensive health check of {{.Summary.CustomerName}}'s OpenShift {{.Summary.ClusterName}} cluster. This review evaluated infrastructure readiness, policy enforcement, security posture, compliance maturity, and operational excellence.

**Overall Cluster Health: {{printf "%.2f" .Score}}%**

The {{.Summary.ClusterName}} cluster was assessed across {{.Summary.Total}} total checks. The health score is based on the number of items that required no change, were advisory, considered not applicable, or recommended with low criticality.

* *Infrastructure Setup*: {{.Summary.ScoreInfra}}%
+
{{.Summary.InfraDescription}}

* *Policy Governance*: {{.Summary.ScoreGovernance}}%
+
{{.Summary.GovernanceDescription}}

* *Compliance Benchmarking*: {{.Summary.ScoreCompliance}}%
+
{{.Summary.ComplianceDescription}}

* *Central Monitoring and Logging*: {{.Summary.ScoreMonitoring}}%
+
{{.Summary.MonitoringDescription}}

* *Build/Deploy Security (ACS)*: {{.Summary.ScoreBuildSecurity}}%
+
{{.Summary.BuildSecurityDescription}}

**Priority-Based Actions Breakdown**

* *Changes Required*: {{.Summary.ChangesRequired}} Items
{{- if .Summary.ItemsRequired }}
{{ range .Summary.ItemsRequired }}
- {{ . }}
{{ end }}
{{- else }}
- None
{{- end }}

* *Changes Recommended*: {{.Summary.ChangesRecommended}} Items
{{- if .Summary.ItemsRecommended }}
{{ range .Summary.ItemsRecommended }}
- {{ . }}
{{ end }}
{{- else }}
- None
{{- end }}

* *Advisory Actions*: {{.Summary.Advisory}} Items
{{- if .Summary.ItemsAdvisory }}
{{ range .Summary.ItemsAdvisory }}
- {{ . }}
{{ end }}
{{- else }}
- None
{{- end }}

* *No Changes Required*: {{.Summary.NoChange}} Items
- These areas fully align with Red Hat platform standards.

* *Not Applicable (N/A)*: {{.Summary.NotApplicable}} Items
- Checks not applicable to this cluster environment.

[IMPORTANT]
====
While the {{.Summary.ClusterName}} cluster is well-structured and adheres to most best practices, key risks remain in the following areas:
{{- if .Summary.ItemsRequired }}
{{ range .Summary.ItemsRequired }}
- {{ . }}
{{ end }}
{{- else }}
None
{{- end }}
Timely resolution of these findings will ensure full production readiness and platform resilience.
====`

	// Parse the template
	tmpl, err := template.New("executive-summary").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	// Prepare the data
	data := struct {
		Summary CheckSummary
		Score   float64
	}{
		Summary: summary,
		Score:   score,
	}

	// Execute the template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
