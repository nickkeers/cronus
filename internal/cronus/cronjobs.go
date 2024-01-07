package cronus

import (
	"bytes"
	"context"
	"fmt"
	"github.com/gorhill/cronexpr"
	"io"
	v1 "k8s.io/api/batch/v1"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	batch "k8s.io/client-go/listers/batch/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"reflect"
	"time"
)

type CronJobManager struct {
	clientset *kubernetes.Clientset
	lister    batch.CronJobLister
}

type CronJobInfo struct {
	Name            string
	Namespace       string
	CronScheduleRaw string // the cron expression

	LastScheduledTime  time.Time
	LastSuccessfulTime time.Time

	NextRunTime time.Time
}

func NewCronJobManager(stopCh <-chan struct{}) (*CronJobManager, error) {
	config, err := rest.InClusterConfig()

	if err != nil {
		return nil, err
	}

	fmt.Printf("Config: %+v\n", config)

	clientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		return nil, err
	}

	factory := informers.NewSharedInformerFactory(clientset, time.Minute*10)

	informer := factory.Batch().V1().CronJobs().Informer()

	factory.Start(stopCh)
	synced := factory.WaitForCacheSync(stopCh)

	if !synced[reflect.TypeOf(&v1.CronJob{})] {
		return nil, fmt.Errorf("CronJob informer did not sync")
	}

	// we have to do something with the informer, maybe this can be useful later?
	_, err = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{AddFunc: func(obj interface{}) {
		obj, ok := obj.(*v1.CronJob)

		if !ok {
			fmt.Println("cronjob added wasn't ok")
		}
	}})

	if err != nil {
		return nil, err
	}

	cronFactory := factory.Batch().V1().CronJobs()

	cronLister := batch.NewCronJobLister(cronFactory.Informer().GetIndexer())

	return &CronJobManager{
		clientset: clientset,
		lister:    cronLister,
	}, nil
}

func (c *CronJobManager) ListCronJobs() (*[]CronJobInfo, error) {
	jobs, err := c.lister.List(labels.Everything())

	if err != nil {
		return nil, err
	}

	cronJobInfos := make([]CronJobInfo, 0)

	for _, job := range jobs {
		cronJobInfos = append(cronJobInfos, CronJobInfo{
			Name:               job.Name,
			Namespace:          job.Namespace,
			CronScheduleRaw:    job.Spec.Schedule,
			LastScheduledTime:  job.Status.LastScheduleTime.Time,
			LastSuccessfulTime: job.Status.LastSuccessfulTime.Time,
			NextRunTime:        cronexpr.MustParse(job.Spec.Schedule).Next(time.Now()),
		})
	}

	return &cronJobInfos, nil
}

type PodDetails struct {
	Name      string
	Namespace string
}

// GetPods retrieves Pods for each Job associated with the provided CronJobs.
func (c *CronJobManager) GetPods(cronJobs []CronJobInfo) (map[string][]PodDetails, error) {
	podsMap := make(map[string][]PodDetails)
	var errorsList []error

	for _, cronJob := range cronJobs {
		// First, get the Jobs for the CronJob
		jobs, err := c.GetJobsForCronJob(cronJob.Name, cronJob.Namespace)
		if err != nil {
			errorsList = append(errorsList, err)
			continue
		}

		for _, jobName := range jobs {
			// For each Job, list the Pods
			podList, err := c.clientset.CoreV1().Pods(cronJob.Namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: fmt.Sprintf("job-name=%s", jobName),
			})
			if err != nil {
				errorsList = append(errorsList, err)
				continue
			}

			var pods []PodDetails
			for _, pod := range podList.Items {
				pods = append(pods, PodDetails{
					Name:      pod.Name,
					Namespace: pod.Namespace,
				})
			}

			podsMap[jobName] = pods
		}
	}

	if len(errorsList) > 0 {
		// Combine all errors into a single error
		return nil, fmt.Errorf("errors occurred while getting pods: %v", errorsList)
	}

	return podsMap, nil
}

// GetJobsForCronJob lists the names of Jobs that were created by a specific CronJob.
func (c *CronJobManager) GetJobsForCronJob(cronJobName, namespace string) ([]string, error) {
	// List all Jobs in the namespace
	jobs, err := c.clientset.BatchV1().Jobs(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var jobNames []string
	for _, job := range jobs.Items {
		// Check if the CronJob is the owner of this Job
		for _, ref := range job.OwnerReferences {
			if ref.Kind == "CronJob" && ref.Name == cronJobName {
				jobNames = append(jobNames, job.Name)
				break
			}
		}
	}

	return jobNames, nil
}

// GetPodLogs fetches logs for all Pods associated with a given CronJob.
func (c *CronJobManager) GetPodLogs(cronJobName, namespace string) (map[string]string, error) {
	// First, get all the Pods for the CronJob
	cronJobInfo := []CronJobInfo{{Name: cronJobName, Namespace: namespace}}
	podsMap, err := c.GetPods(cronJobInfo)
	if err != nil {
		return nil, fmt.Errorf("error getting pods for cron job %s: %v", cronJobName, err)
	}

	logsMap := make(map[string]string)
	for jobName, pods := range podsMap {
		for _, pod := range pods {
			logContent, err := c.fetchPodLog(pod.Name, namespace)
			if err != nil {
				fmt.Printf("Error fetching logs for pod %s: %v\n", pod.Name, err)
				continue
			}
			logsMap[fmt.Sprintf("%s/%s", jobName, pod.Name)] = *logContent
		}
	}

	return logsMap, nil
}

// fetchPodLog is a helper function to get logs for a single Pod.
func (c *CronJobManager) fetchPodLog(podName, namespace string) (*string, error) {
	req := c.clientset.CoreV1().Pods(namespace).GetLogs(podName, &v12.PodLogOptions{})
	podLogs, err := req.Stream(context.Background())
	if err != nil {
		return nil, err
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return nil, err
	}
	str := buf.String()

	return &str, nil
}
