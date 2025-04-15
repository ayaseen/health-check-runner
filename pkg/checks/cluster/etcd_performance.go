package cluster

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ayaseen/health-check-runner/pkg/healthcheck"
	"github.com/ayaseen/health-check-runner/pkg/types"
	"github.com/ayaseen/health-check-runner/pkg/utils"
)

// EtcdPerformanceCheck checks the performance of the etcd cluster
type EtcdPerformanceCheck struct {
	healthcheck.BaseCheck
}

// NewEtcdPerformanceCheck creates a new etcd performance check
func NewEtcdPerformanceCheck() *EtcdPerformanceCheck {
	return &EtcdPerformanceCheck{
		BaseCheck: healthcheck.NewBaseCheck(
			"etcd-performance",
			"ETCD Performance",
			"Checks etcd performance metrics for optimal operation",
			types.CategoryClusterConfig,
		),
	}
}

// Run executes the health check
func (c *EtcdPerformanceCheck) Run() (healthcheck.Result, error) {
	var details strings.Builder
	details.WriteString("=== ETCD Performance Analysis ===\n\n")

	// Variables to track performance issues
	var performanceIssues []string
	hasCritical := false

	// Check for slow compaction issues
	compactionStats, compactionIssues := checkCompactionPerformance()
	if len(compactionIssues) > 0 {
		performanceIssues = append(performanceIssues, compactionIssues...)
		details.WriteString("== Compaction Performance ==\n")
		details.WriteString(compactionStats)
		details.WriteString("\n\n")
	} else {
		details.WriteString("== Compaction Performance ==\n")
		details.WriteString("Compaction performance is good. No slow compactions detected.\n")
		details.WriteString(compactionStats)
		details.WriteString("\n\n")
	}

	// Check for "took too long" entries
	applyStats, applyIssues := checkApplyEntriesPerformance()
	if len(applyIssues) > 0 {
		performanceIssues = append(performanceIssues, applyIssues...)
		if strings.Contains(applyStats, "excessive delays") {
			hasCritical = true
		}
		details.WriteString("== Entry Apply Performance ==\n")
		details.WriteString(applyStats)
		details.WriteString("\n\n")
	} else {
		details.WriteString("== Entry Apply Performance ==\n")
		details.WriteString("Entry apply performance is good. No delays detected.\n")
		details.WriteString(applyStats)
		details.WriteString("\n\n")
	}

	// Check for heartbeat issues
	heartbeatStats, heartbeatIssues := checkHeartbeatPerformance()
	if len(heartbeatIssues) > 0 {
		performanceIssues = append(performanceIssues, heartbeatIssues...)
		if strings.Contains(heartbeatStats, "frequent failures") {
			hasCritical = true
		}
		details.WriteString("== Heartbeat Performance ==\n")
		details.WriteString(heartbeatStats)
		details.WriteString("\n\n")
	} else {
		details.WriteString("== Heartbeat Performance ==\n")
		details.WriteString("Heartbeat performance is good. No issues detected.\n")
		details.WriteString(heartbeatStats)
		details.WriteString("\n\n")
	}

	// Check for clock drift issues
	clockDriftStats, clockDriftIssues := checkClockDriftIssues()
	if len(clockDriftIssues) > 0 {
		performanceIssues = append(performanceIssues, clockDriftIssues...)
		details.WriteString("== Clock Synchronization ==\n")
		details.WriteString(clockDriftStats)
		details.WriteString("\n\n")
	} else {
		details.WriteString("== Clock Synchronization ==\n")
		details.WriteString("Clock synchronization appears good. No drift issues detected.\n")
		details.WriteString(clockDriftStats)
		details.WriteString("\n\n")
	}

	// Run diagnostic test if possible (may not work in all environments)
	diagStats, diagIssues := runEtcdDiagnostics()
	if len(diagIssues) > 0 {
		performanceIssues = append(performanceIssues, diagIssues...)
		details.WriteString("== ETCD Diagnostics ==\n")
		details.WriteString(diagStats)
		details.WriteString("\n\n")
	} else if diagStats != "" {
		details.WriteString("== ETCD Diagnostics ==\n")
		details.WriteString("ETCD diagnostic tests passed.\n")
		details.WriteString(diagStats)
		details.WriteString("\n\n")
	}

	// Check node resource usage for etcd pods
	resourceStats, resourceIssues := checkNodeResourceUsage()
	if len(resourceIssues) > 0 {
		performanceIssues = append(performanceIssues, resourceIssues...)
		details.WriteString("== Node Resource Usage ==\n")
		details.WriteString(resourceStats)
		details.WriteString("\n\n")
	} else {
		details.WriteString("== Node Resource Usage ==\n")
		details.WriteString("Node resource usage appears to be within normal ranges.\n")
		details.WriteString(resourceStats)
		details.WriteString("\n\n")
	}

	// Generate performance recommendation guide
	details.WriteString("== ETCD Performance Recommendations ==\n")
	details.WriteString(generatePerformanceGuide())
	details.WriteString("\n\n")

	// Determine final status based on findings
	if len(performanceIssues) == 0 {
		result := healthcheck.NewResult(
			c.ID(),
			types.StatusOK,
			"ETCD is performing optimally",
			types.ResultKeyNoChange,
		)
		result.Detail = details.String()
		return result, nil
	}

	// Create result with detailed message about performance issues
	status := types.StatusWarning
	resultKey := types.ResultKeyRecommended
	messagePrefix := "ETCD performance concerns"

	if hasCritical {
		status = types.StatusCritical
		resultKey = types.ResultKeyRequired
		messagePrefix = "ETCD critical performance issues"
	}

	result := healthcheck.NewResult(
		c.ID(),
		status,
		fmt.Sprintf("%s: %s", messagePrefix, strings.Join(performanceIssues[:minInt(3, len(performanceIssues))], ", ")),
		resultKey,
	)

	// Add recommendations
	result.AddRecommendation("Ensure etcd is using dedicated SSD or NVMe storage with high IOPS")
	result.AddRecommendation("Check for resource contention on nodes running etcd pods")
	result.AddRecommendation("Verify network latency between etcd nodes is below 2ms")
	result.AddRecommendation("Consider increasing resources allocated to etcd if experiencing CPU or memory pressure")
	result.AddRecommendation("Review the documentation at https://docs.openshift.com/container-platform/latest/scalability_and_performance/recommended-host-practices.html#recommended-etcd-practices_recommended-host-practices")

	// Add detailed information
	result.Detail = details.String()

	return result, nil
}

// checkCompactionPerformance checks for slow compaction issues
func checkCompactionPerformance() (string, []string) {
	var issues []string
	var details strings.Builder

	// Get recent logs with compaction info
	compactionOut, err := utils.RunCommand("oc", "logs", "--tail=500", "-n", "openshift-etcd", "-l", "app=etcd", "-c", "etcd", "|", "grep", "-i", "finished scheduled compaction")
	if err != nil {
		return "Could not retrieve compaction logs", issues
	}

	details.WriteString("Compaction performance logs:\n")
	if compactionOut == "" {
		details.WriteString("No compaction logs found in recent entries.\n")
		return details.String(), issues
	}

	details.WriteString(compactionOut)
	details.WriteString("\n")

	// Parse compaction times
	compactionPattern := regexp.MustCompile(`finished scheduled compaction.*took\s+(\d+\.\d+)(m?s)`)
	matches := compactionPattern.FindAllStringSubmatch(compactionOut, -1)

	slowCompactions := 0
	verySlowCompactions := 0
	var compactionTimes []float64

	for _, match := range matches {
		if len(match) >= 3 {
			duration, _ := strconv.ParseFloat(match[1], 64)
			unit := match[2]

			// Convert to milliseconds if necessary
			if unit == "ms" {
				// Already in ms
			} else {
				// Assuming it's in seconds
				duration *= 1000
			}

			compactionTimes = append(compactionTimes, duration)

			// Check thresholds
			if duration > 800 {
				verySlowCompactions++
			} else if duration > 100 {
				slowCompactions++
			}
		}
	}

	// Calculate average compaction time
	var avgCompactionTime float64
	if len(compactionTimes) > 0 {
		sum := 0.0
		for _, time := range compactionTimes {
			sum += time
		}
		avgCompactionTime = sum / float64(len(compactionTimes))
	}

	details.WriteString(fmt.Sprintf("\nCompaction statistics:\n"))
	details.WriteString(fmt.Sprintf("- Total compactions analyzed: %d\n", len(compactionTimes)))
	details.WriteString(fmt.Sprintf("- Average compaction time: %.2f ms\n", avgCompactionTime))
	details.WriteString(fmt.Sprintf("- Slow compactions (> 100ms): %d\n", slowCompactions))
	details.WriteString(fmt.Sprintf("- Very slow compactions (> 800ms): %d\n", verySlowCompactions))

	// For small clusters, compaction should be below 10ms
	// For large clusters, compaction should be below 100ms
	if verySlowCompactions > 0 {
		issues = append(issues, fmt.Sprintf("very slow etcd compactions detected (%d instances > 800ms)", verySlowCompactions))
		details.WriteString("\nWARNING: Very slow compactions detected. This indicates severe disk performance issues.\n")
	} else if slowCompactions > 3 { // Allow a few slow compactions as normal
		issues = append(issues, fmt.Sprintf("slow etcd compactions detected (%d instances > 100ms)", slowCompactions))
		details.WriteString("\nWARNING: Multiple slow compactions detected. This may indicate disk performance issues.\n")
	}

	if avgCompactionTime > 50 { // Average should be much lower than the occasional spikes
		issues = append(issues, fmt.Sprintf("high average etcd compaction time (%.1fms)", avgCompactionTime))
		details.WriteString("\nWARNING: Average compaction time is higher than expected.\n")
	}

	return details.String(), issues
}

// checkApplyEntriesPerformance checks for "took too long" issues
func checkApplyEntriesPerformance() (string, []string) {
	var issues []string
	var details strings.Builder

	// Get recent logs with "apply entries took too long" messages
	tookTooLongOut, err := utils.RunCommand("oc", "logs", "--tail=500", "-n", "openshift-etcd", "-l", "app=etcd", "-c", "etcd", "|", "grep", "-i", "took too long")
	if err != nil {
		return "Could not retrieve apply entries logs", issues
	}

	details.WriteString("Apply entries performance logs:\n")
	if tookTooLongOut == "" {
		details.WriteString("No 'took too long' messages found in recent logs (good).\n")
		return details.String(), issues
	}

	details.WriteString(tookTooLongOut)
	details.WriteString("\n")

	// Count occurrences
	lines := strings.Split(tookTooLongOut, "\n")
	numOccurrences := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			numOccurrences++
		}
	}

	// Look for time patterns like "(123.45ms)" to analyze severity
	timePattern := regexp.MustCompile(`took too long \(([0-9.]+)ms\)`)
	matches := timePattern.FindAllStringSubmatch(tookTooLongOut, -1)

	var highDelays int
	var veryHighDelays int
	for _, match := range matches {
		if len(match) >= 2 {
			delay, _ := strconv.ParseFloat(match[1], 64)
			if delay > 500 {
				veryHighDelays++
			} else if delay > 100 {
				highDelays++
			}
		}
	}

	details.WriteString(fmt.Sprintf("\nApply entries statistics:\n"))
	details.WriteString(fmt.Sprintf("- Total 'took too long' messages: %d\n", numOccurrences))
	details.WriteString(fmt.Sprintf("- High delay entries (> 100ms): %d\n", highDelays))
	details.WriteString(fmt.Sprintf("- Very high delay entries (> 500ms): %d\n", veryHighDelays))

	// Evaluate the severity
	if numOccurrences > 20 || veryHighDelays > 5 {
		issues = append(issues, fmt.Sprintf("excessive delays applying etcd entries (%d occurrences)", numOccurrences))
		details.WriteString("\nCRITICAL: Frequent delays applying etcd entries. This indicates serious disk performance issues.\n")
	} else if numOccurrences > 5 || highDelays > 2 {
		issues = append(issues, fmt.Sprintf("delays applying etcd entries (%d occurrences)", numOccurrences))
		details.WriteString("\nWARNING: Delays applying etcd entries. This indicates disk performance issues.\n")
	}

	return details.String(), issues
}

// checkHeartbeatPerformance checks for heartbeat issues
func checkHeartbeatPerformance() (string, []string) {
	var issues []string
	var details strings.Builder

	// Get recent logs with heartbeat issues
	heartbeatOut, err := utils.RunCommand("oc", "logs", "--tail=500", "-n", "openshift-etcd", "-l", "app=etcd", "-c", "etcd", "|", "grep", "-i", "failed to send out heartbeat")
	if err != nil {
		return "Could not retrieve heartbeat logs", issues
	}

	details.WriteString("Heartbeat performance logs:\n")
	if heartbeatOut == "" {
		details.WriteString("No heartbeat issues found in recent logs (good).\n")
		return details.String(), issues
	}

	details.WriteString(heartbeatOut)
	details.WriteString("\n")

	// Count occurrences
	lines := strings.Split(heartbeatOut, "\n")
	numOccurrences := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			numOccurrences++
		}
	}

	details.WriteString(fmt.Sprintf("\nHeartbeat statistics:\n"))
	details.WriteString(fmt.Sprintf("- Total heartbeat failure messages: %d\n", numOccurrences))

	// Look for "server is likely overloaded" messages
	overloadedOut, _ := utils.RunCommand("oc", "logs", "--tail=500", "-n", "openshift-etcd", "-l", "app=etcd", "-c", "etcd", "|", "grep", "-i", "server is likely overloaded")
	overloadedCount := 0
	if overloadedOut != "" {
		overloadedLines := strings.Split(overloadedOut, "\n")
		for _, line := range overloadedLines {
			if strings.TrimSpace(line) != "" {
				overloadedCount++
			}
		}
		details.WriteString(fmt.Sprintf("- 'Server is likely overloaded' messages: %d\n", overloadedCount))
	}

	// Evaluate the severity
	if numOccurrences > 20 || overloadedCount > 5 {
		issues = append(issues, fmt.Sprintf("frequent heartbeat failures (%d occurrences)", numOccurrences))
		details.WriteString("\nCRITICAL: Frequent heartbeat failures. This indicates resource constraints or disk performance issues.\n")
	} else if numOccurrences > 5 {
		issues = append(issues, fmt.Sprintf("heartbeat failures (%d occurrences)", numOccurrences))
		details.WriteString("\nWARNING: Heartbeat failures detected. This indicates resource constraints or disk performance issues.\n")
	}

	return details.String(), issues
}

// checkClockDriftIssues checks for clock drift between etcd nodes
func checkClockDriftIssues() (string, []string) {
	var issues []string
	var details strings.Builder

	// Get recent logs with clock difference issues
	clockDriftOut, err := utils.RunCommand("oc", "logs", "--tail=500", "-n", "openshift-etcd", "-l", "app=etcd", "-c", "etcd", "|", "grep", "-i", "clock difference")
	if err != nil {
		return "Could not retrieve clock drift logs", issues
	}

	details.WriteString("Clock drift logs:\n")
	if clockDriftOut == "" {
		details.WriteString("No clock drift issues found in recent logs (good).\n")
		return details.String(), issues
	}

	details.WriteString(clockDriftOut)
	details.WriteString("\n")

	// Count occurrences
	lines := strings.Split(clockDriftOut, "\n")
	numOccurrences := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			numOccurrences++
		}
	}

	// Extract drift values
	driftPattern := regexp.MustCompile(`clock difference.*?\[([0-9.]+)s > ([0-9.]+)s\]`)
	matches := driftPattern.FindAllStringSubmatch(clockDriftOut, -1)

	var highDrifts int
	for _, match := range matches {
		if len(match) >= 3 {
			drift, _ := strconv.ParseFloat(match[1], 64)
			threshold, _ := strconv.ParseFloat(match[2], 64)
			if drift > threshold*2 {
				highDrifts++
			}
		}
	}

	details.WriteString(fmt.Sprintf("\nClock drift statistics:\n"))
	details.WriteString(fmt.Sprintf("- Total clock drift messages: %d\n", numOccurrences))
	details.WriteString(fmt.Sprintf("- High drift instances (>2x threshold): %d\n", highDrifts))

	// Check chrony status on nodes
	chronyOut, _ := utils.RunCommand("oc", "debug", "node/$(oc get nodes -l node-role.kubernetes.io/master -o name | head -1)", "--", "chroot", "/host", "chronyc", "tracking")
	if chronyOut != "" {
		details.WriteString("\nChrony status on a master node:\n")
		details.WriteString(chronyOut)
	}

	// Evaluate the severity
	if numOccurrences > 10 || highDrifts > 5 {
		issues = append(issues, fmt.Sprintf("significant clock drift between etcd nodes (%d occurrences)", numOccurrences))
		details.WriteString("\nWARNING: Significant clock drift detected. Verify chrony configuration across nodes.\n")
	} else if numOccurrences > 2 {
		issues = append(issues, "clock drift between etcd nodes")
		details.WriteString("\nADVISORY: Some clock drift detected. Monitor chrony synchronization.\n")
	}

	return details.String(), issues
}

// runEtcdDiagnostics attempts to run etcd performance diagnostics
func runEtcdDiagnostics() (string, []string) {
	var issues []string
	var details strings.Builder

	// Try to run etcdctl check perf command
	perfOut, err := utils.RunCommandWithRetry(2, time.Second, "oc", "exec", "-n", "openshift-etcd", "-c", "etcd", "$(oc get pods -n openshift-etcd -l app=etcd -o name | head -1)", "--", "etcdctl", "--command-timeout=30s", "check", "perf")
	if err != nil {
		// Try alternative approach
		perfOut, err = utils.RunCommandWithRetry(2, time.Second, "oc", "debug", "node/$(oc get nodes -l node-role.kubernetes.io/master -o name | head -1)", "--", "chroot", "/host", "crictl", "exec", "$(crictl ps -q --label io.kubernetes.container.name=etcd)", "etcdctl", "--command-timeout=30s", "check", "perf")

		if err != nil {
			// Both approaches failed
			return "", issues
		}
	}

	details.WriteString("ETCD Diagnostics Output:\n")
	details.WriteString(perfOut)
	details.WriteString("\n")

	// Analyze the diagnostic output
	if strings.Contains(perfOut, "FAIL") {
		failLines := 0
		for _, line := range strings.Split(perfOut, "\n") {
			if strings.Contains(line, "FAIL") {
				failLines++
			}
		}

		issues = append(issues, fmt.Sprintf("etcd performance tests failed (%d failures)", failLines))
		details.WriteString(fmt.Sprintf("\nWARNING: %d performance test failures detected.\n", failLines))
	} else if strings.Contains(perfOut, "WARNING") {
		warningLines := 0
		for _, line := range strings.Split(perfOut, "\n") {
			if strings.Contains(line, "WARNING") {
				warningLines++
			}
		}

		issues = append(issues, fmt.Sprintf("etcd performance tests showed warnings (%d warnings)", warningLines))
		details.WriteString(fmt.Sprintf("\nADVISORY: %d performance test warnings detected.\n", warningLines))
	}

	// Try to extract fsync performance data
	fsyncPattern := regexp.MustCompile(`99th percentile of fsync is (\d+) ns`)
	fsyncMatch := fsyncPattern.FindStringSubmatch(perfOut)

	if len(fsyncMatch) >= 2 {
		fsyncTimeNs, _ := strconv.Atoi(fsyncMatch[1])
		fsyncTimeMs := float64(fsyncTimeNs) / 1000000.0

		details.WriteString(fmt.Sprintf("\n99th percentile fsync time: %.2f ms\n", fsyncTimeMs))

		// Evaluate against recommended threshold (10ms)
		if fsyncTimeMs > 10.0 {
			issues = append(issues, fmt.Sprintf("high fsync latency (%.1f ms, recommended < 10ms)", fsyncTimeMs))
			details.WriteString("\nWARNING: 99th percentile fsync time exceeds recommended threshold of 10ms.\n")
		}
	}

	return details.String(), issues
}

// checkNodeResourceUsage checks resource usage on nodes running etcd
func checkNodeResourceUsage() (string, []string) {
	var issues []string
	var details strings.Builder

	// Get etcd pod information
	podsOut, err := utils.RunCommand("oc", "get", "pods", "-n", "openshift-etcd", "-l", "app=etcd", "-o", "wide")
	if err != nil {
		return "Could not retrieve etcd pod information", issues
	}

	details.WriteString("ETCD Pods:\n")
	details.WriteString(podsOut)
	details.WriteString("\n\n")

	// Extract node names from pod output
	nodePattern := regexp.MustCompile(`etcd-[\w-]+\s+.*\s+Running\s+.*\s+([\w-]+)`)
	matches := nodePattern.FindAllStringSubmatch(podsOut, -1)

	var nodeNames []string
	for _, match := range matches {
		if len(match) >= 2 {
			nodeNames = append(nodeNames, match[1])
		}
	}

	if len(nodeNames) == 0 {
		details.WriteString("Could not extract node names from pod output.\n")
		return details.String(), issues
	}

	// Check resource usage on each node
	for _, nodeName := range nodeNames {
		nodeUsage, err := utils.RunCommand("oc", "adm", "top", "node", nodeName)
		if err != nil {
			details.WriteString(fmt.Sprintf("Could not get resource usage for node %s.\n", nodeName))
			continue
		}

		details.WriteString(fmt.Sprintf("Node %s resource usage:\n%s\n", nodeName, nodeUsage))

		// Extract CPU and memory usage
		usagePattern := regexp.MustCompile(`\s+(\d+)([m%])\s+(\d+)%`)
		usageMatch := usagePattern.FindStringSubmatch(nodeUsage)

		if len(usageMatch) >= 4 {
			cpuValue := usageMatch[1]
			cpuUnit := usageMatch[2]
			memValue := usageMatch[3]

			cpuUsage, _ := strconv.Atoi(cpuValue)
			memUsage, _ := strconv.Atoi(memValue)

			// Convert CPU usage to percentage if in millicores
			if cpuUnit == "m" {
				cpuUsage = cpuUsage / 10 // Rough approximation, depends on the node's CPU capacity
			}

			details.WriteString(fmt.Sprintf("CPU usage: ~%d%%, Memory usage: %d%%\n", cpuUsage, memUsage))

			if cpuUsage > 80 {
				issues = append(issues, fmt.Sprintf("high CPU usage (%d%%) on node %s", cpuUsage, nodeName))
				details.WriteString(fmt.Sprintf("WARNING: High CPU usage detected on node %s.\n", nodeName))
			}

			if memUsage > 85 {
				issues = append(issues, fmt.Sprintf("high memory usage (%d%%) on node %s", memUsage, nodeName))
				details.WriteString(fmt.Sprintf("WARNING: High memory usage detected on node %s.\n", nodeName))
			}
		}

		// Check CPU steal time if possible
		stealOut, _ := utils.RunCommand("oc", "debug", "node/"+nodeName, "--", "chroot", "/host", "bash", "-c", "top -b -n 1 | grep Cpu")
		if stealOut != "" {
			details.WriteString(fmt.Sprintf("CPU details from top command:\n%s\n", stealOut))

			// Look for CPU steal percentage
			stealPattern := regexp.MustCompile(`(\d+\.\d+) st`)
			stealMatch := stealPattern.FindStringSubmatch(stealOut)

			if len(stealMatch) >= 2 {
				stealPct, _ := strconv.ParseFloat(stealMatch[1], 64)
				details.WriteString(fmt.Sprintf("CPU steal time: %.1f%%\n", stealPct))

				if stealPct > 1.0 {
					issues = append(issues, fmt.Sprintf("CPU steal time (%.1f%%) on node %s", stealPct, nodeName))
					details.WriteString("WARNING: CPU steal time detected. This indicates CPU overcommitment on the hypervisor.\n")
				}
			}
		}
	}

	return details.String(), issues
}

// generatePerformanceGuide provides recommended settings for optimal etcd performance
func generatePerformanceGuide() string {
	var guide strings.Builder

	guide.WriteString("For optimal etcd performance, the following are recommended:\n\n")
	guide.WriteString("1. Storage requirements:\n")
	guide.WriteString("   - Use dedicated SSDs (preferably NVMe) for etcd\n")
	guide.WriteString("   - For cloud environments, use high-performance storage:\n")
	guide.WriteString("     * AWS: io1/io2/io2-block-express with at least 2000 IOPS\n")
	guide.WriteString("     * Azure: Premium SSD or Ultra Disk\n")
	guide.WriteString("     * GCP: SSD Persistent Disk\n")
	guide.WriteString("   - For on-premise: avoid shared storage, use local NVMe or high-performance SSD\n")
	guide.WriteString("   - Avoid VM snapshots which can severely impact I/O performance\n\n")

	guide.WriteString("2. Performance metrics to aim for:\n")
	guide.WriteString("   - WAL fsync 99th percentile: < 10ms (ideally < 5ms)\n")
	guide.WriteString("   - Sequential fsync IOPS:\n")
	guide.WriteString("     * Small clusters: at least 50 IOPS\n")
	guide.WriteString("     * Medium clusters: at least 300 IOPS\n")
	guide.WriteString("     * Large clusters: at least 500 IOPS\n")
	guide.WriteString("     * Heavy workloads: at least 800 IOPS\n\n")

	guide.WriteString("3. Node configuration:\n")
	guide.WriteString("   - Ensure sufficient CPU resources (avoid overcommitment)\n")
	guide.WriteString("   - Configure proper time synchronization with chrony\n")
	guide.WriteString("   - Network latency between etcd nodes should be < 2ms\n")
	guide.WriteString("   - Avoid placing etcd nodes across different data centers\n\n")

	guide.WriteString("4. Maintenance practices:\n")
	guide.WriteString("   - Regular defragmentation during maintenance windows\n")
	guide.WriteString("   - Clean up unused projects, secrets, deployments, and other resources\n")
	guide.WriteString("   - Evaluate the impact of installing new operators\n")

	return guide.String()
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
