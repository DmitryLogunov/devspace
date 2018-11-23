package cmd

import (
	"github.com/covexo/devspace/pkg/devspace/configure"
	"github.com/covexo/devspace/pkg/util/log"
	"github.com/spf13/cobra"
)

// RemoveCmd holds the information needed for the remove command
type RemoveCmd struct {
	syncFlags       *removeSyncCmdFlags
	portFlags       *removePortCmdFlags
	packageFlags    *removePackageCmdFlags
	deploymentFlags *removeDeploymentCmdFlags
}

type removeSyncCmdFlags struct {
	Selector      string
	LocalPath     string
	ContainerPath string
	RemoveAll     bool
}

type removePortCmdFlags struct {
	Selector  string
	RemoveAll bool
}

type removePackageCmdFlags struct {
	RemoveAll  bool
	Deployment string
}

type removeDeploymentCmdFlags struct {
	RemoveAll bool
}

func init() {
	cmd := &RemoveCmd{
		syncFlags:       &removeSyncCmdFlags{},
		portFlags:       &removePortCmdFlags{},
		packageFlags:    &removePackageCmdFlags{},
		deploymentFlags: &removeDeploymentCmdFlags{},
	}

	removeCmd := &cobra.Command{
		Use:   "remove",
		Short: "Changes devspace configuration",
		Long: `
	#######################################################
	################## devspace remove ####################
	#######################################################
	You can remove the following configuration with the
	remove command:
	
	* Sync paths (sync)
	* Forwarded ports (port)
	* Deployment (deployment)
	* Helm Packages (package)
	#######################################################
	`,
		Args: cobra.NoArgs,
	}

	rootCmd.AddCommand(removeCmd)

	removeSyncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Remove sync paths from the devspace",
		Long: `
	#######################################################
	############### devspace remove sync ##################
	#######################################################
	Remove sync paths from the devspace

	How to use:
	devspace remove sync --local=app
	devspace remove sync --container=/app
	devspace remove sync --selector=release=test
	devspace remove sync --all
	#######################################################
	`,
		Args: cobra.NoArgs,
		Run:  cmd.RunRemoveSync,
	}

	removeCmd.AddCommand(removeSyncCmd)

	removeSyncCmd.Flags().StringVar(&cmd.syncFlags.Selector, "selector", "", "Comma separated key=value selector list (e.g. release=test)")
	removeSyncCmd.Flags().StringVar(&cmd.syncFlags.LocalPath, "local", "", "Relative local path to remove")
	removeSyncCmd.Flags().StringVar(&cmd.syncFlags.ContainerPath, "container", "", "Absolute container path to remove")
	removeSyncCmd.Flags().BoolVar(&cmd.syncFlags.RemoveAll, "all", false, "Remove all configured sync paths")

	removePortCmd := &cobra.Command{
		Use:   "port",
		Short: "Removes forwarded ports from a devspace",
		Long: `
	#######################################################
	############### devspace remove port ##################
	#######################################################
	Removes port mappings from the devspace configuration:
	devspace remove port 8080,3000
	devspace remove port --selector=release=test
	devspace remove port --all
	#######################################################
	`,
		Args: cobra.MaximumNArgs(1),
		Run:  cmd.RunRemovePort,
	}

	removePortCmd.Flags().StringVar(&cmd.portFlags.Selector, "selector", "", "Comma separated key=value selector list (e.g. release=test)")
	removePortCmd.Flags().BoolVar(&cmd.portFlags.RemoveAll, "all", false, "Remove all configured ports")

	removeCmd.AddCommand(removePortCmd)
	removePackageCmd := &cobra.Command{
		Use:   "package",
		Short: "Removes forwarded ports from a devspace",
		Long: `
	#######################################################
	############## devspace remove package ################
	#######################################################
	Removes a package from the devspace:
	devspace remove package mysql
	devspace remove package mysql -d devspace-default
	#######################################################
	`,
		Args: cobra.MaximumNArgs(1),
		Run:  cmd.RunRemovePackage,
	}

	removePackageCmd.Flags().BoolVar(&cmd.packageFlags.RemoveAll, "all", false, "Remove all packages")
	removePackageCmd.Flags().StringVarP(&cmd.packageFlags.Deployment, "deployment", "d", "", "The deployment name to use")
	removeCmd.AddCommand(removePackageCmd)

	removeDeploymentCmd := &cobra.Command{
		Use:   "deployment",
		Short: "Removes one or all deployments from the devspace",
		Long: `
	#######################################################
	############ devspace remove deployment ###############
	#######################################################
	Removes one or all deployments from a devspace:
	devspace remove deployment devspace-default
	devspace remove deployment --all
	#######################################################
	`,
		Args: cobra.MaximumNArgs(1),
		Run:  cmd.RunRemoveDeployment,
	}

	removeDeploymentCmd.Flags().BoolVar(&cmd.deploymentFlags.RemoveAll, "all", false, "Remove all deployments")
	removeCmd.AddCommand(removeDeploymentCmd)
}

// RunRemoveDeployment executes the specified deployment
func (cmd *RemoveCmd) RunRemoveDeployment(cobraCmd *cobra.Command, args []string) {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}

	err := configure.RemoveDeployment(cmd.deploymentFlags.RemoveAll, name)
	if err != nil {
		log.Fatal(err)
	}
}

// RunRemovePackage executes the remove package command logic
func (cmd *RemoveCmd) RunRemovePackage(cobraCmd *cobra.Command, args []string) {
	err := configure.RemovePackage(cmd.packageFlags.RemoveAll, cmd.packageFlags.Deployment, args, log.GetInstance())
	if err != nil {
		log.Fatal(err)
	}
}

// RunRemoveSync executes the remove sync command logic
func (cmd *RemoveCmd) RunRemoveSync(cobraCmd *cobra.Command, args []string) {
	err := configure.RemoveSyncPath(cmd.syncFlags.RemoveAll, cmd.syncFlags.LocalPath, cmd.syncFlags.ContainerPath, cmd.syncFlags.Selector)
	if err != nil {
		log.Fatal(err)
	}
}

// RunRemovePort executes the remove port command logic
func (cmd *RemoveCmd) RunRemovePort(cobraCmd *cobra.Command, args []string) {
	err := configure.RemovePort(cmd.portFlags.RemoveAll, cmd.portFlags.Selector, args)
	if err != nil {
		log.Fatal(err)
	}
}
