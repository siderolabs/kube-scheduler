package node

import (
	"context"
	"fmt"
	"log"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	klog "k8s.io/klog/v2"

	"github.com/siderolabs/kube-scheduler/pkg/bmc"
	"github.com/siderolabs/kube-scheduler/pkg/energy/watttime"
)

const bmcEndpointAnnotation = "bmc.siderolabs.com/endpoint"
const bmcUserAnnotation = "bmc.siderolabs.com/username"
const bmcPasswordAnnotation = "bmc.siderolabs.com/password"

type BMCs map[string]*bmc.BMCInfo

// NodeManager manages the power state of nodes.
type NodeManager struct {
	informerFactory informers.SharedInformerFactory
	nodeInformer    coreinformers.NodeInformer
	clientset       *kubernetes.Clientset
	wattTimeClient  *watttime.Client
}

// Run starts shared informers and waits for the shared informer cache to
// synchronize.
func (c *NodeManager) Run(stopCh <-chan struct{}) error {
	c.informerFactory.Start(stopCh)
	if !cache.WaitForCacheSync(stopCh, c.nodeInformer.Informer().HasSynced) {
		return fmt.Errorf("failed to sync")
	}

	return nil
}

func (c *NodeManager) nodeAdd(obj interface{}) {
	node := obj.(*v1.Node)

	endpoint, ok := node.Annotations[bmcEndpointAnnotation]
	if !ok {
		return
	}

	user, ok := node.Annotations[bmcUserAnnotation]
	if !ok {
		return
	}

	pass, ok := node.Annotations[bmcPasswordAnnotation]
	if !ok {
		return
	}

	log.Printf("node %q has BMC annotations", node.Name)

	bmcInfo := &bmc.BMCInfo{Endpoint: endpoint, User: user, Pass: pass}

	client, err := bmc.NewClient(bmcInfo)
	if err != nil {
		log.Printf("failed to create IPMI client: %v\n", err)
	}
	defer client.Close()

	index, err := c.wattTimeClient.Index()
	if err != nil {
		log.Printf("failed to get index: %v\n", err)

		return
	}

	podIsInQueueThatFits, err := podInQueueThatFits(c.clientset, index)
	if err != nil {
		log.Printf("failed to determine if a pod is in the queue: %v", err)

		return
	}

	isPoweredOn, err := client.IsPoweredOn()
	if err != nil {
		log.Printf("failed to determine current power status of %q: %v", node.Name, err)

		return
	}

	if podIsInQueueThatFits {
		log.Printf("pod(s) in queue that can fit node")

		if isPoweredOn {
			// Nothing to do.
			return
		}

		if !isPoweredOn {
			log.Printf("index is %d%%, powering off %q", index, node.Name)

			err = client.PowerOn()
			if err != nil {
				log.Printf("failed to power on node %q", node.Name)
			}
		}
	} else {
		if isIdle(node) {
			log.Printf("node %q is idle, powering off", node.Name)

			err = client.PowerOff()
			if err != nil {
				log.Printf("failed to power off node %q", node.Name)
			}
		}
	}
}

func (c *NodeManager) nodeUpdate(old, new interface{}) {
	newNode := new.(*v1.Node)
	c.nodeAdd(newNode)
}

func (c *NodeManager) nodeDelete(obj interface{}) {
	node := obj.(*v1.Node)

	klog.Infof("node deleted: %q", node.Name)
}

// NewNodeManager creates a NodeController.
func NewNodeManager(informerFactory informers.SharedInformerFactory, clientset *kubernetes.Clientset, wattTimeClient *watttime.Client) (*NodeManager, error) {
	nodeInformer := informerFactory.Core().V1().Nodes()

	c := &NodeManager{
		informerFactory: informerFactory,
		nodeInformer:    nodeInformer,
		clientset:       clientset,
		wattTimeClient:  wattTimeClient,
	}
	_, err := nodeInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.nodeAdd,
			UpdateFunc: c.nodeUpdate,
			DeleteFunc: c.nodeDelete,
		},
	)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func isIdle(node *v1.Node) bool {
	return node.Status.Allocatable.Pods() == node.Status.Capacity.Pods()
}

func podInQueueThatFits(clientset *kubernetes.Clientset, index int) (bool, error) {
	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return false, err
	}

	for _, pod := range pods.Items {
		if pod.Spec.SchedulerName != "kube-scheduler-siderolabs" {
			continue
		}

		if pod.Status.Phase == v1.PodPending {
			if pod.Spec.Priority != nil && *pod.Spec.Priority >= int32(index) {
				return true, nil
			}
		}
	}

	return false, nil
}
