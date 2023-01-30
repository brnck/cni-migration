package delete

import (
	"context"
	"github.com/brnck/cni-migration/pkg"
	"github.com/brnck/cni-migration/pkg/config"
	"github.com/brnck/cni-migration/pkg/util"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var _ pkg.Step = &Delete{}

type Delete struct {
	ctx    context.Context
	config *config.Config
	client *kubernetes.Clientset

	log     *logrus.Entry
	factory *util.Factory
}

func New(ctx context.Context, config *config.Config) pkg.Step {
	log := config.Log.WithField("step", "5-delete")
	return &Delete{
		ctx:     ctx,
		log:     log,
		config:  config,
		client:  config.Client,
		factory: util.New(ctx, log, config.Client),
	}
}

// Ready ensures that
// - AWS VPC CNI daemon set is removed
func (d *Delete) Ready() (bool, error) {
	d.log.Info("check aws-node daemon removal step is ready")

	exists, err := d.awsVpcCniExists()
	return !exists, err
}

// Run will ensure that
// - AWS VPC CNI daemon set is removed
func (d *Delete) Run(dryrun bool) error {
	exists, err := d.awsVpcCniExists()
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	if err = d.client.AppsV1().
		DaemonSets(d.config.AwsVpcCni.Namespace).
		Delete(d.ctx, d.config.AwsVpcCni.DaemonsetName, metav1.DeleteOptions{}); err != nil {
		return err
	}

	d.log.Info("aws-node daemon set removed")

	return nil
}

func (d *Delete) awsVpcCniExists() (bool, error) {
	ds, err := d.client.AppsV1().
		DaemonSets(d.config.AwsVpcCni.Namespace).
		Get(d.ctx, d.config.AwsVpcCni.DaemonsetName, metav1.GetOptions{})

	if err != nil && !apierrors.IsNotFound(err) {
		return false, err
	}

	if err != nil && apierrors.IsNotFound(err) {
		return false, nil
	}

	if ds.Status.NumberReady > 0 {
		return true, nil
	}

	return false, nil
}
