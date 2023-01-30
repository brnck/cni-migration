package app

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&o.NoDryRun, "no-dry-run", false, "Run the CLI tool _not_ in dry run mode. This will attempt to migrate your cluster.")

	fs.BoolVar(&o.StepAllPreMigration, "pre-migration", false, "[pre-migration] Run all steps that are needed to do pre-migration actions")
	fs.BoolVar(&o.StepAllPostMigration, "post-migration", false, "[post-migration] Run all steps that are needed to do post-migration actions")

	fs.BoolVarP(&o.PreMigration.StepPreflight, "step-preflight", "0", false, "[0] - [pre-migration] Install knet-stress and ensure connectivity.")
	fs.BoolVarP(&o.PreMigration.StepDisable, "step-disable", "1", false, "[1] - [pre-migration] Descale cluster autoscaler to 0.")
	fs.BoolVarP(&o.PreMigration.StepPrepare, "step-prepare", "2", false, "[2] - [pre-migration] Install required resource and prepare cluster.")
	fs.BoolVarP(&o.PreMigration.StepPriority, "step-priority", "3", false, "[3] - [pre-migration] Set node selector on AWS VPC CNI daemon set.")
	fs.BoolVarP(&o.PreMigration.StepDeploy, "step-deploy", "4", false, "[4] - [pre-migration] Deploy Cilium helm chart to the cluster")

	fs.BoolVarP(&o.PostMigration.StepDelete, "step-delete", "5", false, "[5] - [post-migration] Remove AWS VPC CNI daemon set from the cluster")
	fs.BoolVarP(&o.PostMigration.StepRemove, "step-remove", "6", false, "[6] - [post-migration] Remove AWS VPC CNI node role label from the nodes")
	fs.BoolVarP(&o.PostMigration.StepUpdate, "step-update", "7", false, "[7] - [post-migration] Upgrade Cilium by removing node selector")
	fs.BoolVarP(&o.PostMigration.StepUpdate, "step-finalize", "8", false, "[8] - [post-migration] Remove Cilium node role label from the nodes")
	fs.BoolVarP(&o.PostMigration.StepEnable, "step-enable", "9", false, "[9] - [post-migration] Upscale cluster autoscaler back to configured replicas")

	fs.StringVarP(&o.LogLevel, "log-level", "v", "debug", "Set logging level [debug|info|warn|error|fatal]")
	fs.StringVarP(&o.ConfigPath, "config", "c", "config.yaml", "File path to the config path.")
}

func AddKubeFlags(cmd *cobra.Command, fs *pflag.FlagSet) cmdutil.Factory {
	kubeConfigFlags := genericclioptions.NewConfigFlags(true)
	kubeConfigFlags.AddFlags(fs)
	matchVersionKubeConfigFlags := cmdutil.NewMatchVersionFlags(kubeConfigFlags)
	matchVersionKubeConfigFlags.AddFlags(fs)
	factory := cmdutil.NewFactory(matchVersionKubeConfigFlags)

	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	flag.CommandLine.Parse([]string{})
	fakefs := flag.NewFlagSet("fake", flag.ExitOnError)
	klog.InitFlags(fakefs)
	if err := fakefs.Parse([]string{"-logtostderr=false"}); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	return factory
}

// Validate evaluate if flags are compliant with how the program should run
func (o *Options) Validate() error {
	preMigrationStepsActivated := false
	v := reflect.ValueOf(o.PreMigration)
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).Bool() {
			preMigrationStepsActivated = true
		}
	}

	postMigrationStepsActivated := false
	v = reflect.ValueOf(o.PostMigration)
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).Bool() {
			postMigrationStepsActivated = true
		}
	}

	err := errors.New("running pre-migration and post-migration steps at the same run is not allowed")
	if o.StepAllPreMigration && o.StepAllPostMigration {
		return err
	}
	if preMigrationStepsActivated && postMigrationStepsActivated {
		return err
	}
	if preMigrationStepsActivated && o.StepAllPostMigration {
		return err
	}
	if o.StepAllPreMigration && postMigrationStepsActivated {
		return err
	}

	return nil
}
