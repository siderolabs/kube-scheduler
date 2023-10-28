package pod

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/siderolabs/kube-scheduler/pkg/energy/watttime"
	v1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	klog "k8s.io/klog/v2"
)

// PodManager manages the power state of pods.
type PodManager struct {
	informerFactory informers.SharedInformerFactory
	podInformer     coreinformers.PodInformer
	clientset       *kubernetes.Clientset
	wattTimeClient  *watttime.Client
}

// Run starts shared informers and waits for the shared informer cache to
// synchronize.
func (c *PodManager) Run(stopCh <-chan struct{}) error {
	c.informerFactory.Start(stopCh)
	if !cache.WaitForCacheSync(stopCh, c.podInformer.Informer().HasSynced) {
		return fmt.Errorf("failed to sync")
	}

	return nil
}

func (c *PodManager) podAdd(obj interface{}) {
	pod := obj.(*v1.Pod)

	if pod.Spec.Priority == nil {
		log.Printf("priority not set on pod %s/%s", pod.Namespace, pod.Name)

		return
	}

	// TODO: Login every 30 minutes
	err := c.wattTimeClient.Login()
	if err != nil {
		log.Printf("failed to login: %v\n", err)
		return
	}

	index, err := c.wattTimeClient.Index()
	if err != nil {
		log.Printf("failed to get index: %v\n", err)

		return
	}

	log.Printf("pod (%s/%s) priority is %d, index is %d", pod.Namespace, pod.Name, *pod.Spec.Priority, index)

	if *pod.Spec.Priority < int32(index) && pod.Status.Phase != v1.PodPending {
		err = c.clientset.PolicyV1().Evictions(pod.Namespace).Evict(context.TODO(), &policy.Eviction{ObjectMeta: pod.ObjectMeta})
		if err != nil {
			log.Printf("failed to evict pod %q: %v\n", pod.Name, err)
		}

		log.Printf("evicted pod %s/%s", pod.Namespace, pod.Name)

		return
	}
}

func (c *PodManager) podUpdate(old, new interface{}) {
	oldPod := old.(*v1.Pod)
	newPod := new.(*v1.Pod)

	if oldPod.Spec.Priority != newPod.Spec.Priority {
		c.podAdd(newPod)
	}
}

func (c *PodManager) podDelete(obj interface{}) {
	pod := obj.(*v1.Pod)

	klog.Infof("pod deleted: %q", pod.Name)
}

// NewPodManager creates a PodManager.
func NewPodManager(informerFactory informers.SharedInformerFactory, clientset *kubernetes.Clientset, wattTimeClient *watttime.Client) (*PodManager, error) {
	podInformer := informerFactory.Core().V1().Pods()

	c := &PodManager{
		informerFactory: informerFactory,
		podInformer:     podInformer,
		clientset:       clientset,
		wattTimeClient:  wattTimeClient,
	}
	_, err := podInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.podAdd,
			UpdateFunc: c.podUpdate,
			DeleteFunc: c.podDelete,
		},
	)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func Run(clientset *kubernetes.Clientset, wattTimeClient *watttime.Client) {
	factory := informers.NewSharedInformerFactory(clientset, 5*time.Minute)
	manager, err := NewPodManager(factory, clientset, wattTimeClient)
	if err != nil {
		klog.Fatal(err)
	}

	stop := make(chan struct{})
	defer close(stop)

	go func() {
		err = manager.Run(stop)
		if err != nil {
			klog.Fatal(err)
		}
	}()

	select {}
}
