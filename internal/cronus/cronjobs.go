package cronus

import (
	"fmt"
	v1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	batch "k8s.io/client-go/listers/batch/v1"
	"k8s.io/client-go/rest"
	"reflect"
	"time"
)

type CronJobManager struct {
	clientset *kubernetes.Clientset
	lister    batch.CronJobLister
}

func NewCronJobManager(stopCh <-chan struct{}) (*CronJobManager, error) {
	config, err := rest.InClusterConfig()

	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		return nil, err
	}

	factory := informers.NewSharedInformerFactory(clientset, time.Second*10)
	factory.Start(stopCh)
	synced := factory.WaitForCacheSync(stopCh)

	if !synced[reflect.TypeOf(&v1.CronJob{})] {
		return nil, fmt.Errorf("CronJob informer did not sync")
	}

	cronFactory := factory.Batch().V1().CronJobs()

	cronLister := batch.NewCronJobLister(cronFactory.Informer().GetIndexer())

	return &CronJobManager{
		clientset: clientset,
		lister:    cronLister,
	}, nil
}

func (c *CronJobManager) ListCronJobs() ([]*v1.CronJob, error) {
	return c.lister.List(labels.Everything())
}
