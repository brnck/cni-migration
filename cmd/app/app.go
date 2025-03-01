package app

import (
	"context"
	"fmt"
	"github.com/brnck/cni-migration/pkg/delete"
	"github.com/brnck/cni-migration/pkg/deploy"
	"github.com/brnck/cni-migration/pkg/disable"
	"github.com/brnck/cni-migration/pkg/enable"
	"github.com/brnck/cni-migration/pkg/finalize"
	"github.com/brnck/cni-migration/pkg/preflight"
	"github.com/brnck/cni-migration/pkg/prepare"
	"github.com/brnck/cni-migration/pkg/priority"
	"github.com/brnck/cni-migration/pkg/remove"
	"github.com/brnck/cni-migration/pkg/update"
	"os"
	"reflect"

	// Load all auth plugins
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	cliflag "k8s.io/component-base/cli/flag"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"github.com/brnck/cni-migration/pkg"
	"github.com/brnck/cni-migration/pkg/config"
)

type NewFunc func(context.Context, *config.Config) pkg.Step
type ReadyFunc func() (bool, error)
type RunFunc func(bool) error

type Options struct {
	NoDryRun   bool
	LogLevel   string
	ConfigPath string

	StepAllPreMigration  bool
	StepAllPostMigration bool

	PreMigration struct {
		//0
		StepPreflight bool

		// 1
		StepDisable bool

		// 2
		StepPrepare bool

		// 3
		StepPriority bool

		// 4
		StepDeploy bool
	}

	PostMigration struct {
		// 5
		StepDelete bool

		// 6
		StepRemove bool

		// 7
		StepUpdate bool

		// 8
		StepFinalize bool

		// 9
		StepEnable bool
	}
}

const (
	long = `  cni-migration is a CLI tool to migrate a Kubernetes cluster from using AWS VPC CNI
  to Cilium. By default, the CLI tool will run in debug mode, and not perform any
  steps. All previous steps must be successful in order to run further steps.`
	examples = `
  # Execute a dry run of a full migration
  cni-migration --step-all

  # Perform a migration only the first 2 steps
  cni-migration --no-dry-run -1 -2

  # Perform a full live migration
  cni-migration --no-dry-run --step-all`
)

var preMigrationSteps []pkg.Step
var postMigrationSteps []pkg.Step

func NewRunCmd(ctx context.Context) *cobra.Command {
	var factory cmdutil.Factory

	o := new(Options)

	cmd := &cobra.Command{
		Use:     "cni-migration",
		Short:   "cni-migration is a CLI tool to migrate a Kubernetes cluster from using AWS VPC CNI to Cilium.",
		Long:    long,
		Example: examples,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Validate(); err != nil {
				return err
			}

			lvl, err := logrus.ParseLevel(o.LogLevel)
			if err != nil {
				return fmt.Errorf("failed to parse --log-level: %s", err)
			}

			config, err := config.New(o.ConfigPath, lvl, factory)
			if err != nil {
				return fmt.Errorf("failed to build config: %s", err)
			}

			for _, f := range []NewFunc{
				preflight.New,
				disable.New,
				prepare.New,
				priority.New,
				deploy.New,
			} {
				preMigrationSteps = append(preMigrationSteps, f(ctx, config))
			}

			for _, f := range []NewFunc{
				delete.New,
				remove.New,
				update.New,
				finalize.New,
				enable.New,
			} {
				postMigrationSteps = append(postMigrationSteps, f(ctx, config))
			}

			if err := run(config, o); err != nil {
				config.Log.Error(err)
				os.Exit(1)
			}

			return nil
		},
	}

	nfs := new(cliflag.NamedFlagSets)

	// pretty output from kube-apiserver
	usageFmt := "Usage:\n  %s\n\n"
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStderr(), *nfs, -1)
		return nil
	})

	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		fmt.Fprintf(cmd.OutOrStdout(), "Examples:%s\n", cmd.Example)
		cliflag.PrintSections(cmd.OutOrStdout(), *nfs, -1)
	})

	o.AddFlags(nfs.FlagSet("Option"))
	factory = AddKubeFlags(cmd, nfs.FlagSet("Client"))

	fs := cmd.Flags()
	for _, f := range nfs.FlagSets {
		fs.AddFlagSet(f)
	}

	return cmd
}

func run(config *config.Config, o *Options) error {
	dryrun := !o.NoDryRun

	if dryrun {
		config.Log = config.Log.WithField("dry-run", "true")
	}

	if o.StepAllPreMigration {
		return runAllSteps(preMigrationSteps, dryrun)
	}

	if o.StepAllPostMigration {
		return runAllSteps(postMigrationSteps, dryrun)
	}

	v := reflect.ValueOf(o.PreMigration)
	maxStep := resolveMaxStep(v)
	if maxStep != -1 {
		return runSteps(preMigrationSteps, maxStep, v, dryrun)
	}

	v = reflect.ValueOf(o.PostMigration)
	maxStep = resolveMaxStep(v)
	if resolveMaxStep(v) != -1 {
		return runSteps(postMigrationSteps, maxStep, v, dryrun)
	}

	config.Log.Info("steps successful.")

	return nil
}

func runAllSteps(steps []pkg.Step, dryrun bool) error {
	for i := 0; i < len(steps); i++ {
		if i > 0 {
			if err := ensureStepReady(i-1, steps[i-1]); err != nil {
				return err
			}
		}

		if err := steps[i].Run(dryrun); err != nil {
			return err
		}
	}

	return nil
}

func resolveMaxStep(flags reflect.Value) int {
	maxStep := -1

	for i := 0; i < flags.NumField(); i++ {
		if flags.Field(i).Bool() {
			maxStep = i
		}
	}

	return maxStep
}

func runSteps(steps []pkg.Step, maxStep int, flags reflect.Value, dryrun bool) error {
	for i := 0; i < flags.NumField(); i++ {
		if i > maxStep {
			break
		}

		if flags.Field(i).Bool() {

			if i > 0 {
				// Ensure previous step is read before proceeding
				if err := ensureStepReady(i-1, steps[i-1]); err != nil {
					return err
				}
			}

			if err := steps[i].Run(dryrun); err != nil {
				return err
			}

		} else {
			if err := ensureStepReady(i, steps[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

func ensureStepReady(i int, step pkg.Step) error {
	ready, err := step.Ready()
	if err != nil {
		return fmt.Errorf("step %d failed: %s", i, err)
	}

	if !ready {
		return fmt.Errorf("step %d not ready...", i)
	}

	return nil
}
