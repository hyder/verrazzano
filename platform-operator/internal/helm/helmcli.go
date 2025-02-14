// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	vzos "github.com/verrazzano/verrazzano/platform-operator/internal/os"
	"go.uber.org/zap"
)

// cmdRunner needed for unit tests
var runner vzos.CmdRunner = vzos.DefaultRunner{}

// Helm chart status values: unknown, deployed, uninstalled, superseded, failed, uninstalling, pending-install, pending-upgrade or pending-rollback
const ChartNotFound = "NotFound"
const ChartStatusDeployed = "deployed"
const ChartStatusPendingInstall = "pending-install"
const ChartStatusFailed = "failed"

// Package-level var and functions to allow overriding GetChartStatus for unit test purposes
type ChartStatusFnType func(releaseName string, namespace string) (string, error)

var chartStatusFn ChartStatusFnType = getChartStatus

// SetChartStatusFunction Override the chart status function for unit testing
func SetChartStatusFunction(f ChartStatusFnType) {
	chartStatusFn = f
}

// SetDefaultChartStatusFunction Reset the chart status function
func SetDefaultChartStatusFunction() {
	chartStatusFn = getChartStatus
}

// Package-level var and functions to allow overriding getReleaseState for unit test purposes
type releaseStateFnType func(releaseName string, namespace string) (string, error)

var releaseStateFn releaseStateFnType = getReleaseState

// SetChartStateFunction Override the chart state function for unit testing
func SetChartStateFunction(f releaseStateFnType) {
	releaseStateFn = f
}

// SetDefaultChartStateFunction Reset the chart state function
func SetDefaultChartStateFunction() {
	releaseStateFn = getChartStatus
}

// GetValues will run 'helm get values' command and return the output from the command.
func GetValues(log *zap.SugaredLogger, releaseName string, namespace string) ([]byte, error) {
	// Helm get values command will get the current set values for the installed chart.
	// The output will be used as input to the helm upgrade command.
	args := []string{"get", "values", releaseName}
	if namespace != "" {
		args = append(args, "--namespace")
		args = append(args, namespace)
	}

	cmd := exec.Command("helm", args...)
	log.Infof("Running command: %s", cmd.String())
	stdout, stderr, err := runner.Run(cmd)
	if err != nil {
		log.Errorf("helm get values for %s failed with stderr: %s", releaseName, string(stderr))
		return nil, err
	}

	//  Log get values output
	log.Infof("helm get values succeeded for %s", releaseName)

	return stdout, nil
}

// Upgrade will upgrade a Helm release with the specified charts.  The overrideFiles array
// are in order with the first files in the array have lower precedence than latter files.
func Upgrade(log *zap.SugaredLogger, releaseName string, namespace string, chartDir string, wait bool, dryRun bool, overrides string, overridesFiles ...string) (stdout []byte, stderr []byte, err error) {
	// Helm upgrade command will apply the new chart, but use all the existing
	// overrides that we used during the install.
	args := []string{"--install"}

	// Do not pass the --reuse-values arg to 'helm upgrade'.  Instead, pass the
	// values retrieved from 'helm get values' with the -f arg to 'helm upgrade'. This is a workaround to avoid
	// a failed helm upgrade that results from a nil reference.  The nil reference occurs when a default value
	// is added to a new chart and new chart references the new value.
	for _, overridesFileName := range overridesFiles {
		args = append(args, "-f")
		args = append(args, overridesFileName)
	}

	// Add the override strings
	if len(overrides) > 0 {
		args = append(args, "--set")
		args = append(args, overrides)
	}
	stdout, stderr, err = runHelm(log, releaseName, namespace, chartDir, "upgrade", wait, args, dryRun)
	if err != nil {
		return stdout, stderr, err
	}

	return stdout, stderr, nil
}

// Upgrade will upgrade a Helm release with the specified charts.  The overrideFiles array
// are in order with the first files in the array have lower precedence than latter files.
func Uninstall(log *zap.SugaredLogger, releaseName string, namespace string, dryRun bool) (stdout []byte, stderr []byte, err error) {
	// Helm upgrade command will apply the new chart, but use all the existing
	// overrides that we used during the install.
	args := []string{}

	stdout, stderr, err = runHelm(log, releaseName, namespace, "", "uninstall", false, args, dryRun)
	if err != nil {
		return stdout, stderr, err
	}

	return stdout, stderr, nil
}

// runHelm is a helper function to execute the helm CLI and return a result
func runHelm(log *zap.SugaredLogger, releaseName string, namespace string, chartDir string, operation string, wait bool, args []string, dryRun bool) (stdout []byte, stderr []byte, err error) {
	cmdArgs := []string{operation, releaseName}
	if len(chartDir) > 0 {
		cmdArgs = append(cmdArgs, chartDir)
	}
	if dryRun {
		cmdArgs = append(cmdArgs, "--dry-run")
	}
	if wait {
		cmdArgs = append(cmdArgs, "--wait")
	}
	if namespace != "" {
		cmdArgs = append(cmdArgs, "--namespace")
		cmdArgs = append(cmdArgs, namespace)
	}
	cmdArgs = append(cmdArgs, args...)

	// Try to upgrade several times.  Sometimes upgrade fails with "already exists" or "no deployed release".
	// We have seen from tests that doing a retry will eventually succeed if these 2 errors occur.
	const maxRetry = 5
	for i := 1; i <= maxRetry; i++ {
		cmd := exec.Command("helm", cmdArgs...)
		log.Infof("Running command: %s", cmd.String())
		stdout, stderr, err = runner.Run(cmd)
		if err == nil {
			log.Infof("helm %s for %s succeeded: %s", operation, releaseName, stdout)
			break
		}
		log.Errorf("helm %s for %s failed with stderr: %s", operation, releaseName, string(stderr))
		if i == maxRetry {
			return stdout, stderr, err
		}
		log.Infof("Retrying %s for %s, attempt %v", operation, releaseName, i+1)
	}

	//  Log upgrade output
	log.Infof("helm upgrade succeeded for %s", releaseName)
	return stdout, stderr, nil
}

// IsReleaseFailed Returns true if the chart release state is marked 'failed'
func IsReleaseFailed(releaseName string, namespace string) (bool, error) {
	log := zap.S()
	releaseStatus, err := releaseStateFn(releaseName, namespace)
	if err != nil {
		log.Errorf("Getting status for chart %s/%s failed with stderr: %v\n", namespace, releaseName, err)
		return false, err
	}
	return releaseStatus == ChartStatusFailed, nil
}

// IsReleaseInstalled returns true if the release is installed
func IsReleaseInstalled(releaseName string, namespace string) (found bool, err error) {
	log := zap.S()
	releaseStatus, err := chartStatusFn(releaseName, namespace)
	if err != nil {
		log.Errorf("Getting status for chart %s/%s failed with stderr: %v\n", namespace, releaseName, err)
		return false, err
	}
	switch releaseStatus {
	case ChartNotFound:
		log.Infof("Chart %s/%s not found", namespace, releaseName)
	case ChartStatusDeployed:
		return true, nil
	}
	return false, nil
}

// getChartStatus extracts the Helm deployment status of the specified chart from the JSON output as a string
func getChartStatus(releaseName string, namespace string) (string, error) {
	args := []string{"status", releaseName}
	if namespace != "" {
		args = append(args, "--namespace")
		args = append(args, namespace)
		args = append(args, "-o")
		args = append(args, "json")
	}
	cmd := exec.Command("helm", args...)
	stdout, stderr, err := runner.Run(cmd)
	if err != nil {
		if strings.Contains(string(stderr), "not found") {
			return ChartNotFound, nil
		}
		return "", fmt.Errorf("helm status for release %s failed with stderr: %s", releaseName, string(stderr))
	}

	var statusInfo map[string]interface{}
	if err := json.Unmarshal(stdout, &statusInfo); err != nil {
		return "", err
	}

	if info, infoFound := statusInfo["info"].(map[string]interface{}); infoFound {
		if status, statusFound := info["status"].(string); statusFound {
			return strings.TrimSpace(status), nil
		}
	}
	return "", fmt.Errorf("No chart status found for %s/%s", namespace, releaseName)
}

// getReleaseState extracts the release state from an "ls -o json" command for a specific release/namespace
func getReleaseState(releaseName string, namespace string) (string, error) {
	args := []string{"ls"}
	if namespace != "" {
		args = append(args, "--namespace")
		args = append(args, namespace)
		args = append(args, "-o")
		args = append(args, "json")
	}
	cmd := exec.Command("helm", args...)
	stdout, stderr, err := runner.Run(cmd)
	if err != nil {
		if strings.Contains(string(stderr), "not found") {
			return ChartNotFound, nil
		}
		return "", fmt.Errorf("helm status for release %s failed with stderr: %s", releaseName, string(stderr))
	}

	var statusInfo []map[string]interface{}
	if err := json.Unmarshal(stdout, &statusInfo); err != nil {
		return "", err
	}

	var status string
	for _, info := range statusInfo {
		release := info["name"].(string)
		if release == releaseName {
			status = info["status"].(string)
			break
		}
	}
	return strings.TrimSpace(status), nil
}

// SetCmdRunner sets the command runner as needed by unit tests
func SetCmdRunner(r vzos.CmdRunner) {
	runner = r
}

// SetDefaultRunner sets the command runner to default
func SetDefaultRunner() {
	runner = vzos.DefaultRunner{}
}
