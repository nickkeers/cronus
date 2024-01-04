package cronus

import (
	v1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	batch "k8s.io/client-go/listers/batch/v1"
	"k8s.io/client-go/rest"
	"time"
)

type CronJobManager struct {
	clientset *kubernetes.Clientset
	lister    batch.CronJobLister
}

func NewCronJobManager() (*CronJobManager, error) {
	config, err := rest.InClusterConfig()

	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		return nil, err
	}

	factory := informers.NewSharedInformerFactory(clientset, time.Minute*2)

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
