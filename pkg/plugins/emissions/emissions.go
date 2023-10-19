package emissions

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"

	"github.com/siderolabs/kube-scheduler/apis/config"
)

// Emissions is a score plugin that favors nodes based on their
// network traffic amount. Nodes with less traffic are favored.
// Implements framework.ScorePlugin
type Emissions struct {
	handle framework.Handle
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

	return &Emissions{
		handle: h,
	}, nil
}

// Name returns name of the plugin. It is used in logs, etc.
func (n *Emissions) Name() string {
	return Name
}

func (e *Emissions) PreFilter(ctx context.Context, state *framework.CycleState, pod *v1.Pod) (*framework.PreFilterResult, *framework.Status) {
	// The absence of a priority implies the highest priority.
	if pod.Spec.Priority == nil {
		return nil, framework.NewStatus(framework.Success, "")
	}

	// TODO: We need to determine this number dynamically AND check if the current index allows for this.
	if *pod.Spec.Priority > 50 {
		return nil, framework.NewStatus(framework.Success, "")
	}

	// Return framework.UnschedulableAndUnresolvable to avoid any preemption attempts.
	return nil, framework.NewStatus(framework.UnschedulableAndUnresolvable, "low priority")
}

func (e *Emissions) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

// func (n *Emissions) NormalizeScore(ctx context.Context, state *framework.CycleState, pod *v1.Pod, scores framework.NodeScoreList) *framework.Status {
// 	var higherScore int64
// 	for _, node := range scores {
// 		if higherScore < node.Score {
// 			higherScore = node.Score
// 		}
// 	}

// 	for i, node := range scores {
// 		scores[i].Score = framework.MaxNodeScore - (node.Score * framework.MaxNodeScore / higherScore)
// 	}

// 	klog.Infof("[Emissions] Nodes final score: %v", scores)
// 	return nil
// }
