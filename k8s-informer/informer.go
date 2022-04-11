package kinformer

import (
	"context"
	"log"
	"time"

	"github.com/zbitech/common/pkg/logger"
	"github.com/zbitech/common/pkg/model/entity"
	"github.com/zbitech/common/pkg/model/ztypes"
	"github.com/zbitech/common/pkg/vars"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type InformerWorkQueue struct {
	queue workqueue.RateLimitingInterface
	//	requeueLimit int
	//	requeueDelay time.Duration
}

func NewInformerWorkQueue() *InformerWorkQueue {
	return &InformerWorkQueue{
		queue: workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		//		requeueLimit: vars.INFORMER_REQUEUE_LIMIT,
		//		requeueDelay: time.Duration(vars.INFORMER_REQUEUE_DELAY) * vars.INFORMER_REQUEUE_DELAY_DURATION,
	}
}

func (inf *InformerWorkQueue) QueueItem(qe QueueElement) {
	if qe.RequeueCount < vars.AppConfig.Kubernetes.Informer.RequeueLimit {

		go func() {
			requeueDelay := time.Duration(vars.AppConfig.Kubernetes.Informer.RequeueDelay) * time.Second
			time.Sleep(requeueDelay)
			log.Printf("Adding %s back to the queue", qe.Key)
			if qe.RequeueCount == 0 {
				inf.queue.Add(qe)
			} else {
				inf.queue.AddRateLimited(qe)
			}
		}()

	} else {
		//TODO generate system alert of failed deployment
	}
}

type KlientInformerController struct {
	typedFactory   informers.SharedInformerFactory
	dynamicFactory dynamicinformer.DynamicSharedInformerFactory
	stopper        chan struct{}
	informers      map[ztypes.ResourceObjectType]ZKlientInformer
	workQueue      *InformerWorkQueue
	//	queue          workqueue.RateLimitingInterface
	//	requeueLimit   int
	//	requeueDelay   int
	//	delayDuration  time.Duration
}

type ResourceAction string

const (
	AddResource    ResourceAction = "ADD"
	UpdateResource ResourceAction = "UPDATE"
	DeleteResource ResourceAction = "DELETE"
)

type QueueElement struct {
	Action       ResourceAction
	Type         ztypes.ResourceObjectType
	Key          string
	Status       *ObjectStatus
	RequeueCount int
}

func NewKlientInformerController(clientSet kubernetes.Interface, dynClient dynamic.Interface) *KlientInformerController {

	return &KlientInformerController{
		typedFactory:   informers.NewSharedInformerFactoryWithOptions(clientSet, time.Second*30),   // informers.NewFilteredSharedInformerFactory(clientset, time.Second*30, metav1.NamespaceAll, nil),
		dynamicFactory: dynamicinformer.NewDynamicSharedInformerFactory(dynClient, time.Second*30), // dynamicinformer.NewFilteredDynamicSharedInformerFactory(dynClient, time.Second*30, metav1.NamespaceAll, nil),
		stopper:        make(chan struct{}),
		informers:      make(map[ztypes.ResourceObjectType]ZKlientInformer, 0),
		workQueue:      NewInformerWorkQueue(),
	}
}

func (k *KlientInformerController) AddInformer(rType ztypes.ResourceObjectType) {

	var inf cache.SharedIndexInformer

	switch rType {
	case ztypes.DEPLOYMENT_RESOURCE:
		inf = k.typedFactory.Apps().V1().Deployments().Informer()
	case ztypes.POD_RESOURCE:
		inf = k.typedFactory.Core().V1().Pods().Informer()
	case ztypes.PERSISTENT_VOLUME_RESOURCE:
		inf = k.typedFactory.Core().V1().PersistentVolumes().Informer()
	case ztypes.PERSISTENT_VOLUME_CLAIM_RESOURCE:
		inf = k.typedFactory.Core().V1().PersistentVolumeClaims().Informer()
	case ztypes.VOLUME_SNAPHOT_RESOURCE:
		var gvr schema.GroupVersionResource
		inf = k.dynamicFactory.ForResource(gvr).Informer()
	}

	kinf := ZKlientInformer{informer: inf, objectType: rType}
	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc:    k.AddEvent,
		UpdateFunc: k.UpdateEvent,
		DeleteFunc: k.DeleteEvent,
	}

	kinf.informer.AddEventHandler(handlers)
}

func (k *KlientInformerController) processEvent(action ResourceAction, obj interface{}, requeueCount int) {
	kObj, ok := obj.(runtime.Object)
	if ok {
		var result *ObjectStatus
		rType := ztypes.ResourceObjectType(kObj.GetObjectKind().GroupVersionKind().Kind)
		switch rType {
		case ztypes.DEPLOYMENT_RESOURCE:
			result = DeploymentEvent(kObj.(*appsv1.Deployment))

		case ztypes.POD_RESOURCE:
			result = PodEvent(kObj.(*corev1.Pod))

		case ztypes.PERSISTENT_VOLUME_RESOURCE:
			result = PersistentVolumeEvent(kObj.(*corev1.PersistentVolume))

		case ztypes.PERSISTENT_VOLUME_CLAIM_RESOURCE:
			result = PersistentVolumeClaimEvent(kObj.(*corev1.PersistentVolumeClaim))

			// 	case ztypes.VOLUME_SNAPHOT_RESOURCE:
		}

		if !result.Ignore {
			if result.State.IsActive() {
				//TODO update resource in repository
				k.SaveKubernetesResource(context.Background(), action, rType, result)
			} else {
				//TODO Put back on queue
				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err != nil {
					logger.Errorf(context.Background(), "Unable to add item %s to queue - %s", key, err)
				} else {
					logger.Errorf(context.Background(), "Adding item %s to queue", key)
					// k.queue.Add(QueueElement{Action: action, Key: key, Type: rType, Status: result, RequeueCount: requeueCount})
					qe := QueueElement{Action: action, Key: key, Type: rType, Status: result, RequeueCount: requeueCount}
					k.workQueue.QueueItem(qe)
				}
			}
		}
	}
}

func (k *KlientInformerController) AddEvent(obj interface{}) {
	k.processEvent(AddResource, obj, 0)
}

func (k *KlientInformerController) UpdateEvent(obj_1, obj_2 interface{}) {
	if isObjectModified(obj_1, obj_2) {
		k.processEvent(UpdateResource, obj_2, 0)
	}
}

func (k *KlientInformerController) DeleteEvent(obj interface{}) {
	k.processEvent(DeleteResource, obj, 0)
}

func (k *KlientInformerController) Start() {
	k.typedFactory.Start(k.stopper)
	k.typedFactory.WaitForCacheSync(k.stopper)

	k.dynamicFactory.Start(k.stopper)
	k.dynamicFactory.WaitForCacheSync(k.stopper)
}

func (k *KlientInformerController) GetIndexer(rType ztypes.ResourceObjectType) cache.Indexer {
	inf, ok := k.informers[rType]

	if ok {
		return inf.GetIndexer()
	}

	return nil
}

func (k *KlientInformerController) Run() {

	log.Print("Running KlientInformer")

	defer k.workQueue.queue.ShutDown()
	defer close(k.stopper)

	k.Start()

	log.Print("Starting runWorker ...")
	go wait.Until(k.runWorker, time.Second, k.stopper)
	log.Print("Waiting for informers to complete")

	select {}
}

func (k *KlientInformerController) runWorker() {
	for k.processNextItem(context.Background()) {

	}

}

func (k *KlientInformerController) processNextItem(ctx context.Context) bool {

	//	log.Printf("Processing next item. Queue size - %d", k.queue.Len())

	item, quit := k.workQueue.queue.Get()
	if quit {
		return false
	}

	defer k.workQueue.queue.Done(item)

	qItem := item.(QueueElement)
	requeueCount := k.workQueue.queue.NumRequeues(qItem)

	log.Printf("Action: %s, Key: %s, Kind: %s", qItem.Action, qItem.Key, qItem.Type)

	indexer := k.GetIndexer(qItem.Type)
	if indexer != nil {
		obj, exists, err := indexer.GetByKey(qItem.Key)

		if err != nil {
			logger.Errorf(ctx, "Fetching object with key %s from store failed with %s", qItem.Key, err)

		} else if !exists {
			logger.Errorf(ctx, "Object with key %s no longer exists", qItem.Key)
			status := ObjectStatus{
				Type:      qItem.Type,
				GVR:       qItem.Status.GVR,
				Project:   qItem.Status.Project,
				Instance:  qItem.Status.Instance,
				Version:   qItem.Status.Version,
				Id:        qItem.Status.Id,
				Name:      qItem.Status.Name,
				Namespace: qItem.Status.Namespace,
				Owner:     qItem.Status.Owner,
				Network:   qItem.Status.Network,
				State:     &entity.ResourceState{Deleted: true},
			}
			k.SaveKubernetesResource(ctx, qItem.Action, qItem.Type, &status)
		} else {
			k.processEvent(qItem.Action, obj, requeueCount+1)
		}
	}

	return false
}

func (k *KlientInformerController) SaveKubernetesResource(ctx context.Context, action ResourceAction, oType ztypes.ResourceObjectType, status *ObjectStatus) {

	if len(status.Project) > 0 {

		resource := entity.KubernetesResource{
			Id:        status.Id,
			Name:      status.Name,
			Namespace: status.Namespace,
			Type:      string(oType),
			GVR:       status.GVR,
			State:     status.State,
			Timestamp: time.Now(),
		}

		proj_repo := vars.RepositoryFactory.GetProjectRepository()

		if len(status.Instance) == 0 {
			//			proj_resource := entity.KubernetesProjectResource{KubernetesResource: resource, Project: status.Project}
			proj_repo.SaveProjectResource(ctx, status.Project, &resource)
		} else {
			//			inst_resource := entity.KubernetesInstanceResource{KubernetesResource: resource, Project: status.Project, Instance: status.Instance}
			proj_repo.SaveInstanceResource(ctx, status.Project, status.Instance, &resource)
		}
	}

}

type ZKlientInformer struct {
	informer   cache.SharedIndexInformer
	objectType ztypes.ResourceObjectType
}

// func NewTypedKlientInformer(c cache.SharedIndexInformer, objectType ztypes.ResourceObjectType) ZKlientInformer {
// 	k := ZKlientInformer{
// 		informer:   c,
// 		objectType: objectType,
// 	}

// 	handlers := cache.ResourceEventHandlerFuncs{
// 		AddFunc:    k.Add,
// 		UpdateFunc: k.Update,
// 		DeleteFunc: k.Delete,
// 	}

// 	k.informer.AddEventHandler(handlers)
// 	return k
// }

func (k *ZKlientInformer) GetIndexer() cache.Indexer {
	return k.informer.GetIndexer()
}

// func (k *ZKlientInformer) Add(obj interface{}) {

// 	k_obj, ok := obj.(runtime.Object)
// 	if ok {
// 		var result *ObjectStatus
// 		kind := ztypes.ResourceObjectType(k_obj.GetObjectKind().GroupVersionKind().Kind)
// 		switch kind {
// 		case ztypes.DEPLOYMENT_RESOURCE:
// 			result = DeploymentEvent(k_obj.(*appsv1.Deployment))

// 		case ztypes.POD_RESOURCE:
// 			result = PodEvent(k_obj.(*corev1.Pod))

// 		case ztypes.PERSISTENT_VOLUME_RESOURCE:
// 			result = PersistentVolumeEvent(k_obj.(*corev1.PersistentVolume))

// 		case ztypes.PERSISTENT_VOLUME_CLAIM_RESOURCE:
// 			result = PersistentVolumeClaimEvent(k_obj.(*corev1.PersistentVolumeClaim))

// 			// 	case ztypes.VOLUME_SNAPHOT_RESOURCE:
// 		}

// 		if result.State.IsActive() {

// 		} else {
// 			//TODO Put back on queue
// 			key, err := cache.MetaNamespaceKeyFunc(obj)
// 			if err == nil {
// 				log.Printf("NamespaceKeyFunc - %s", key)
// 			}
// 		}
// 	}
// }

// func (k *ZKlientInformer) Update(obj_1, obj_2 interface{}) {
// 	if isObjectModified(obj_1, obj_2) {

// 	}
// }

// func (k *ZKlientInformer) Delete(obj interface{}) {
// 	k_obj, ok := obj.(runtime.Object)

// 	if ok {
// 		kind := ztypes.ResourceObjectType(k_obj.GetObjectKind().GroupVersionKind().Kind)
// 	}
// }

// func NewKlientInformer(clientset kubernetes.Interface, requeueLimit, requeueDelay int,
// 	delayDuration time.Duration, repository interfaces.ProjectRepositoryIF) *KlientInformer {

// 	log.Print("Creating KlientInformer")
// 	factory := informers.NewSharedInformerFactory(clientset, time.Second*30)
// 	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

// 	return &KlientInformer{
// 		factory:       factory,
// 		queue:         queue,
// 		processors:    map[ztypes.ResourceObjectType]ZBIProcessorInterface{},
// 		stopper:       make(chan struct{}),
// 		repository:    repository,
// 		requeueLimit:  requeueLimit,
// 		requeueDelay:  requeueDelay,
// 		delayDuration: delayDuration,
// 	}
// }

// func (k *KlientInformer) CloseChannel() {
// 	close(k.stopper)
// }

// func (k *KlientInformer) queueItem(action ZBIResourceActionType, obj interface{}, objType ztypes.ResourceObjectType) {

// 	key, err := cache.MetaNamespaceKeyFunc(obj)
// 	if err == nil {
// 		log.Printf("NamespaceKeyFunc - %s", key)
// 		k.queue.Add(QueueItem{Action: action, Key: key, Type: objType})
// 	}

// }

// func (k *KlientInformer) RegisterInformers() {
// 	//	k.registerProcessor(newZBINamespaceProcessor(k.factory))
// 	//	k.registerProcessor(newZBIConfigMapProcessor(k.factory))
// 	//	k.registerProcessor(newZBISecretProcessor(k.factory))
// 	//	k.registerProcessor(newPersistentVolumeProcessor(k.factory))
// 	//	k.registerProcessor(newPersistentVolumeClaimProcessor(k.factory))
// 	//	k.registerProcessor(newZBIDeploymentInformer(k.factory))
// 	//	k.registerProcessor(newZBIPodInformer(k.factory))
// 	//	k.registerProcessor(newZBIServiceProcessor(k.factory))
// }

// func (k *KlientInformer) registerProcessor(i ZBIProcessorInterface) {
// 	log.Printf("Creating %s Processor", i.GetInformerType())

// 	i.GetInformer().AddEventHandler(cache.ResourceEventHandlerFuncs{
// 		AddFunc: func(obj interface{}) {
// 			status := i.AddEventHandler(obj)
// 			if status.ZBI {
// 				log.Printf("Received AddEvent for %s. Status: %s", i.GetInformerType(), fn.MarshalObject(status))

// 				if status.Ready {
// 					k.updateStatus("ACTIVE", i.GetInformerType(), status)

// 					// resource := entity.KubernetesResource{
// 					// 	ProjectId:  status.ProjectId,
// 					// 	InstanceId: status.Id,
// 					// 	Name:       status.Name,
// 					// 	Type:       string(i.GetInformerType()),
// 					// 	Status:     status.Status,
// 					// 	Timestamp:  time.Now(),
// 					// }

// 					// if status.Level == entity.PROJECT_LEVEL {
// 					// 	k.repository.SaveProjectResource(&entity.ProjectResources{})
// 					// } else {
// 					// 	k.repository.SaveInstanceResource(&entity.InstanceResources{})
// 					// }

// 				} else {
// 					k.queueItem(AddResourceAction, obj, i.GetInformerType())
// 				}
// 			}
// 		},
// 		DeleteFunc: func(obj interface{}) {
// 			status := i.DeleteEventHandler(obj)
// 			if status.ZBI {
// 				log.Printf("Received Delete for %s. Status: %s", i.GetInformerType(), fn.MarshalObject(status))

// 				if status.Ready {
// 					k.updateStatus("DELETED", i.GetInformerType(), status)

// 					// resource := entity.KubernetesResource{
// 					// 	ProjectId:  status.ProjectId,
// 					// 	InstanceId: status.Id,
// 					// 	Name:       status.Name,
// 					// 	Type:       string(i.GetInformerType()),
// 					// 	Status:     status.Status,
// 					// 	Timestamp:  time.Now(),
// 					// }

// 					// if status.Level == entity.PROJECT_LEVEL {
// 					// 	k.repository.SaveProjectResource(&entity.ProjectResources{})
// 					// } else {
// 					// 	k.repository.SaveInstanceResource(&entity.InstanceResources{})
// 					// }

// 				} else {
// 					k.queueItem(DeleteResourceAction, obj, i.GetInformerType())
// 				}
// 			}
// 		},
// 		UpdateFunc: func(old, new interface{}) {
// 			if isObjectModified(old, new) {
// 				status := i.UpdateEventHandler(old, new)
// 				if status.ZBI {
// 					log.Printf("Received Update for %s. Status: %s", i.GetInformerType(), fn.MarshalObject(status))

// 					if status.Ready {
// 						k.updateStatus("ACTIVE", i.GetInformerType(), status)

// 						// resource := entity.KubernetesResource{
// 						// 	ProjectId:  status.ProjectId,
// 						// 	InstanceId: status.Id,
// 						// 	Name:       status.Name,
// 						// 	Type:       string(i.GetInformerType()),
// 						// 	Status:     status.Status,
// 						// 	Timestamp:  time.Now(),
// 						// }

// 						// if status.Level == entity.PROJECT_LEVEL {
// 						// 	k.repository.SaveProjectResource(&entity.ProjectResources{})
// 						// } else {
// 						// 	k.repository.SaveInstanceResource(&entity.InstanceResources{})
// 						// }

// 					} else {
// 						k.queueItem(UpdateResourceAction, new, i.GetInformerType())
// 					}
// 				}
// 			}
// 		},
// 	})

// 	k.processors[i.GetInformerType()] = i
// }

// Run
// func (k *KlientInformer) Run() {

// 	log.Print("Running KlientInformer")

// 	defer k.queue.ShutDown()
// 	defer close(k.stopper)

// 	k.factory.Start(k.stopper)
// 	k.factory.WaitForCacheSync(k.stopper)

// 	log.Print("Starting runWorker ...")
// 	go wait.Until(k.runWorker, time.Second, k.stopper)
// 	log.Print("Waiting for informers to complete")

// 	select {}
// }

// func (k *KlientInformer) runWorker() {
// 	for k.processNextItem() {

// 	}

// }

// func (k *KlientInformer) processNextItem() bool {

// 	log.Printf("Processing next item. Queue size - %d", k.queue.Len())

// 	item, quit := k.queue.Get()
// 	if quit {
// 		return false
// 	}

// 	defer k.queue.Done(item)

// 	qItem := item.(QueueItem)
// 	requeueCount := k.queue.NumRequeues(qItem)

// 	log.Printf("Action: %s, Key: %s, Kind: %s", qItem.Action, qItem.Key, qItem.Type)

// 	t := ztypes.ResourceObjectType(qItem.Type)
// 	processor := k.processors[t]
// 	obj, exists, err := processor.GetInformer().GetIndexer().GetByKey(qItem.Key)
// 	if err != nil {
// 		log.Printf("Fetching object with key %s from store failed with %v", qItem.Key, err)
// 	}

// 	if !exists {
// 		log.Printf("Object %s of type %s does not exist anymore", qItem.Key, qItem.Type)

// 		//TODO - may need to handle delete status for deployment
// 	} else {

// 		itemStatus := processor.ProcessItem(qItem.Action, obj)

// 		var saved bool = false

// 		if itemStatus.Ready {
// 			switch qItem.Action {
// 			case AddResourceAction:
// 				saved = k.updateStatus("ACTIVE", processor.GetInformerType(), itemStatus)

// 			case DeleteResourceAction:
// 				saved = k.updateStatus("DELETED", processor.GetInformerType(), itemStatus)

// 			case UpdateResourceAction:
// 				saved = k.updateStatus("ACTIVE", processor.GetInformerType(), itemStatus)
// 			}

// 			//			k.updateStatus("ACTIVE", processor.GetInformerType(), itemStatus)
// 			// resource := entity.KubernetesResource{
// 			// 	ProjectId:  itemStatus.ProjectId,
// 			// 	InstanceId: itemStatus.Id,
// 			// 	Name:       itemStatus.Name,
// 			// 	Type:       string(processor.GetInformerType()),
// 			// 	Status:     itemStatus.Status,
// 			// 	Timestamp:  time.Now(),
// 			// }

// 			// if itemStatus.Level == entity.PROJECT_LEVEL {
// 			// 	k.repository.SaveProjectResource(&entity.ProjectResources{})
// 			// } else {
// 			// 	k.repository.SaveInstanceResource(&entity.InstanceResources{})
// 			// }

// 		}

// 		if !saved {
// 			log.Printf("Resource %s of type %s not yet ready.", qItem.Key, qItem.Type)
// 			if requeueCount < k.requeueLimit {
// 				log.Printf("Attempt %d - %s not ready. Will add back to queue after %d seconds", requeueCount, qItem.Key, k.requeueDelay)
// 				go func() {
// 					time.Sleep(time.Duration(k.requeueDelay) * k.delayDuration)
// 					log.Printf("Adding %s back to the queue", qItem.Key)
// 					k.queue.AddRateLimited(qItem)
// 				}()
// 			} else {
// 				//TODO Interrogate error condition and report for monitoring ...

// 				k.updateStatus("PENDING", processor.GetInformerType(), itemStatus)
// 				// resource := entity.KubernetesResource{
// 				// 	ProjectId:  itemStatus.ProjectId,
// 				// 	InstanceId: itemStatus.Id,
// 				// 	Name:       itemStatus.Name,
// 				// 	Type:       string(processor.GetInformerType()),
// 				// 	Status:     "PENDING",
// 				// 	Timestamp:  time.Now(),
// 				// }

// 				// if itemStatus.Level == entity.PROJECT_LEVEL {
// 				// 	k.repository.SaveProjectResource(&entity.ProjectResources{})
// 				// } else {
// 				// 	k.repository.SaveInstanceResource(&entity.InstanceResources{})
// 				// }

// 			}
// 		}
// 	}

// 	return true
// }

// func (k *KlientInformer) updateStatus(status string, resourceType ztypes.ResourceObjectType, itemStatus ItemStatus) bool {

// 	if itemStatus.Level == ztypes.PROJECT_LEVEL {
// 		if resourceType == ztypes.NAMESPACE_RESOURCE {
// 			log.Printf("Updating project %s status to %s", itemStatus.ProjectId, status)
// 			err := k.repository.UpdateProjectStatus(context.Background(), itemStatus.ProjectId, status)

// 			if err != nil {
// 				return false
// 			}

// 		}
// 	} else {
// 		if resourceType == ztypes.DEPLOYMENT_RESOURCE {
// 			log.Printf("Updating instance %s status to %s", itemStatus.Id, status)
// 			err := k.repository.UpdateInstanceStatus(context.Background(), itemStatus.Id, status)

// 			if err != nil {
// 				return false
// 			}
// 		}
// 	}

// 	// only report failures back to the database ?? Perhaps, only report for monitoring
// 	if !itemStatus.Ready {
// 		log.Printf("Updating %s resource %s status to %s", itemStatus.Id, resourceType, status)
// 		// tstamp := time.Now()
// 		// rsc_obj := models.NewZBIResource(itemStatus.Id, itemStatus.Owner, &status, &tstamp, itemStatus.Network,
// 		// 	itemStatus.Level, resourceType, itemStatus.Conditions)

// 		// saved, err := k.repository.SaveProjectResourceStatus(itemStatus.Id, *rsc_obj)

// 		// if err != nil || !saved {
// 		// 	return false
// 		// }

// 	}

// 	return true
// }
