package cronus

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	"github.com/gorhill/cronexpr"
	v1 "k8s.io/api/batch/v1"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	batch "k8s.io/client-go/listers/batch/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type CronJobManager struct {
	clientset *kubernetes.Clientset
	lister    batch.CronJobLister
}

type PodDetails struct {
	Name      string
	Namespace string
	Image     string
	Command   string
	StartTime *metav1.Time
}

type CronJobInfo struct {
	Name            string
	Namespace       string
	CronScheduleRaw string // the cron expression

	LastScheduledTime  time.Time
	LastSuccessfulTime time.Time

	NextRunTime time.Time
	Jobs        *[]JobInfo
}

type JobInfo struct {
	Name      string
	Namespace string
	StartTime time.Time
	Pods      *[]PodDetails
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

// GetPods retrieves Pods for each Job
func (c *CronJobManager) GetPods(jobs []JobInfo) (map[string][]PodDetails, error) {
	podsMap := make(map[string][]PodDetails)
	var errorsList []error

	for _, job := range jobs {
		// List all Pods in the namespace
		allPods, err := c.clientset.CoreV1().Pods(job.Namespace).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			errorsList = append(errorsList, err)
			continue
		}

		var pods []PodDetails
		for _, pod := range allPods.Items {
			// Check if the Pod's owner is the current Job
			for _, ref := range pod.OwnerReferences {
				if ref.Kind == "Job" && ref.Name == job.Name {
					pods = append(pods, PodDetails{
						Name:      pod.Name,
						Namespace: pod.Namespace,
						Image:     pod.Spec.Containers[0].Image,
						Command:   strings.Join(pod.Spec.Containers[0].Command, "\n"),
						StartTime: pod.Status.StartTime,
					})
					break
				}
			}
		}
		podsMap[job.Name] = pods
	}

	if len(errorsList) > 0 {
		// Combine all errors into a single error
		return nil, fmt.Errorf("errors occurred while getting pods: %v", errorsList)
	}

	return podsMap, nil
}

// GetJobsForCronJob lists the names of Jobs that were created by a specific CronJob.
func (c *CronJobManager) GetJobsForCronJob(cronJobName, namespace string) ([]JobInfo, error) {
	// List all Jobs in the namespace
	jobs, err := c.clientset.BatchV1().Jobs(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var jobInfo []JobInfo
	for _, job := range jobs.Items {
		// Check if the CronJob is the owner of this Job
		for _, ref := range job.OwnerReferences {
			if ref.Kind == "CronJob" && ref.Name == cronJobName {
				jobInfo = append(jobInfo, JobInfo{
					Name:      job.Name,
					Namespace: job.Namespace,
					StartTime: job.Status.StartTime.Time,
				})
				break
			}
		}
	}

	return jobInfo, nil
}

// GetPodLogs fetches logs for all Pods associated with a given CronJob.
func (c *CronJobManager) GetPodLogs(cronJobName, namespace string) (map[string]string, error) {
	jobs, err := c.GetJobsForCronJob(cronJobName, namespace)

	if err != nil {
		fmt.Printf("Error fetching pod logs for %s/%s\n", cronJobName, namespace)
		return map[string]string{}, err
	}

	podsMap, err := c.GetPods(jobs)
	if err != nil {
		return nil, fmt.Errorf("error getting pods for cron job %s: %v", cronJobName, err)
	}

	logsMap := make(map[string]string)
	for jobName, pods := range podsMap {
		for _, pod := range pods {
			logContent, err := c.FetchPodLog(pod.Name, namespace)
			if err != nil {
				fmt.Printf("Error fetching logs for pod %s: %v\n", pod.Name, err)
				continue
			}
			logsMap[fmt.Sprintf("%s/%s", jobName, pod.Name)] = *logContent
		}
	}

	return logsMap, nil
}

// FetchPodLog is a helper function to get logs for a single Pod.
func (c *CronJobManager) FetchPodLog(podName, namespace string) (*string, error) {
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

// CronJobFullInfo includes information about the CronJob, its child jobs, and pods in each job
type CronJobFullInfo struct {
	CronJobInformation *CronJobInfo
	Jobs               []JobInfo
}

func (c *CronJobManager) GetCronJobAndPods() (*[]CronJobInfo, error) {
	cronJobs, err := c.ListCronJobs()
	if err != nil || cronJobs == nil {
		fmt.Printf("no cronJobs found in GetCronJobAndPods: %s\n", err.Error())
		return nil, err
	}

	for i, cronJob := range *cronJobs {
		jobs, err := c.GetJobsForCronJob(cronJob.Name, cronJob.Namespace)

		if err != nil {
			fmt.Printf("No jobs found for %s/%s", cronJob.Namespace, cronJob.Name)
			return nil, err
		}

		podsList, err := c.GetPods(jobs)
		if err != nil {
			fmt.Printf("no pods found in GetCronJobAndPods: %s\n", err.Error())
			return nil, err
		}

		for j, job := range jobs {
			fmt.Printf("found pods: %+v\n", podsList)
			for jobName, pods := range podsList {
				if jobName == job.Name {
					fmt.Printf("Set pods for job '%s' to '%v'\n", jobName, pods)
					jobs[j].Pods = &pods
				}
			}
		}

		(*cronJobs)[i].Jobs = &jobs
	}

	return cronJobs, nil
}
