package manager

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"k8s.io/klog/v2"

	"github.com/edgeai-neptune/neptune/cmd/neptune-lc/app/options"
	neptunev1 "github.com/edgeai-neptune/neptune/pkg/apis/neptune/v1alpha1"
	"github.com/edgeai-neptune/neptune/pkg/localcontroller/db"
	"github.com/edgeai-neptune/neptune/pkg/localcontroller/trigger"
	"github.com/edgeai-neptune/neptune/pkg/localcontroller/util"
	"github.com/edgeai-neptune/neptune/pkg/localcontroller/wsclient"
)

// IncrementalLearningJob defines config for incremental-learning-job
type IncrementalLearningJob struct {
	neptunev1.IncrementalLearningJob
	JobConfig *JobConfig
	Dataset   *neptunev1.Dataset
	Done      chan struct{}
}

// JobConfig defines config for incremental-learning-job
type JobConfig struct {
	UniqueIdentifier string
	Version          int
	Phase            string
	WorkerStatus     string
	TriggerStatus    string
	TriggerTime      time.Time
	TrainDataURL     string
	EvalDataURL      string
	OutputDir        string
	OutputConfig     *OutputConfig
	DataSamples      *DataSamples
	TrainModel       *TrainModel
	DeployModel      *ModelInfo
	EvalResult       []*ModelInfo
	Lock             sync.Mutex
}

// OutputConfig defines config for job output
type OutputConfig struct {
	SamplesOutput map[string]string `json:"trainData"`
	TrainOutput   string            `json:"trainOutput"`
	EvalOutput    string            `json:"evalOutput"`
}

// DataSamples defines samples information
type DataSamples struct {
	Numbers            int
	TrainSamples       []string
	EvalVersionSamples [][]string
	EvalSamples        []string
}

// TrainModel defines config about training model
type TrainModel struct {
	Model        *ModelInfo        `json:"model"`
	TrainedModel map[string]string `json:"trainedModel"`
	OutputURL    string            `json:"outputUrl"`
}

// IncrementalLearningJob defines incremental-learning-job manager
type IncrementalJobManager struct {
	Client               *wsclient.Client
	WorkerMessageChannel chan WorkerMessage
	DatasetManager       *DatasetManager
	ModelManager         *ModelManager
	IncrementalJobMap    map[string]*IncrementalLearningJob
	IncrementalJobSignal map[string]bool
	VolumeMountPrefix    string
}

const (
	// JobIterationIntervalSeconds is interval time of each iteration of job
	JobIterationIntervalSeconds = 10
	// DatasetHandlerIntervalSeconds is interval time of handling dataset
	DatasetHandlerIntervalSeconds = 10
	// ModelHandlerIntervalSeconds is interval time of handling model
	ModelHandlerIntervalSeconds = 10
	// EvalSamplesCapacity is capacity of eval samples
	EvalSamplesCapacity = 5
	//IncrementalLearningJobKind is kind of incremental-learning-job resource
	IncrementalLearningJobKind = "incrementallearningjob"
)

// NewIncrementalJobManager creates a incremental-learning-job manager
func NewIncrementalJobManager(client *wsclient.Client, datasetManager *DatasetManager,
	modelManager *ModelManager, options *options.LocalControllerOptions) *IncrementalJobManager {
	im := IncrementalJobManager{
		Client:               client,
		WorkerMessageChannel: make(chan WorkerMessage, WorkerMessageChannelCacheSize),
		DatasetManager:       datasetManager,
		ModelManager:         modelManager,
		IncrementalJobMap:    make(map[string]*IncrementalLearningJob),
		IncrementalJobSignal: make(map[string]bool),
		VolumeMountPrefix:    options.VolumeMountPrefix,
	}

	return &im
}

// Start starts incremental-learning-job manager
func (im *IncrementalJobManager) Start() error {
	im.IncrementalJobSignal = make(map[string]bool)

	if err := im.Client.Subscribe(IncrementalLearningJobKind, im.handleMessage); err != nil {
		klog.Errorf("register incremental-learning-job manager to the client failed, error: %v", err)
		return err
	}

	go im.monitorWorker()

	klog.Infof("start incremental-learning-job manager successfully")
	return nil
}

// handleMessage handles the message from GlobalManager
func (im *IncrementalJobManager) handleMessage(message *wsclient.Message) {
	uniqueIdentifier := util.GetUniqueIdentifier(message.Header.Namespace, message.Header.ResourceName, message.Header.ResourceKind)

	switch message.Header.Operation {
	case InsertOperation:
		{
			if err := im.insertJob(uniqueIdentifier, message.Content); err != nil {
				klog.Errorf("insert %s(name=%s) to db failed, error: %v", message.Header.ResourceKind, uniqueIdentifier, err)
			}

			if _, ok := im.IncrementalJobSignal[uniqueIdentifier]; !ok {
				im.IncrementalJobSignal[uniqueIdentifier] = true
				go im.startJob(uniqueIdentifier, message)
			}
		}
	case DeleteOperation:
		{
			if err := im.deleteJob(uniqueIdentifier); err != nil {
				klog.Errorf("delete %s(name=%s) to db failed, error: %v", message.Header.ResourceKind, uniqueIdentifier, err)
			}

			if _, ok := im.IncrementalJobSignal[uniqueIdentifier]; ok {
				im.IncrementalJobSignal[uniqueIdentifier] = false
			}
		}
	}
}

// trainTask starts training task
func (im *IncrementalJobManager) trainTask(job *IncrementalLearningJob, message *wsclient.Message) error {
	jobConfig := job.JobConfig

	if jobConfig.WorkerStatus == WorkerReadyStatus && jobConfig.TriggerStatus == TriggerReadyStatus {
		payload, ok, err := im.triggerTrainTask(job)
		if !ok {
			return nil
		}

		if err != nil {
			klog.Errorf("job(name=%s) complete the %sing phase triggering task failed, error: %v",
				jobConfig.UniqueIdentifier, jobConfig.Phase, err)
			return err
		}

		message.Header.Operation = StatusOperation
		err = im.Client.WriteMessage(payload, message.Header)
		if err != nil {
			return err
		}

		jobConfig.TriggerStatus = TriggerCompletedStatus

		klog.Infof("job(name=%s) complete the %sing phase triggering task successfully",
			jobConfig.UniqueIdentifier, jobConfig.Phase)
	}

	if jobConfig.WorkerStatus == WorkerFailedStatus {
		klog.Warningf("found the %sing phase worker that ran failed, "+
			"back the training phase triggering task", jobConfig.Phase)
		backTask(jobConfig)
	}

	if jobConfig.WorkerStatus == WorkerCompletedStatus {
		klog.Infof("job(name=%s) complete the %s task successfully", jobConfig.UniqueIdentifier, jobConfig.Phase)
		nextTask(jobConfig)
	}

	return nil
}

// evalTask starts eval task
func (im *IncrementalJobManager) evalTask(job *IncrementalLearningJob, message *wsclient.Message) error {
	jobConfig := job.JobConfig

	if jobConfig.WorkerStatus == WorkerReadyStatus && jobConfig.TriggerStatus == TriggerReadyStatus {
		payload, err := im.triggerEvalTask(job)
		if err != nil {
			klog.Errorf("job(name=%s) complete the %sing phase triggering task failed, error: %v",
				jobConfig.UniqueIdentifier, jobConfig.Phase, err)
			return err
		}

		message.Header.Operation = StatusOperation
		err = im.Client.WriteMessage(payload, message.Header)
		if err != nil {
			return err
		}

		jobConfig.TriggerStatus = TriggerCompletedStatus

		klog.Infof("job(name=%s) complete the %sing phase triggering task successfully",
			jobConfig.UniqueIdentifier, jobConfig.Phase)
	}

	if jobConfig.WorkerStatus == WorkerFailedStatus {
		msg := fmt.Sprintf("job(name=%s) found the %sing phase worker that ran failed, "+
			"back the training phase triggering task", jobConfig.UniqueIdentifier, jobConfig.Phase)
		klog.Errorf(msg)
		return fmt.Errorf(msg)
	}

	if jobConfig.WorkerStatus == WorkerCompletedStatus {
		klog.Infof("job(name=%s) complete the %s task successfully", jobConfig.UniqueIdentifier, jobConfig.Phase)
		nextTask(jobConfig)
	}

	return nil
}

// deployTask starts deploy task
func (im *IncrementalJobManager) deployTask(job *IncrementalLearningJob, message *wsclient.Message) error {
	jobConfig := job.JobConfig

	if jobConfig.WorkerStatus == WorkerReadyStatus && jobConfig.TriggerStatus == TriggerReadyStatus {
		neededDeploy, err := im.triggerDeployTask(job)
		status := UpstreamMessage{}
		if err == nil && neededDeploy {
			status.Phase = DeployPhase

			deployModel, err := im.deployModel(job)
			if err != nil {
				klog.Errorf("failed to deploy model for job(name=%s): %v", jobConfig.UniqueIdentifier, err)
			} else {
				klog.Infof("deployed model for job(name=%s) successfully", jobConfig.UniqueIdentifier)
			}
			if err != nil || deployModel == nil {
				status.Status = WorkerFailedStatus
			} else {
				status.Status = WorkerReadyStatus
				status.Input.Models = []ModelInfo{
					*deployModel,
				}
			}
		} else {
			// TODO
			status.Phase = TrainPhase
			status.Status = WorkerWaitingStatus
		}

		message.Header.Operation = StatusOperation
		if err = im.Client.WriteMessage(status, message.Header); err != nil {
			return err
		}

		jobConfig.TriggerStatus = TriggerCompletedStatus

		klog.Infof("job(name=%s) complete the %sing phase triggering task successfully",
			jobConfig.UniqueIdentifier, jobConfig.Phase)
	}

	nextTask(jobConfig)

	klog.Infof("job(name=%s) complete the %s task successfully", jobConfig.UniqueIdentifier, jobConfig.Phase)

	return nil
}

// startJob starts a job
func (im *IncrementalJobManager) startJob(name string, message *wsclient.Message) {
	var err error
	job := im.IncrementalJobMap[name]

	job.JobConfig = new(JobConfig)
	jobConfig := job.JobConfig
	jobConfig.UniqueIdentifier = name

	err = im.initJob(job)
	if err != nil {
		klog.Errorf("init job (name=%s) failed", jobConfig.UniqueIdentifier)
		return
	}

	klog.Infof("incremental job(name=%s) is started", name)
	defer klog.Infof("incremental learning job(name=%s) is stopped", name)
	err = im.handleData(job)
	if err == nil {
		err = im.handleModel(job)
	}
	if err != nil {
		klog.Errorf("failed to handle incremental learning job: %+v", err)
		return
	}

	for {
		select {
		case <-job.Done:
			return

		case <-time.After(JobIterationIntervalSeconds * time.Second):
		}

		switch jobConfig.Phase {
		case TrainPhase:
			err = im.trainTask(job, message)
		case EvalPhase:
			err = im.evalTask(job, message)
		case DeployPhase:
			err = im.deployTask(job, message)
		default:
			klog.Errorf("not vaild phase: %s", jobConfig.Phase)
			continue
		}

		if err != nil {
			klog.Errorf("job(name=%s) complete the %s task failed, error: %v",
				jobConfig.UniqueIdentifier, jobConfig.Phase, err)
			continue
		}
	}
}

// insertJob inserts incremental-learning-job config to db
func (im *IncrementalJobManager) insertJob(name string, payload []byte) error {
	job, ok := im.IncrementalJobMap[name]
	if !ok {
		job = &IncrementalLearningJob{}
		job.Done = make(chan struct{})
		im.IncrementalJobMap[name] = job
	}

	if err := json.Unmarshal(payload, &job); err != nil {
		return err
	}

	if err := db.SaveResource(name, job.TypeMeta, job.ObjectMeta, job.Spec); err != nil {
		return err
	}

	return nil
}

// deleteJob deletes incremental-learning-job config in db
func (im *IncrementalJobManager) deleteJob(name string) error {
	if err := db.DeleteResource(name); err != nil {
		return err
	}

	if job, ok := im.IncrementalJobMap[name]; ok && job.Done != nil {
		close(job.Done)
	}

	delete(im.IncrementalJobMap, name)

	delete(im.IncrementalJobSignal, name)

	return nil
}

// initJob inits the job object
func (im *IncrementalJobManager) initJob(job *IncrementalLearningJob) error {
	jobConfig := job.JobConfig
	jobConfig.OutputDir = util.AddPrefixPath(im.VolumeMountPrefix, job.Spec.OutputDir)
	jobConfig.TrainModel = new(TrainModel)
	jobConfig.TrainModel.OutputURL = jobConfig.OutputDir
	jobConfig.DeployModel = new(ModelInfo)
	jobConfig.Lock = sync.Mutex{}

	jobConfig.Version = 0
	jobConfig.Phase = TrainPhase
	jobConfig.WorkerStatus = WorkerReadyStatus
	jobConfig.TriggerStatus = TriggerReadyStatus

	if err := createOutputDir(jobConfig); err != nil {
		return err
	}

	return nil
}

// triggerTrainTask triggers the train task
func (im *IncrementalJobManager) triggerTrainTask(job *IncrementalLearningJob) (interface{}, bool, error) {
	var err error
	jobConfig := job.JobConfig
	tt := job.Spec.TrainSpec.Trigger

	// convert tt.Condition to map
	triggerMap := make(map[string]interface{})
	c, err := json.Marshal(tt)
	if err != nil {
		return nil, false, err
	}

	err = json.Unmarshal(c, &triggerMap)
	if err != nil {
		return nil, false, err
	}
	const numOfSamples = "num_of_samples"
	samples := map[string]interface{}{
		numOfSamples: len(jobConfig.DataSamples.TrainSamples),
	}

	trainTrigger, err := trigger.NewTrigger(triggerMap)
	if err != nil {
		klog.Errorf("train phase: get trigger object failed, error: %v", err)
		return nil, false, err
	}
	isTrigger := trainTrigger.Trigger(samples)

	if !isTrigger {
		return nil, false, nil
	}

	jobConfig.Version++

	jobConfig.TrainDataURL, err = im.writeSamples(jobConfig.DataSamples.TrainSamples,
		jobConfig.OutputConfig.SamplesOutput["train"], jobConfig.Version, job.Dataset.Spec.Format)
	if err != nil {
		klog.Errorf("train phase: write samples to the file(%s) is failed, error: %v", jobConfig.TrainDataURL, err)
		return nil, false, err
	}

	format := jobConfig.TrainModel.Model.Format
	m := ModelInfo{
		Format: format,
		URL:    jobConfig.TrainModel.TrainedModel[format],
	}
	input := WorkerInput{
		Models:  []ModelInfo{m},
		DataURL: util.TrimPrefixPath(im.VolumeMountPrefix, jobConfig.TrainDataURL),
		OutputDir: util.TrimPrefixPath(im.VolumeMountPrefix,
			path.Join(jobConfig.OutputConfig.TrainOutput, strconv.Itoa(jobConfig.Version))),
	}
	msg := UpstreamMessage{
		Phase:  TrainPhase,
		Status: WorkerReadyStatus,
		Input:  &input,
	}
	jobConfig.TriggerTime = time.Now()
	return &msg, true, nil
}

// triggerEvalTask triggers the eval task
func (im *IncrementalJobManager) triggerEvalTask(job *IncrementalLearningJob) (*UpstreamMessage, error) {
	jobConfig := job.JobConfig
	var err error

	(*jobConfig).EvalDataURL, err = im.writeSamples(jobConfig.DataSamples.EvalSamples, jobConfig.OutputConfig.SamplesOutput["eval"],
		jobConfig.Version, job.Dataset.Spec.Format)
	if err != nil {
		klog.Errorf("job(name=%s) eval phase: write samples to the file(%s) is failed, error: %v",
			jobConfig.UniqueIdentifier, jobConfig.EvalDataURL, err)
		return nil, err
	}

	var models []ModelInfo
	models = append(models, ModelInfo{
		Format: "pb",
		URL:    jobConfig.TrainModel.TrainedModel["pb"],
	})

	models = append(models, ModelInfo{
		Format: jobConfig.DeployModel.Format,
		URL:    jobConfig.DeployModel.URL,
	})

	input := WorkerInput{
		Models:  models,
		DataURL: util.TrimPrefixPath(im.VolumeMountPrefix, jobConfig.EvalDataURL),
		OutputDir: util.TrimPrefixPath(im.VolumeMountPrefix,
			path.Join(jobConfig.OutputConfig.EvalOutput, strconv.Itoa(jobConfig.Version))),
	}
	msg := &UpstreamMessage{
		Phase:  EvalPhase,
		Status: WorkerReadyStatus,
		Input:  &input,
	}

	return msg, nil
}

// triggerDeployTask triggers the deploy task
func (im *IncrementalJobManager) triggerDeployTask(job *IncrementalLearningJob) (bool, error) {
	jobConfig := job.JobConfig

	if len(jobConfig.EvalResult) != 2 {
		return false, fmt.Errorf("expected 2 evaluation results e, actual: %d", len(jobConfig.EvalResult))
	}

	newMetrics, oldMetrics := jobConfig.EvalResult[0].Metrics, jobConfig.EvalResult[1].Metrics
	metricDelta := make(map[string]interface{})

	for metric := range newMetrics {
		// keep the full metrics
		metricDelta[metric] = newMetrics[metric]
		var l []float64
		for i := range newMetrics[metric] {
			l = append(l, newMetrics[metric][i]-oldMetrics[metric][i])
		}
		metricDelta[metric+"_delta"] = l
	}
	tt := job.Spec.DeploySpec.Trigger

	// convert tt.Condition to map
	triggerMap := make(map[string]interface{})
	c, err := json.Marshal(tt)
	if err != nil {
		return false, err
	}

	err = json.Unmarshal(c, &triggerMap)
	if err != nil {
		return false, err
	}

	deployTrigger, err := trigger.NewTrigger(triggerMap)
	if err != nil {
		klog.Errorf("job(name=%s) deploy phase: get trigger object failed, error: %v", jobConfig.UniqueIdentifier, err)
		return false, err
	}

	return deployTrigger.Trigger(metricDelta), nil
}

// deployModel deploys model
func (im *IncrementalJobManager) deployModel(job *IncrementalLearningJob) (*ModelInfo, error) {
	jobConfig := job.JobConfig

	var models []ModelInfo
	for i := 0; i < len(jobConfig.EvalResult); i++ {
		models = append(models, ModelInfo{
			Format: jobConfig.EvalResult[i].Format,
			URL:    jobConfig.EvalResult[i].URL,
		})
	}

	var err error

	trainedModelFormat := models[0].Format
	deployModelFormat := models[1].Format
	if trainedModelFormat != deployModelFormat {
		msg := fmt.Sprintf("the trained model format(format=%s) is inconsistent with deploy model(format=%s)",
			deployModelFormat, deployModelFormat)
		klog.Errorf(msg)

		return nil, fmt.Errorf(msg)
	}

	trainedModel := util.AddPrefixPath(im.VolumeMountPrefix, models[0].URL)
	deployModel := util.AddPrefixPath(im.VolumeMountPrefix, models[1].URL)
	if _, err = util.CopyFile(trainedModel, deployModel); err != nil {
		klog.Errorf("copy the trained model file(url=%s) to the deployment model file(url=%s) failed",
			trainedModel, deployModel)

		return nil, err
	}

	jobConfig.DeployModel.Format = models[1].Format
	jobConfig.DeployModel.URL = models[1].URL

	klog.Infof("job(name=%s) deploys model(url=%s) successfully", jobConfig.UniqueIdentifier, trainedModel)

	return &models[0], nil
}

// createOutputDir creates the job output dir
func createOutputDir(jobConfig *JobConfig) error {
	if err := util.CreateFolder(jobConfig.OutputDir); err != nil {
		klog.Errorf("job(name=%s) create fold %s failed", jobConfig.UniqueIdentifier, jobConfig.OutputDir)
		return err
	}

	dirNames := []string{"data/train", "data/eval", "train", "eval"}

	for _, v := range dirNames {
		dir := path.Join(jobConfig.OutputDir, v)
		if err := util.CreateFolder(dir); err != nil {
			klog.Errorf("job(name=%s) create fold %s failed", jobConfig.UniqueIdentifier, dir)
			return err
		}
	}

	outputConfig := OutputConfig{
		SamplesOutput: map[string]string{
			"train": path.Join(jobConfig.OutputDir, dirNames[0]),
			"eval":  path.Join(jobConfig.OutputDir, dirNames[1]),
		},
		TrainOutput: path.Join(jobConfig.OutputDir, dirNames[2]),
		EvalOutput:  path.Join(jobConfig.OutputDir, dirNames[3]),
	}
	jobConfig.OutputConfig = &outputConfig

	return nil
}

// handleModel updates model information for training and deploying
func (im *IncrementalJobManager) handleModel(job *IncrementalLearningJob) error {
	jobConfig := job.JobConfig
	jobConfig.TrainModel.Model = new(ModelInfo)
	jobConfig.TrainModel.TrainedModel = map[string]string{}
	jobConfig.DeployModel = new(ModelInfo)

	var modelName string
	modelName = util.GetUniqueIdentifier(
		job.Namespace,
		job.Spec.InitialModel.Name, ModelResourceKind)
	trainModel, ok := im.ModelManager.GetModel(modelName)
	// wait 30 seconds for model synced to lc
	for i := 0; i < 300; i++ {
		if ok {
			break
		}
		<-time.After(time.Millisecond * 100)
		trainModel, ok = im.ModelManager.GetModel(modelName)
	}
	if !ok {
		return fmt.Errorf("not exists model(name=%s)", modelName)
	}

	format := trainModel.Spec.Format
	url := trainModel.Spec.ModelURL
	jobConfig.TrainModel.Model.Format = format
	jobConfig.TrainModel.Model.URL = url
	if _, ok := jobConfig.TrainModel.TrainedModel[format]; !ok {
		jobConfig.TrainModel.TrainedModel[format] = url
	}

	modelName = util.GetUniqueIdentifier(
		job.Namespace, job.Spec.DeploySpec.Model.Name,
		ModelResourceKind)
	evalModel, ok := im.ModelManager.GetModel(modelName)
	if !ok {
		return fmt.Errorf("not exists model(name=%s)", modelName)
	}

	jobConfig.DeployModel.Format = evalModel.Spec.Format
	jobConfig.DeployModel.URL = evalModel.Spec.ModelURL
	return nil
}

// handleData updates samples information
func (im *IncrementalJobManager) handleData(job *IncrementalLearningJob) error {
	jobConfig := job.JobConfig
	jobConfig.DataSamples = &DataSamples{
		Numbers:            0,
		TrainSamples:       make([]string, 0),
		EvalVersionSamples: make([][]string, 0),
		EvalSamples:        make([]string, 0),
	}

	datasetName := util.GetUniqueIdentifier(job.Namespace, job.Spec.Dataset.Name, DatasetResourceKind)

	// wait 30 seconds for dataset synced to lc
	for i := 0; i < 300; i++ {
		dataset, ok := im.DatasetManager.GetDataset(datasetName)
		if ok && dataset != nil {
			break
		}
		<-time.After(time.Millisecond * 100)
	}

	dataset, ok := im.DatasetManager.GetDataset(datasetName)

	if !ok || dataset == nil {
		return fmt.Errorf("not exists dataset(name=%s)", datasetName)
	}

	job.Dataset = dataset.Dataset

	go func() {
		tick := time.NewTicker(DatasetHandlerIntervalSeconds * time.Second)
		for {
			if dataset.DataSource != nil && len(dataset.DataSource.TrainSamples) > jobConfig.DataSamples.Numbers {
				samples := dataset.DataSource.TrainSamples
				trainNum := int(job.Spec.Dataset.TrainProb * float64(len(samples)-jobConfig.DataSamples.Numbers))

				jobConfig.Lock.Lock()
				jobConfig.DataSamples.TrainSamples = append(jobConfig.DataSamples.TrainSamples,
					samples[(jobConfig.DataSamples.Numbers+1):(jobConfig.DataSamples.Numbers+trainNum+1)]...)
				klog.Infof("job(name=%s) current train samples nums is %d",
					jobConfig.UniqueIdentifier, len(jobConfig.DataSamples.TrainSamples))

				jobConfig.DataSamples.EvalVersionSamples = append(jobConfig.DataSamples.EvalVersionSamples,
					samples[(jobConfig.DataSamples.Numbers+trainNum+1):])
				jobConfig.Lock.Unlock()

				for _, v := range jobConfig.DataSamples.EvalVersionSamples {
					jobConfig.DataSamples.EvalSamples = append(jobConfig.DataSamples.EvalSamples, v...)
				}
				klog.Infof("job(name=%s) current eval samples nums is %d",
					jobConfig.UniqueIdentifier, len(jobConfig.DataSamples.EvalSamples))

				jobConfig.DataSamples.Numbers = len(samples)
			} else {
				klog.Warningf("job(name=%s) didn't get new data from dataset(name=%s)",
					jobConfig.UniqueIdentifier, job.Spec.Dataset.Name)
			}
			select {
			case <-job.Done:
				return
			case <-tick.C:
			}
		}
	}()
	return nil
}

// createFile create a file
func createFile(dir string, format string) (string, bool) {
	switch format {
	case "txt":
		return path.Join(dir, "data.txt"), true
	}
	return "", false
}

// writeSamples writes samples information to a file
func (im *IncrementalJobManager) writeSamples(samples []string, dir string, version int, format string) (string, error) {
	subDir := path.Join(dir, strconv.Itoa(version))
	if err := util.CreateFolder(subDir); err != nil {
		return "", err
	}

	fileURL, isFile := createFile(subDir, format)
	if isFile {
		if err := im.writeByLine(samples, fileURL); err != nil {
			return "", err
		}
	} else {
		return "", fmt.Errorf("create a %s format file in %s failed", format, subDir)
	}

	return fileURL, nil
}

// writeByLine writes file by line
func (im *IncrementalJobManager) writeByLine(samples []string, fileURL string) error {
	file, err := os.Create(fileURL)
	if err != nil {
		klog.Errorf("create file(%s) failed", fileURL)
		return err
	}

	w := bufio.NewWriter(file)
	for _, line := range samples {
		_, _ = fmt.Fprintln(w, line)
	}
	if err := w.Flush(); err != nil {
		klog.Errorf("write file(%s) failed", fileURL)
		return err
	}

	if err := file.Close(); err != nil {
		klog.Errorf("close file failed, error: %v", err)
		return err
	}

	return nil
}

// monitorWorker monitors message from worker
func (im *IncrementalJobManager) monitorWorker() {
	for {
		workerMessageChannel := im.WorkerMessageChannel
		workerMessage, ok := <-workerMessageChannel
		if !ok {
			break
		}
		klog.V(4).Infof("handling worker message %+v", workerMessage)

		name := util.GetUniqueIdentifier(workerMessage.Namespace, workerMessage.OwnerName, workerMessage.OwnerKind)
		header := wsclient.MessageHeader{
			Namespace:    workerMessage.Namespace,
			ResourceKind: workerMessage.OwnerKind,
			ResourceName: workerMessage.OwnerName,
			Operation:    StatusOperation,
		}

		if err := im.Client.WriteMessage(workerMessage, header); err != nil {
			klog.Errorf("job(name=%s) uploads worker(name=%s) message failed, error: %v",
				name, workerMessage.Name, err)
		}

		job, ok := im.IncrementalJobMap[name]
		if !ok {
			continue
		}

		im.handleWorkerMessage(job, workerMessage)
	}
}

// handleWorkerMessage handles message from worker
func (im *IncrementalJobManager) handleWorkerMessage(job *IncrementalLearningJob, workerMessage WorkerMessage) {
	job.JobConfig.TrainModel.TrainedModel = make(map[string]string)

	jobPhase := job.JobConfig.Phase
	workerKind := workerMessage.Kind
	if jobPhase != workerKind {
		klog.Warningf("job(name=%s) %s phase get worker(kind=%s)", job.JobConfig.UniqueIdentifier,
			jobPhase, workerKind)
		return
	}

	var models []*ModelInfo
	for _, result := range workerMessage.Results {
		metrics := map[string][]float64{}
		if m, ok := result["metrics"]; ok {
			bytes, err := json.Marshal(m)
			if err != nil {
				return
			}

			err = json.Unmarshal(bytes, &metrics)
			if err != nil {
				klog.Warningf("failed to unmarshal the worker(name=%s) metrics %v, err: %v",
					workerMessage.Name,
					m,
					err)
			}
		}

		model := ModelInfo{
			result["format"].(string),
			result["url"].(string),
			metrics}
		models = append(models, &model)
	}

	job.JobConfig.WorkerStatus = workerMessage.Status

	if job.JobConfig.WorkerStatus == WorkerCompletedStatus {
		switch job.JobConfig.Phase {
		case TrainPhase:
			{
				for i := 0; i < len(models); i++ {
					format := models[i].Format
					if format != "" {
						job.JobConfig.TrainModel.TrainedModel[format] = models[i].URL
					}
				}
			}

		case EvalPhase:
			job.JobConfig.EvalResult = models
		}
	}
}

// forwardSamples deletes the samples information in the memory
func forwardSamples(jobConfig *JobConfig) {
	switch jobConfig.Phase {
	case TrainPhase:
		{
			jobConfig.Lock.Lock()
			jobConfig.DataSamples.TrainSamples = jobConfig.DataSamples.TrainSamples[:0]
			jobConfig.Lock.Unlock()
		}
	case EvalPhase:
		{
			if len(jobConfig.DataSamples.EvalVersionSamples) > EvalSamplesCapacity {
				jobConfig.DataSamples.EvalVersionSamples = jobConfig.DataSamples.EvalVersionSamples[1:]
			}
		}
	}
}

// backTask backs train task status
func backTask(jobConfig *JobConfig) {
	jobConfig.Phase = TrainPhase
	initTaskStatus(jobConfig)
}

// initTaskStatus inits task status
func initTaskStatus(jobConfig *JobConfig) {
	jobConfig.WorkerStatus = WorkerReadyStatus
	jobConfig.TriggerStatus = TriggerReadyStatus
}

// nextTask converts next task status
func nextTask(jobConfig *JobConfig) {
	switch jobConfig.Phase {
	case TrainPhase:
		{
			forwardSamples(jobConfig)
			initTaskStatus(jobConfig)
			jobConfig.Phase = EvalPhase
		}

	case EvalPhase:
		{
			forwardSamples(jobConfig)
			initTaskStatus(jobConfig)
			jobConfig.Phase = DeployPhase
		}
	case DeployPhase:
		{
			backTask(jobConfig)
		}
	}
}

// AddWorkerMessageToChannel adds worker messages to the channel
func (im *IncrementalJobManager) AddWorkerMessageToChannel(message WorkerMessage) {
	im.WorkerMessageChannel <- message
}

// GetKind gets kind of the manager
func (im *IncrementalJobManager) GetKind() string {
	return IncrementalLearningJobKind
}