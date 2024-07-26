// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package main denotes the entry point of finch CLI.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/runfinch/finch/pkg/command"
	"github.com/runfinch/finch/pkg/config"
	"github.com/runfinch/finch/pkg/flog"
	"github.com/runfinch/finch/pkg/fmemory"
	"github.com/runfinch/finch/pkg/lima/wrapper"
	"github.com/runfinch/finch/pkg/path"
	"github.com/runfinch/finch/pkg/support"
	"github.com/runfinch/finch/pkg/system"
	"github.com/runfinch/finch/pkg/version"
)

const finchRootCmd = "finch"

func main() {
	logger := flog.NewLogrus()
	stdLib := system.NewStdLib()
	fs := afero.NewOsFs()
	mem := fmemory.NewMemory()
	stdOut := os.Stdout
	if err := xmain(logger, stdLib, fs, stdLib, mem, stdOut); err != nil {
		logger.Fatal(err)
	}
}

func xmain(logger flog.Logger,
	ffd path.FinchFinderDeps,
	fs afero.Fs,
	loadCfgDeps config.LoadSystemDeps,
	mem fmemory.Memory,
	stdOut io.Writer,
) error {
	fp, err := path.FindFinch(ffd)
	if err != nil {
		return fmt.Errorf("failed to find the installation path of Finch: %w", err)
	}

	home, err := ffd.GetUserHome()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}
	finchRootPath, err := fp.FinchRootDir(ffd)
	if err != nil {
		return fmt.Errorf("failed to get finch root path: %w", err)
	}
	ecc := command.NewExecCmdCreator()
	fc, err := config.Load(
		fs,
		fp.ConfigFilePath(finchRootPath),
		logger,
		loadCfgDeps,
		mem,
		ecc,
	)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	return newApp(
		logger,
		fp,
		fs,
		fc,
		stdOut,
		home,
		finchRootPath,
		ecc,
	).Execute()
}

var newApp = func(
	logger flog.Logger,
	fp path.Finch,
	fs afero.Fs,
	fc *config.Finch,
	stdOut io.Writer,
	home,
	finchRootPath string,
	ecc command.Creator,
) *cobra.Command {
	usage := fmt.Sprintf("%v <command>", finchRootCmd)
	rootCmd := &cobra.Command{
		Use:           usage,
		Short:         "Finch: open-source container development tool",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.Version,
	}
	// TODO: Decide when to forward --debug to the dependencies
	// (e.g. nerdctl for container commands and limactl for VM commands).
	rootCmd.PersistentFlags().Bool("debug", false, "running under debug mode")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		// running commands under debug mode will print out debug logs
		debugMode, _ := cmd.Flags().GetBool("debug")
		if debugMode {
			logger.SetLevel(flog.Debug)
		}
		return nil
	}

	ncc := nerdctlCmdCreator(ecc, logger, fp, finchRootPath)
	lima := wrapper.NewLimaWrapper()
	supportBundleBuilder := support.NewBundleBuilder(
		logger,
		fs,
		support.NewBundleConfig(fp, finchRootPath),
		fp,
		ecc,
		ncc,
		lima,
	)

	// append nerdctl commands
	allCommands := initializeNerdctlCommands(ncc, ecc, logger, fs, fc)
	// append finch specific commands
	allCommands = append(allCommands,
		newVersionCommand(ncc, logger, stdOut),
		virtualMachineCommands(logger, fp, ncc, ecc, fs, fc, home, finchRootPath),
		newSupportBundleCommand(logger, supportBundleBuilder, ncc),
		newGenDocsCommand(rootCmd, logger, fs, system.NewStdLib()),
	)

	rootCmd.AddCommand(allCommands...)

	if err := configureNerdctl(fs); err != nil {
		logger.Fatal(err)
	}

	return rootCmd
}

func initializeNerdctlCommands(
	ncc command.NerdctlCmdCreator,
	ecc command.Creator,
	logger flog.Logger,
	fs afero.Fs,
	fc *config.Finch,
) []*cobra.Command {
	nerdctlCommandCreator := newNerdctlCommandCreator(ncc, ecc, system.NewStdLib(), logger, fs, fc)
	var allNerdctlCommands []*cobra.Command
	for cmdName, cmdDescription := range nerdctlCmds {
		allNerdctlCommands = append(allNerdctlCommands, nerdctlCommandCreator.create(cmdName, cmdDescription))
	}
	return allNerdctlCommands
}
