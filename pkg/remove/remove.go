package remove

import (
	"context"
	"github.com/brnck/cni-migration/pkg"
	"github.com/brnck/cni-migration/pkg/config"
	"github.com/brnck/cni-migration/pkg/util"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var _ pkg.Step = &Remove{}

type Remove struct {
	ctx    context.Context
	config *config.Config
	client *kubernetes.Clientset

	log     *logrus.Entry
	factory *util.Factory
}

func New(ctx context.Context, config *config.Config) pkg.Step {
	log := config.Log.WithField("step", "6-remove")
	return &Remove{
		ctx:     ctx,
		log:     log,
		config:  config,
		client:  config.Client,
		factory: util.New(ctx, log, config.Client),
	}
}

// Ready ensures that
// - Label for AWS VPC CNI from the nodes
func (r *Remove) Ready() (bool, error) {
	nodes, err := r.client.CoreV1().Nodes().List(r.ctx, metav1.ListOptions{})
	if err != nil {
		return false, err
	}

	for _, n := range nodes.Items {
		if r.hasRequiredLabel(n.Labels) {
			return false, nil
		}
	}

	r.log.Info("step 6 ready")

	return true, nil
}

// Run will ensure that
// - Label for AWS VPC CNI is removed from the nodes
func (r *Remove) Run(dryrun bool) error {
	nodes, err := r.client.CoreV1().Nodes().List(r.ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, n := range nodes.Items {
		if r.hasRequiredLabel(n.Labels) {
			r.log.Infof("removing label on node %s", n.Name)

			if dryrun {
				continue
			}

			delete(n.Labels, r.config.Labels.AwsVpcCni)

			_, err := r.client.CoreV1().Nodes().Update(r.ctx, n.DeepCopy(), metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
	}

	if !dryrun {
		if err := r.factory.CheckKnetStress(); err != nil {
			return err
		}
	}

	return nil
}

func (r *Remove) hasRequiredLabel(labels map[string]string) bool {
	if labels == nil {
		return false
	}

	// Check if label exists in label list
	// If label exists cclOK would be false
	// If label does not exist cclOK would be true
	if _, cclOK := labels[r.config.Labels.AwsVpcCni]; !cclOK {
		return false
	}

	return true
}
