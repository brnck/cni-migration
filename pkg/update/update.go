package update

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

var _ pkg.Step = &Update{}

type Update struct {
	ctx        context.Context
	config     *config.Config
	client     *kubernetes.Clientset
	helmClient helmclient.Client

	log     *logrus.Entry
	factory *util.Factory
}

func New(ctx context.Context, config *config.Config) pkg.Step {
	log := config.Log.WithField("step", "7-deploy")
	return &Update{
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
func (u *Update) Ready() (bool, error) {
	u.log.Info("checking if cilium helm release exists")

	release, err := u.helmClient.GetRelease("cilium")
	if err != nil || release == nil {
		return false, err
	}
	if release.Info.Status.IsPending() {
		return false, nil
	}

	u.log.Info("cilium deployment exists...")

	return true, nil
}

// Run will ensure that
// - Cilium is deployed to the cluster
func (u *Update) Run(dryrun bool) error {
	u.log.Info("updating cilium helm release")

	if err := u.helmClient.AddOrUpdateChartRepo(repo.Entry{
		Name: "cilium",
		URL:  u.config.Cilium.RepoPath,
	}); err != nil {
		return err
	}

	values, err := os.ReadFile(u.config.CiliumPostMigration)
	if err != nil {
		return err
	}

	spec := &helmclient.ChartSpec{
		ReleaseName: u.config.Cilium.ReleaseName,
		ChartName:   u.config.Cilium.ChartName,
		Namespace:   u.config.Cilium.Namespace,
		ValuesYaml:  string(values),
		Version:     u.config.Cilium.Version,
		Timeout:     30 * time.Minute,
		DryRun:      dryrun,
	}
	if _, err = u.helmClient.UpgradeChart(u.ctx, spec, nil); err != nil {
		return err
	}

	backOff := 5

	for backOff != 0 {
		release, err := u.helmClient.GetRelease(spec.ReleaseName)
		if err != nil {
			return err
		}

		if !release.Info.Status.IsPending() {
			break
		}

		backOff -= 1
		time.Sleep(1 * time.Second)
	}

	u.log.Infof("waiting until %s will become ready", u.config.Cilium.ReleaseName)
	if err = u.factory.WaitDaemonSetReady(u.config.Cilium.Namespace, u.config.Cilium.ReleaseName); err != nil {
		return err
	}
	u.log.Infof("%s is ready", u.config.Cilium.ReleaseName)

	if err = u.factory.CheckKnetStress(); err != nil {
		return nil
	}

	u.log.Infof("upgraded %s in %s namespace", u.config.Cilium.ReleaseName, u.config.Cilium.Namespace)

	return nil
}
