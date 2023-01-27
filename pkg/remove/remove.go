package remove

import (
	"context"
	"github.com/brnck/cni-migration/pkg"
	"github.com/brnck/cni-migration/pkg/config"
	"github.com/brnck/cni-migration/pkg/util"
	"github.com/sirupsen/logrus"
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
	log := config.Log.WithField("step", "5-remove")
	return &Remove{
		ctx:     ctx,
		log:     log,
		config:  config,
		client:  config.Client,
		factory: util.New(ctx, log, config.Client),
	}
}

// Ready ensures that
// - AWS VPC CNI daemon set is removed
func (r *Remove) Ready() (bool, error) {
	// TODO: Check if daemon set exists

	// TODO: Check if ready pods == 0

	r.log.Info("remove aws-node daemon set step is ready")

	return true, nil
}

// Run will ensure that
// - AWS VPC CNI daemon set is removed
func (r *Remove) Run(dryrun bool) error {
	// TODO: Check if daemon set exists

	// TODO: Check if ready pods == 0

	// TODO: Remove aws-node related components

	r.log.Info("aws-node daemon set removed")

	return nil
}
