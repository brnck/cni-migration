package app

import (
	"flag"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&o.NoDryRun, "no-dry-run", false, "Run the CLI tool _not_ in dry run mode. This will attempt to migrate your cluster.")
	fs.BoolVarP(&o.StepPreflight, "step-preflight", "0", false, "[0] - Install knet-stress and ensure connectivity.")
	fs.BoolVarP(&o.StepDisable, "step-disable", "1", false, "[1] - Descale cluster autoscaler to 0.")
	fs.BoolVarP(&o.StepPrepare, "step-prepare", "2", false, "[2] - Install required resource and prepare cluster.")
	fs.BoolVarP(&o.StepPriority, "step-priority", "3", false, "[3] - Set node selector on AWS VPC CNI daemon set.")
	fs.BoolVarP(&o.StepDeploy, "step-deploy", "4", false, "[4] - Deploy Cilium helm chart to the cluster")

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

func (o *Options) Validate() error {
	return nil
}
