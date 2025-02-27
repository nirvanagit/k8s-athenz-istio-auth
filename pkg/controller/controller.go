// Copyright 2019, Verizon Media Inc.
// Licensed under the terms of the 3-Clause BSD license. See LICENSE file in
// github.com/yahoo/k8s-athenz-istio-auth for terms.
package controller

import (
	"errors"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"

	"istio.io/istio/pilot/pkg/config/kube/crd"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pilot/pkg/serviceregistry/kube"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	adv1 "github.com/yahoo/k8s-athenz-istio-auth/pkg/apis/athenz/v1"
	"github.com/yahoo/k8s-athenz-istio-auth/pkg/athenz"
	m "github.com/yahoo/k8s-athenz-istio-auth/pkg/athenz"
	adClientset "github.com/yahoo/k8s-athenz-istio-auth/pkg/client/clientset/versioned"
	adInformer "github.com/yahoo/k8s-athenz-istio-auth/pkg/client/informers/externalversions/athenz/v1"
	"github.com/yahoo/k8s-athenz-istio-auth/pkg/istio/onboarding"
	"github.com/yahoo/k8s-athenz-istio-auth/pkg/istio/processor"
	"github.com/yahoo/k8s-athenz-istio-auth/pkg/istio/rbac"
	rbacv1 "github.com/yahoo/k8s-athenz-istio-auth/pkg/istio/rbac/v1"
	"github.com/yahoo/k8s-athenz-istio-auth/pkg/log"
)

const (
	queueNumRetries = 3
	logPrefix       = "[controller]"
)

type Controller struct {
	configStoreCache     model.ConfigStoreCache
	crcController        *onboarding.Controller
	processor            *processor.Controller
	serviceIndexInformer cache.SharedIndexInformer
	adIndexInformer      cache.SharedIndexInformer
	rbacProvider         rbac.Provider
	queue                workqueue.RateLimitingInterface
	adResyncInterval     time.Duration
}

// convertSliceToKeyedMap converts the input model.Config slice into a map with (type/namespace/name) formatted key
func convertSliceToKeyedMap(in []model.Config) map[string]model.Config {
	out := make(map[string]model.Config, len(in))
	for _, c := range in {
		key := c.Key()
		out[key] = c
	}
	return out
}

// equal compares the Spec of two model.Config items
func equal(c1, c2 model.Config) bool {
	return c1.Key() == c2.Key() && proto.Equal(c1.Spec, c2.Spec)
}

// computeChangeList determines a list of change operations to convert the current state of model.Config items into the
// desired state of model.Config items in the following manner:
// 1. Converts the current and desired slices into a map for quick lookup
// 2. Loops through the desired slice of items and identifies items that need to be created/updated
// 3. Loops through the current slice of items and identifies items that need to be deleted
func computeChangeList(current []model.Config, desired []model.Config, errHandler processor.OnErrorFunc) []*processor.Item {

	currMap := convertSliceToKeyedMap(current)
	desiredMap := convertSliceToKeyedMap(desired)

	changeList := make([]*processor.Item, 0)

	// loop through the desired slice of model.Config and add the items that need to be created or updated
	for _, desiredConfig := range desired {
		key := desiredConfig.Key()
		existingConfig, exists := currMap[key]
		if !exists {
			item := processor.Item{
				Operation:    model.EventAdd,
				Resource:     desiredConfig,
				ErrorHandler: errHandler,
			}
			changeList = append(changeList, &item)
			continue
		}

		if !equal(existingConfig, desiredConfig) {
			// copy metadata(for resource version) from current config to desired config
			desiredConfig.ConfigMeta = existingConfig.ConfigMeta
			item := processor.Item{
				Operation:    model.EventUpdate,
				Resource:     desiredConfig,
				ErrorHandler: errHandler,
			}
			changeList = append(changeList, &item)
			continue
		}
	}

	// loop through the current slice of model.Config and add the items that need to be deleted
	for _, currConfig := range current {
		key := currConfig.Key()
		_, exists := desiredMap[key]
		if !exists {
			item := processor.Item{
				Operation:    model.EventDelete,
				Resource:     currConfig,
				ErrorHandler: errHandler,
			}
			changeList = append(changeList, &item)
		}
	}

	return changeList
}

// getErrHandler returns a error handler func that re-adds the athenz domain back to queue
// this explicit func definition takes in the key to avoid data race while accessing key
func (c *Controller) getErrHandler(key string) processor.OnErrorFunc {
	return func(err error, item *processor.Item) error {
		if err != nil {
			if item != nil {
				log.Errorf("%s Error performing %s on %s: %s", logPrefix, item.Operation, item.Resource.Key(), err)
			}
			c.queue.AddRateLimited(key)
		}
		return nil
	}
}

// sync will be ran for each key in the queue and will be responsible for the following:
// 1. Get the Athenz Domain from the cache for the queue key
// 2. Convert to Athenz Model to group domain members and policies by role
// 3. Convert Athenz Model to Service Role and Service Role Binding objects
// 4. Create / Update / Delete Service Role and Service Role Binding objects
func (c *Controller) sync(key string) error {
	athenzDomainRaw, exists, err := c.adIndexInformer.GetIndexer().GetByKey(key)
	if err != nil {
		return err
	}

	if !exists {
		// TODO, add the non existing athenz domain to the istio custom resource
		// processing controller to delete them
		return fmt.Errorf("athenz domain %s does not exist in cache", key)
	}

	athenzDomain, ok := athenzDomainRaw.(*adv1.AthenzDomain)
	if !ok {
		return errors.New("athenz domain cast failed")
	}

	signedDomain := athenzDomain.Spec.SignedDomain
	domainRBAC := m.ConvertAthenzPoliciesIntoRbacModel(signedDomain.Domain)
	desiredCRs := c.rbacProvider.ConvertAthenzModelIntoIstioRbac(domainRBAC)
	currentCRs := c.rbacProvider.GetCurrentIstioRbac(domainRBAC, c.configStoreCache)
	errHandler := c.getErrHandler(key)

	changeList := computeChangeList(currentCRs, desiredCRs, errHandler)
	for _, item := range changeList {
		c.processor.ProcessConfigChange(item)
	}

	return nil
}

// NewController is responsible for creating the main controller object and
// initializing all of its dependencies:
// 1. Rate limiting queue
// 2. Istio custom resource config store cache for service role, service role
//    bindings, and cluster rbac config
// 3. Onboarding controller responsible for creating / updating / deleting the
//    cluster rbac config object based on a service label
// 4. Service shared index informer
// 5. Athenz Domain shared index informer
func NewController(dnsSuffix string, istioClient *crd.Client, k8sClient kubernetes.Interface, adClient adClientset.Interface, adResyncInterval, crcResyncInterval time.Duration) *Controller {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	configStoreCache := crd.NewController(istioClient, kube.ControllerOptions{})

	serviceListWatch := cache.NewListWatchFromClient(k8sClient.CoreV1().RESTClient(), "services", v1.NamespaceAll, fields.Everything())
	serviceIndexInformer := cache.NewSharedIndexInformer(serviceListWatch, &v1.Service{}, 0, nil)
	processor := processor.NewController(configStoreCache)
	crcController := onboarding.NewController(configStoreCache, dnsSuffix, serviceIndexInformer, crcResyncInterval, processor)
	adIndexInformer := adInformer.NewAthenzDomainInformer(adClient, v1.NamespaceAll, 0, cache.Indexers{})

	c := &Controller{
		serviceIndexInformer: serviceIndexInformer,
		adIndexInformer:      adIndexInformer,
		configStoreCache:     configStoreCache,
		crcController:        crcController,
		processor:            processor,
		rbacProvider:         rbacv1.NewProvider(),
		queue:                queue,
		adResyncInterval:     adResyncInterval,
	}

	configStoreCache.RegisterEventHandler(model.ServiceRole.Type, c.processConfigEvent)
	configStoreCache.RegisterEventHandler(model.ServiceRoleBinding.Type, c.processConfigEvent)
	configStoreCache.RegisterEventHandler(model.ClusterRbacConfig.Type, crcController.EventHandler)

	adIndexInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.processEvent(cache.MetaNamespaceKeyFunc, obj)
		},
		UpdateFunc: func(_, obj interface{}) {
			c.processEvent(cache.MetaNamespaceKeyFunc, obj)
		},
		DeleteFunc: func(obj interface{}) {
			c.processEvent(cache.DeletionHandlingMetaNamespaceKeyFunc, obj)
		},
	})

	return c
}

// processEvent is responsible for calling the key function and adding the
// key of the item to the queue
func (c *Controller) processEvent(fn cache.KeyFunc, obj interface{}) {
	key, err := fn(obj)
	if err == nil {
		c.queue.Add(key)
		return
	}
	log.Errorf("%s processEvent(): Error calling key func: %s", logPrefix, err.Error())
}

// processEvent is responsible for adding the key of the item to the queue
func (c *Controller) processConfigEvent(config model.Config, e model.Event) {
	domain := athenz.NamespaceToDomain(config.Namespace)
	key := config.Namespace + "/" + domain
	c.queue.Add(key)
}

// Run starts the main controller loop running sync at every poll interval. It
// also starts the following controller dependencies:
// 1. Service informer
// 2. Istio custom resource informer
// 3. Athenz Domain informer
func (c *Controller) Run(stopCh <-chan struct{}) {
	go c.serviceIndexInformer.Run(stopCh)
	go c.configStoreCache.Run(stopCh)
	go c.adIndexInformer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, c.configStoreCache.HasSynced, c.serviceIndexInformer.HasSynced, c.adIndexInformer.HasSynced) {
		log.Panicf("%s Run(): Timed out waiting for namespace cache to sync.", logPrefix)
	}

	// crc controller must wait for service informer to sync before starting
	go c.processor.Run(stopCh)
	go c.crcController.Run(stopCh)
	go c.resync(stopCh)

	defer c.queue.ShutDown()
	wait.Until(c.runWorker, 0, stopCh)
}

// runWorker calls processNextItem to process events of the work queue
func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}

// processNextItem takes an item off the queue and calls the controllers sync
// function, handles the logic of requeuing in case any errors occur
func (c *Controller) processNextItem() bool {
	keyRaw, quit := c.queue.Get()
	if quit {
		return false
	}

	defer c.queue.Done(keyRaw)
	key, ok := keyRaw.(string)
	if !ok {
		log.Errorf("%s processNextItem(): String cast failed for key %v", logPrefix, key)
		return true
	}

	log.Infof("%s processNextItem(): Processing key: %s", logPrefix, key)
	err := c.sync(key)
	if err != nil {
		log.Errorf("%s processNextItem(): Error syncing athenz state for key %s: %s", logPrefix, keyRaw, err)
		if c.queue.NumRequeues(keyRaw) < queueNumRetries {
			log.Infof("%s processNextItem(): Retrying key %s due to sync error", logPrefix, keyRaw)
			c.queue.AddRateLimited(keyRaw)
			return true
		}
	}

	c.queue.Forget(keyRaw)
	return true
}

// resync will run as a periodic resync at a given interval, it will take all
// the current athenz domains in the cache and put them onto the queue
func (c *Controller) resync(stopCh <-chan struct{}) {
	t := time.NewTicker(c.adResyncInterval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			log.Infof("%s Running resync for athenz domains...", logPrefix)
			adListRaw := c.adIndexInformer.GetIndexer().List()
			for _, adRaw := range adListRaw {
				c.processEvent(cache.MetaNamespaceKeyFunc, adRaw)
			}
		case <-stopCh:
			log.Infof("%s Stopping athenz domain resync...", logPrefix)
			return
		}
	}
}
