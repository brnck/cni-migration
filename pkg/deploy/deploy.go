package deploy

import (
	"context"
	"github.com/brnck/cni-migration/pkg"
	"github.com/brnck/cni-migration/pkg/config"
	"github.com/brnck/cni-migration/pkg/util"
	helmclient "github.com/mittwald/go-helm-client"
	"github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/client-go/kubernetes"
	"os"
	"time"
)

var _ pkg.Step = &Deploy{}

type Deploy struct {
	ctx        context.Context
	config     *config.Config
	client     *kubernetes.Clientset
	helmClient helmclient.Client

	log     *logrus.Entry
	factory *util.Factory
}

func New(ctx context.Context, config *config.Config) pkg.Step {
	log := config.Log.WithField("step", "4-deploy")
	return &Deploy{
		ctx:        ctx,
		log:        log,
		config:     config,
		client:     config.Client,
		helmClient: config.HelmClient,
		factory:    util.New(ctx, log, config.Client),
	}
}

// Ready ensures that
// - Cilium is deployed to the cluster
func (d *Deploy) Ready() (bool, error) {
	d.log.Info("checking if cilium helm release exists")

	release, err := d.helmClient.GetRelease("cilium")
	if err != nil || release == nil {
		return false, err
	}

	d.log.Info("cilium deployment exists...")

	return true, nil
}

// Run will ensure that
// - Cilium is deployed to the cluster
func (d *Deploy) Run(dryrun bool) error {
	if exists, _ := d.helmClient.GetRelease(d.config.Cilium.ReleaseName); exists != nil {
		d.log.Info("cilium already deployed. Skipping...")
		return nil
	}

	d.log.Info("deploying cilium helm release")

	if err := d.helmClient.AddOrUpdateChartRepo(repo.Entry{
		Name: "cilium",
		URL:  d.config.Cilium.RepoPath,
	}); err != nil {
		return err
	}

	values, err := os.ReadFile(d.config.CiliumPreMigration)
	if err != nil {
		return err
	}

	spec := &helmclient.ChartSpec{
		ReleaseName: d.config.Cilium.ReleaseName,
		ChartName:   d.config.Cilium.ChartName,
		Namespace:   d.config.Cilium.Namespace,
		ValuesYaml:  string(values),
		Version:     d.config.Cilium.Version,
		Timeout:     30 * time.Minute,
		DryRun:      dryrun,
	}
	if _, err = d.helmClient.InstallOrUpgradeChart(d.ctx, spec, nil); err != nil {
		return err
	}

	backOff := 5

	for backOff != 0 {
		release, err := d.helmClient.GetRelease(spec.ReleaseName)
		if err != nil {
			return err
		}

		if !release.Info.Status.IsPending() {
			break
		}

		backOff -= 1
		time.Sleep(1 * time.Second)
	}

	if err = d.factory.CheckKnetStress(); err != nil {
		return nil
	}

	d.log.Infof("%s deployed to %s namespace", d.config.Cilium.ReleaseName, d.config.Cilium.Namespace)

	return nil
}
