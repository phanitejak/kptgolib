// Package leaderselector to select leader select among application POD instances and stop/start leader apecific shedules
package leaderselector

import (
	"context"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/phanitejak/kptgolib/tracing"
	uuid "github.com/satori/go.uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

// LeaderChangeCallback abstracts interfaces expected of the application to handle.
type LeaderChangeCallback interface {
	Start()
	Stop()
}

// LeaderSelector is using k3 client API to select a leader.
type LeaderSelector struct {
	leaseLock      *resourcelock.LeaseLock
	logger         *tracing.Logger
	leaderCallback LeaderChangeCallback
	stopChannel    chan bool
	envConf        *leaderConfig
	lockName       string
}

// config contains expected environmental variables.
type leaderConfig struct {
	KubeconfigPath     string `envconfig:"KUBE_CONFIG_PATH" default:""`
	LeaseLockNamespace string `envconfig:"K8S_LEASE_LOCK_NAMESPACE" default:"neo"`
	LeaseDuration      int    `envconfig:"K8S_LEASE_DURATION" default:"60"`
	RenewDeadline      int    `envconfig:"K8S_LEASE_RENEW_DEADLINE" default:"15"`
	RetryPeriod        int    `envconfig:"K8S_LEASE_RETY_PERIOD" default:"5"`
}

// NewLeaderSelectorFromEnv instantiates new selector object.
func NewLeaderSelectorFromEnv(logger *tracing.Logger, lockName string, leaderCallback LeaderChangeCallback) (*LeaderSelector, error) {
	envConf := &leaderConfig{}
	if err := envconfig.Process("", envConf); err != nil {
		return nil, err
	}

	leaseLock, err := newLeaselock(lockName, envConf)
	if err != nil {
		return nil, err
	}

	return &LeaderSelector{
		leaseLock:      leaseLock,
		logger:         logger,
		leaderCallback: leaderCallback,
		stopChannel:    make(chan bool, 1),
		envConf:        envConf,
		lockName:       lockName,
	}, nil
}

func newLeaselock(lockName string, env *leaderConfig) (*resourcelock.LeaseLock, error) {
	client, err := newClientsetFromKubernetesConfig(env.KubeconfigPath)
	if err != nil {
		return nil, err
	}

	// generate random id for each instance in resourcelock
	id := uuid.NewV4()

	// we use the Lease lock type since edits to Leases are less common
	// and fewer objects in the cluster watch "all Leases".
	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      lockName,
			Namespace: env.LeaseLockNamespace,
		},
		Client: client.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: id.String(),
		},
	}
	return lock, nil
}

func newClientsetFromKubernetesConfig(kubeConfig string) (*kubernetes.Clientset, error) {
	config, err := buildKubeconfig(kubeConfig)
	if err != nil {
		return nil, err
	}
	clientset := kubernetes.NewForConfigOrDie(config)
	return clientset, nil
}

func buildKubeconfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
		return cfg, nil
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// Start is starting the leader selection process.
func (l *LeaderSelector) Start(ctx context.Context) {
	l.logger.Debug("starting up leader selection for scheduler")
	// Implemented according to: https://github.com/kubernetes/client-go/blob/master/examples/leader-election

	// start the leader election code loop
	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock: l.leaseLock,
		// IMPORTANT: you MUST ensure that any code you have that
		// is protected by the lease must terminate **before**
		// you call cancel. Otherwise, you could have a background
		// loop still running and another process could
		// get elected before your background loop finished, violating
		// the stated goal of the lease.
		ReleaseOnCancel: true,
		LeaseDuration:   time.Duration(l.envConf.LeaseDuration) * time.Second,
		RenewDeadline:   time.Duration(l.envConf.RenewDeadline) * time.Second,
		RetryPeriod:     time.Duration(l.envConf.RetryPeriod) * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(_ context.Context) {
				// How should I use ctx here?
				// we're notified when we start - this is where you would
				// usually put your code
				l.logger.Infof("started leading with id: %s, starting scheduler", l.leaseLock.LockConfig.Identity)
				l.leaderCallback.Start()
			},
			OnStoppedLeading: func() {
				// we can do cleanup here
				l.logger.Infof("leader lost: %s, stopping scheduler", l.leaseLock.LockConfig.Identity)
				l.leaderCallback.Stop()
			},
			OnNewLeader: func(identity string) {
				// we're notified when new leader elected
				if identity == l.leaseLock.LockConfig.Identity {
					// I just got the lock
					return
				}
				l.logger.Infof("new leader elected: %s", identity)
			},
		},
	})
}
