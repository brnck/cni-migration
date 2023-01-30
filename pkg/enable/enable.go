package enable

import (
	"context"
	"github.com/brnck/cni-migration/pkg"
	"github.com/brnck/cni-migration/pkg/config"
	"github.com/brnck/cni-migration/pkg/util"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var _ pkg.Step = &Enable{}

type Enable struct {
	ctx    context.Context
	client *kubernetes.Clientset
	config *config.Config

	log     *logrus.Entry
	factory *util.Factory
}

func New(ctx context.Context, config *config.Config) pkg.Step {
	log := config.Log.WithField("step", "9-enable")
	return &Enable{
		ctx:     ctx,
		log:     log,
		config:  config,
		client:  config.Client,
		factory: util.New(ctx, log, config.Client),
	}
}

// Ready ensures that
// - Cluster autoscaler is descaled to 0
func (e *Enable) Ready() (bool, error) {
	// Get scale by name
	e.log.Info("checking if autoscaler is upscaled")
	scale, err := e.client.AppsV1().
		Deployments(e.config.ClusterAutoscaler.Namespace).
		GetScale(e.ctx, e.config.ClusterAutoscaler.DeploymentName, metav1.GetOptions{})

	if err != nil {
		return false, err
	}

	if scale.Spec.Replicas == 0 && scale.Status.Replicas == 0 {
		return false, nil
	}

	if err = e.factory.CheckKnetStress(); err != nil {
		return false, err
	}

	e.log.Info("step 9 ready")

	return true, nil
}

// Run will ensure that
// - Cluster autoscaler is descaled to 0
func (e *Enable) Run(dryrun bool) error {
	e.log.Infof("enabling cluster autoscaler")

	scale, err := e.client.AppsV1().
		Deployments(e.config.ClusterAutoscaler.Namespace).
		GetScale(e.ctx, e.config.ClusterAutoscaler.DeploymentName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if scale.Spec.Replicas != 0 && scale.Status.Replicas != 0 {
		e.log.Infof("cluster autoscaler already upscaled to %d", scale.Status.Replicas)

		return nil
	}

	sc := *scale
	sc.Spec.Replicas = int32(e.config.ClusterAutoscaler.Replicas)

	if !dryrun {
		_, err := e.client.AppsV1().
			Deployments(e.config.ClusterAutoscaler.Namespace).
			UpdateScale(e.ctx, e.config.ClusterAutoscaler.DeploymentName, &sc, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	e.log.Infof("waiting until %s will become ready", e.config.ClusterAutoscaler.DeploymentName)
	if err = e.factory.WaitDeploymentReady(e.config.ClusterAutoscaler.Namespace, e.config.ClusterAutoscaler.DeploymentName); err != nil {
		return err
	}
	e.log.Infof("%s is ready", e.config.ClusterAutoscaler.DeploymentName)

	if err = e.factory.CheckKnetStress(); err != nil {
		return err
	}

	e.log.Infof("cluster autoscaler upscaled to %d", e.config.ClusterAutoscaler.Replicas)

	return nil
}
