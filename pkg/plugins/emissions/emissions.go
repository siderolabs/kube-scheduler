package emissions

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"

	"github.com/siderolabs/kube-scheduler/apis/config"
	"github.com/siderolabs/kube-scheduler/pkg/controllers/node"
	"github.com/siderolabs/kube-scheduler/pkg/controllers/pod"
	"github.com/siderolabs/kube-scheduler/pkg/energy/watttime"
)

// Emissions is a prefilter plugin that schedules pods based
// on the current emssisions score for a region.
// Implements framework.ScorePlugin
type Emissions struct {
	handle framework.Handle
	args   *config.EmissionsArgs
}

// Name is the name of the plugin used in the Registry and configurations.
const Name = "Emissions"

var _ = framework.PreFilterPlugin(&Emissions{})

// New initializes a new plugin and returns it.
func New(obj runtime.Object, h framework.Handle) (framework.Plugin, error) {
	args, ok := obj.(*config.EmissionsArgs)
	if !ok {
		return nil, fmt.Errorf("[Emissions] want args to be of type EmissionsArgs, got %T", obj)
	}

	klog.Infof("[Emissions] args received. %v", args)

	ctx := context.TODO()

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	wattTimeClient := watttime.NewClient(args.WattTimeUsername, args.WattTimePassword, args.WattTimeBA)

	nodeFactory := informers.NewSharedInformerFactory(clientset, 5*time.Minute)
	nodeManager, err := node.NewNodeManager(nodeFactory, clientset, wattTimeClient)
	if err != nil {
		klog.Fatal(err)
	}

	nodeManager.Run(ctx.Done())

	podFactory := informers.NewSharedInformerFactory(clientset, 5*time.Minute)
	podManager, err := pod.NewPodManager(podFactory, clientset, wattTimeClient)
	if err != nil {
		klog.Fatal(err)
	}

	podManager.Run(ctx.Done())

	return &Emissions{
		handle: h,
		args:   args,
	}, nil
}

// Name returns name of the plugin. It is used in logs, etc.
func (n *Emissions) Name() string {
	return Name
}

func (e *Emissions) PreFilter(ctx context.Context, state *framework.CycleState, pod *v1.Pod) (*framework.PreFilterResult, *framework.Status) {
	// A priority is required.
	if pod.Spec.Priority == nil {
		return nil, framework.NewStatus(framework.UnschedulableAndUnresolvable, "no priority set on pod")
	}

	wattTimeClient := watttime.NewClient(e.args.WattTimeUsername, e.args.WattTimePassword, e.args.WattTimeBA)

	err := wattTimeClient.Login()
	if err != nil {
		return nil, framework.NewStatus(framework.UnschedulableAndUnresolvable, fmt.Sprintf("failed to log in to WattTime: %v", err))
	}

	index, err := wattTimeClient.Index()
	if err != nil {
		return nil, framework.NewStatus(framework.UnschedulableAndUnresolvable, fmt.Sprintf("failed to get index from WattTime: %v", err))
	}

	if *pod.Spec.Priority > int32(index) {
		return nil, framework.NewStatus(framework.Success, "")
	}

	return nil, framework.NewStatus(framework.UnschedulableAndUnresolvable, "pod priority lower than index")
}

func (e *Emissions) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}
