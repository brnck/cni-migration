package priority

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"

	"github.com/brnck/cni-migration/pkg"
	"github.com/brnck/cni-migration/pkg/config"
	"github.com/brnck/cni-migration/pkg/util"
)

var _ pkg.Step = &Priority{}

type Priority struct {
	ctx context.Context
	log *logrus.Entry

	config  *config.Config
	client  *kubernetes.Clientset
	factory *util.Factory
}

func New(ctx context.Context, config *config.Config) pkg.Step {
	log := config.Log.WithField("step", "3-priority")
	return &Priority{
		log:     log,
		ctx:     ctx,
		config:  config,
		client:  config.Client,
		factory: util.New(ctx, log, config.Client),
	}
}

// Ready ensures that
// - AWS VPC CNI has node selector that schedules pod only on AWS VPC nodes
func (p *Priority) Ready() (bool, error) {
	patched, err := p.awsVpcCNIisPatched()
	if err != nil || !patched {
		return false, err
	}

	requiredResources, err := p.factory.Has(p.config.WatchedResources)
	if err != nil || !requiredResources {
		return false, err
	}

	p.log.Info("step 3 ready")

	return true, nil
}

// Run ensures that
// - AWS VPC CNI has node selector that schedules pod only on AWS VPC nodes
func (p *Priority) Run(dryrun bool) error {
	if !dryrun {
		if err := p.factory.CheckKnetStress(); err != nil {
			return err
		}
	}

	patched, err := p.awsVpcCNIisPatched()
	if err != nil {
		return err
	}

	if !patched {
		p.log.Infof("patching aws-node DaemonSet with node selector %s=%s",
			p.config.Labels.AwsVpcCni, p.config.Labels.Value)

		if !dryrun {
			if err := p.patchAwsVPC(); err != nil {
				return err
			}
		}
	}

	if !dryrun {
		if err := p.factory.WaitAllReady(p.config.WatchedResources); err != nil {
			return err
		}

		if err := p.factory.CheckKnetStress(); err != nil {
			return err
		}
	}

	return nil
}

func (p *Priority) patchAwsVPC() error {
	ds, err := p.client.AppsV1().
		DaemonSets(p.config.AwsVpcCni.Namespace).
		Get(p.ctx, p.config.AwsVpcCni.DaemonsetName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if ds.Spec.Template.Spec.NodeSelector == nil {
		ds.Spec.Template.Spec.NodeSelector = make(map[string]string)
	}
	ds.Spec.Template.Spec.NodeSelector[p.config.Labels.AwsVpcCni] = p.config.Labels.Value

	_, err = p.client.AppsV1().DaemonSets(p.config.AwsVpcCni.Namespace).Update(p.ctx, ds, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (p *Priority) awsVpcCNIisPatched() (bool, error) {
	ds, err := p.client.AppsV1().
		DaemonSets(p.config.AwsVpcCni.Namespace).
		Get(p.ctx, p.config.AwsVpcCni.DaemonsetName, metav1.GetOptions{})

	if err != nil {
		return false, err
	}

	if ds.Spec.Template.Spec.NodeSelector == nil {
		return false, nil
	}
	if v, ok := ds.Spec.Template.Spec.NodeSelector[p.config.Labels.AwsVpcCni]; !ok || v != p.config.Labels.Value {
		return false, nil
	}

	return true, nil
}
