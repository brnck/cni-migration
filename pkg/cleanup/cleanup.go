package cleanup

import (
	"context"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/brnck/cni-migration/pkg"
	"github.com/brnck/cni-migration/pkg/config"
	"github.com/brnck/cni-migration/pkg/util"
)

var _ pkg.Step = &CleanUp{}

type CleanUp struct {
	ctx context.Context
	log *logrus.Entry

	config  *config.Config
	client  *kubernetes.Clientset
	factory *util.Factory
}

func New(ctx context.Context, config *config.Config) pkg.Step {
	log := config.Log.WithField("step", "5-cleanup")
	return &CleanUp{
		log:     log,
		ctx:     ctx,
		config:  config,
		client:  config.Client,
		factory: util.New(ctx, log, config.Client),
	}
}

// Ready ensures that
// - All migration resources have been cleaned up
func (c *CleanUp) Ready() (bool, error) {
	cleanUpResources, err := c.factory.Has(c.config.CleanUpResources)
	if err != nil || cleanUpResources {
		return !cleanUpResources, err
	}

	ds, err := c.client.AppsV1().DaemonSets("kube-system").Get(c.ctx, "cilium", metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	if _, ok := ds.Spec.Template.Spec.NodeSelector[c.config.Labels.Cilium]; ok {
		return false, nil
	}

	c.log.Info("step 5 ready")

	return true, nil
}

func (c *CleanUp) Run(dryrun bool) error {
	c.log.Info("cleaning up...")

	c.log.Info("deleting aws-node DaemonSet")
	if !dryrun {
		err := c.client.AppsV1().DaemonSets("kube-system").Delete(c.ctx, "aws-node", metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}
