package finalize

import (
	"context"
	"github.com/brnck/cni-migration/pkg"
	"github.com/brnck/cni-migration/pkg/config"
	"github.com/brnck/cni-migration/pkg/util"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var _ pkg.Step = &Finalize{}

type Finalize struct {
	ctx    context.Context
	config *config.Config
	client *kubernetes.Clientset

	log     *logrus.Entry
	factory *util.Factory
}

func New(ctx context.Context, config *config.Config) pkg.Step {
	log := config.Log.WithField("step", "8-finalize")
	return &Finalize{
		ctx:     ctx,
		log:     log,
		config:  config,
		client:  config.Client,
		factory: util.New(ctx, log, config.Client),
	}
}

// Ready ensures that
// - Cilium node role label is removed from the nodes
func (f *Finalize) Ready() (bool, error) {
	nodes, err := f.client.CoreV1().Nodes().List(f.ctx, metav1.ListOptions{})
	if err != nil {
		return false, err
	}

	for _, n := range nodes.Items {
		if !f.hasRequiredLabel(n.Labels) {
			return false, nil
		}
	}

	f.log.Info("step 8 ready")

	return true, nil
}

// Run will ensure that
// - Cilium node role label is removed from the nodes
func (f *Finalize) Run(dryrun bool) error {
	nodes, err := f.client.CoreV1().Nodes().List(f.ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, n := range nodes.Items {
		if f.hasRequiredLabel(n.Labels) {
			f.log.Infof("removing label on node %s", n.Name)

			if dryrun {
				continue
			}

			delete(n.Labels, f.config.Labels.Cilium)

			_, err := f.client.CoreV1().Nodes().Update(f.ctx, n.DeepCopy(), metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
	}

	if !dryrun {
		if err := f.factory.CheckKnetStress(); err != nil {
			return err
		}
	}

	return nil
}

func (f *Finalize) hasRequiredLabel(labels map[string]string) bool {
	if labels == nil {
		return false
	}

	// Check if label exists in label list
	// If label exists cclOK would be false
	// If label does not exist cclOK would be true
	if _, cclOK := labels[f.config.Labels.Cilium]; cclOK {
		return false
	}

	return true
}
