package disable

import (
	"context"
	"github.com/brnck/cni-migration/pkg"
	"github.com/brnck/cni-migration/pkg/config"
	"github.com/brnck/cni-migration/pkg/util"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var _ pkg.Step = &Disable{}

type Disable struct {
	ctx    context.Context
	client *kubernetes.Clientset
	config *config.Config

	log     *logrus.Entry
	factory *util.Factory
}

func New(ctx context.Context, config *config.Config) pkg.Step {
	log := config.Log.WithField("step", "1-disable")
	return &Disable{
		ctx:     ctx,
		log:     log,
		config:  config,
		client:  config.Client,
		factory: util.New(ctx, log, config.Client),
	}
}

// Ready ensures that
// - Cluster autoscaler is descaled to 0
func (d *Disable) Ready() (bool, error) {
	// Get scale by name
	d.log.Info("checking if autoscaler is descaled")
	scale, err := d.client.AppsV1().
		Deployments(d.config.ClusterAutoscaler.Namespace).
		GetScale(d.ctx, d.config.ClusterAutoscaler.DeploymentName, metav1.GetOptions{})

	if err != nil {
		return false, err
	}

	// Check if ready is 0
	if scale.Spec.Replicas != 0 && scale.Status.Replicas != 0 {
		return false, nil
	}

	if err = d.factory.CheckKnetStress(); err != nil {
		return false, err
	}

	d.log.Info("step 1 ready")

	return true, nil
}

// Run will ensure that
// - Cluster autoscaler is descaled to 0
func (d *Disable) Run(dryrun bool) error {
	d.log.Infof("disabling cluster autoscaler...")

	scale, err := d.client.AppsV1().
		Deployments(d.config.ClusterAutoscaler.Namespace).
		GetScale(d.ctx, d.config.ClusterAutoscaler.DeploymentName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	sc := *scale
	sc.Spec.Replicas = 0

	if !dryrun {
		_, err := d.client.AppsV1().
			Deployments(d.config.ClusterAutoscaler.Namespace).
			UpdateScale(d.ctx, d.config.ClusterAutoscaler.DeploymentName, &sc, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	if err = d.factory.CheckKnetStress(); err != nil {
		return err
	}

	d.log.Info("cluster autoscaler descaled to 0")

	return nil
}
