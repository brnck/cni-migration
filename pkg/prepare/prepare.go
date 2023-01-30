package prepare

import (
	"context"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/brnck/cni-migration/pkg"
	"github.com/brnck/cni-migration/pkg/config"
	"github.com/brnck/cni-migration/pkg/util"
)

var _ pkg.Step = &Prepare{}

type Prepare struct {
	ctx context.Context
	log *logrus.Entry

	config  *config.Config
	client  *kubernetes.Clientset
	factory *util.Factory
}

func New(ctx context.Context, config *config.Config) pkg.Step {
	log := config.Log.WithField("step", "2-prepare")
	return &Prepare{
		log:     log,
		ctx:     ctx,
		config:  config,
		client:  config.Client,
		factory: util.New(ctx, log, config.Client),
	}
}

// Ready ensures that
// - Nodes have correct labels
// - The required resources exist
// - AWS VPC DaemonSet has been patched
func (p *Prepare) Ready() (bool, error) {
	nodes, err := p.client.CoreV1().Nodes().List(p.ctx, metav1.ListOptions{})
	if err != nil {
		return false, err
	}

	for _, n := range nodes.Items {
		if !p.hasRequiredLabel(n.Labels) {
			return false, nil
		}
	}

	p.log.Info("step 2 ready")

	return true, nil
}

// Run will ensure that
// - Node have correct labels
// - The required resources exist
// - Canal DaemonSet has been patched
func (p *Prepare) Run(dryrun bool) error {
	p.log.Infof("preparing migration...")

	nodes, err := p.client.CoreV1().Nodes().List(p.ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, n := range nodes.Items {
		if !p.hasRequiredLabel(n.Labels) {
			p.log.Infof("updating label on node %s", n.Name)

			if dryrun {
				continue
			}

			delete(n.Labels, p.config.Labels.Cilium)

			n.Labels[p.config.Labels.AwsVpcCni] = p.config.Labels.Value

			_, err := p.client.CoreV1().Nodes().Update(p.ctx, n.DeepCopy(), metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
	}

	if !dryrun {
		if err := p.factory.CheckKnetStress(); err != nil {
			return err
		}
	}

	return nil
}

func (p *Prepare) hasRequiredLabel(labels map[string]string) bool {
	if labels == nil {
		return false
	}

	_, cclOK := labels[p.config.Labels.AwsVpcCni]
	_, clOK := labels[p.config.Labels.Cilium]

	// If both true, or both false, does not have correct labels
	if cclOK == clOK {
		return false
	}

	return true
}
