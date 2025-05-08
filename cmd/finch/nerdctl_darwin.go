// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build darwin

package finch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	dockerops "github.com/docker/docker/opts"
	"github.com/lima-vm/lima/pkg/networks"

	"github.com/runfinch/finch/pkg/command"
	"github.com/runfinch/finch/pkg/flog"
)

func convertToWSLPath(_ NerdctlCommandSystemDeps, _ string) (string, error) {
	return "", nil
}

var osAliasMap = map[string]string{}

var osArgHandlerMap = map[string]map[string]argHandler{}

var osCommandHandlerMap = map[string]commandHandler{}

func (nc *nerdctlCommand) GetCmdArgs() []string {
	return []string{"shell", limaInstanceName, "sudo", "-E"}
}

func resolveIP(host string, logger flog.Logger, _ command.Creator) (string, error) {
	parts := strings.SplitN(host, ":", 2)
	// If the IP Address is a string called "host-gateway", replace this value with the IP address that can be used to
	// access host from the containers.
	// TODO: make the host gateway ip configurable.
	var resolvedIP string
	if parts[1] == dockerops.HostGatewayName {
		resolvedIP = networks.SlirpGateway

		logger.Debugf(`Resolving special IP "host-gateway" to %q for host %q`, resolvedIP, parts[0])
		return fmt.Sprintf("%s:%s", parts[0], resolvedIP), nil
	}
	return host, nil
}

func handleBindMountPath(_ NerdctlCommandSystemDeps, m map[string]string) error {
	// For MacOS, ensure proper permissions for .vscode-server directory
	// Add rwx options to ensure executables work properly in bind mounts
	
	// Check if we have a source path
	sourcePath := ""
	if src, hasSource := m["source"]; hasSource {
		sourcePath = src
	} else if src, hasSource := m["src"]; hasSource {
		sourcePath = src
	}

	if sourcePath != "" {
		// Keep existing options if any
		if _, hasOptions := m["options"]; !hasOptions {
			// Set default options to ensure proper permissions
			m["options"] = "rbind,exec,rw"
		} else {
			// Ensure the mount has exec and rw options
			opts := strings.Split(m["options"], ",")
			hasExec := false
			hasRw := false
			for _, opt := range opts {
				if opt == "exec" {
					hasExec = true
				}
				if opt == "rw" {
					hasRw = true
				}
			}
			if !hasExec {
				m["options"] = m["options"] + ",exec"
			}
			if !hasRw {
				m["options"] = m["options"] + ",rw"
			}
		}

		// If this is a .vscode-server directory mount, ensure it has proper permissions
		if strings.Contains(sourcePath, ".vscode-server") {
			// Get base directory (/home/vscode)
			baseDir := filepath.Dir(filepath.Dir(sourcePath))
			
			// Ensure parent directories exist with proper permissions
			if err := os.MkdirAll(baseDir, 0777); err != nil {
				return fmt.Errorf("failed to create parent directories with proper permissions: %v", err)
			}
			
			// Ensure .vscode-server directory exists with proper permissions
			if err := os.MkdirAll(sourcePath, 0777); err != nil {
				return fmt.Errorf("failed to create .vscode-server directory with proper permissions: %v", err)
			}

			// Walk through the directory and ensure all files and subdirectories have proper permissions
			err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				// For directories, ensure rwx permissions
				if info.IsDir() {
					return os.Chmod(path, 0755)
				}
				// For executable files (like node), ensure rwx permissions
				if info.Mode()&0111 != 0 {
					return os.Chmod(path, 0755)
				}
				// For regular files, ensure rw permissions
				return os.Chmod(path, 0644)
			})
			if err != nil {
				return fmt.Errorf("failed to set permissions for .vscode-server contents: %v", err)
			}
			
			// Also set permissions on the parent .vscode directory
			vsCodeDir := filepath.Dir(sourcePath)
			if err := os.Chmod(vsCodeDir, 0777); err != nil {
				return fmt.Errorf("failed to set permissions for .vscode directory: %v", err)
			}
		}
	}
	
	return nil
}

func mapToString(m map[string]string) string {
	var parts []string
	for k, v := range m {
		part := fmt.Sprintf("%s=%s", k, v)
		parts = append(parts, part)
	}
	return strings.Join(parts, ",")
}
