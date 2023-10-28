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

var (
	bmcs = make(BMCs)
)

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

	log.Printf("node %q is candidate", node.Name)

	bmcInfo := &bmc.BMCInfo{Endpoint: endpoint, User: user, Pass: pass}

	if isIdle(node) {
		log.Printf("node %q is idle", node.Name)
	}

	client, err := bmc.NewClient(bmcInfo)
	if err != nil {
		log.Printf("failed to create IPMI client: %v\n", err)
	}
	defer client.Close()

	// TODO: Login every 30 minutes
	err = c.wattTimeClient.Login()
	if err != nil {
		log.Printf("failed to login: %v\n", err)

		return
	}

	index, err := c.wattTimeClient.Index()
	if err != nil {
		log.Printf("failed to get index: %v\n", err)

		return
	}

	pod, err := podInQueueThatFits(c.clientset, index)
	if err != nil {
		log.Printf("failed to determine if a pod is in the queue: %v", err)

		return
	}

	if pod {
		log.Printf("pod(s) in queue that can fit node")

		// Ensure is powered on.
		log.Printf("node %q is idle", node.Name)
	} else {
		// ?
	}

	isPoweredOn, err := client.IsPoweredOn()
	if err != nil {
		log.Printf("failed to determine current power status of %q: %v", node.Name, err)

		return
	}

	// TODO: Make this configurable.
	if index > 50 {
		if isPoweredOn {
			log.Printf("index is %d%%, powering off %q", index, node.Name)

			client.PowerOff()
		} else {
			log.Printf("node is in desired power state (off): %q", node.Name)
		}
	} else {
		if isPoweredOn {
			log.Printf("node is in desired power state (on): %q", node.Name)
		} else {
			log.Printf("index is %d%%, powering on %q", index, node.Name)

			client.PowerOn()
		}
	}
}

func (c *NodeManager) nodeUpdate(old, new interface{}) {
	oldNode := old.(*v1.Node)
	newNode := new.(*v1.Node)

	needsUpdate := false

	if oldNode.Annotations[bmcEndpointAnnotation] != newNode.Annotations[bmcEndpointAnnotation] {
		needsUpdate = true
	}

	if oldNode.Annotations[bmcUserAnnotation] != newNode.Annotations[bmcUserAnnotation] {
		needsUpdate = true
	}

	if oldNode.Annotations[bmcPasswordAnnotation] != newNode.Annotations[bmcPasswordAnnotation] {
		needsUpdate = true
	}

	if needsUpdate {
		c.nodeAdd(newNode)
	}
}

func (c *NodeManager) nodeDelete(obj interface{}) {
	node := obj.(*v1.Node)

	delete(bmcs, node.Name)

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
		if pod.Status.Phase == v1.PodPending {
			if pod.Spec.Priority != nil && *pod.Spec.Priority >= int32(index) {
				return true, nil
			}
		}
	}

	return false, nil
}
