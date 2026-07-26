/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package polaris

import (
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	api "github.com/polarismesh/polaris-go"
	internalapi "github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/model"
)

import (
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type item func(remoting.EventType, []model.Instance)

type subscriberKind uint8

const (
	registrySubscriber subscriberKind = iota
	applicationSubscriber
)

type initialReconcileMode uint8

const (
	reconcileWithBaseline initialReconcileMode = iota
	reconcileWithFullSnapshot
)

type subscriberState struct {
	notify             item
	notifyFullSnapshot func([]model.Instance)
	initialSnapshot    []model.Instance
	initialMode        initialReconcileMode
	reconciled         bool
	kind               subscriberKind
}

type PolarisServiceWatcher struct {
	consumer         api.ConsumerAPI
	subscribeParam   *api.WatchServiceRequest
	lock             *sync.Mutex
	subscribers      []*subscriberState
	execOnce         *sync.Once
	currentInstances []model.Instance
	snapshotReady    bool
}

// newPolarisWatcher create PolarisServiceWatcher to do watch service action
func newPolarisWatcher(param *api.WatchServiceRequest, consumer api.ConsumerAPI) (*PolarisServiceWatcher, error) {
	watcher := &PolarisServiceWatcher{
		subscribeParam: param,
		consumer:       consumer,
		lock:           &sync.Mutex{},
		subscribers:    make([]*subscriberState, 0),
		execOnce:       &sync.Once{},
	}
	return watcher, nil
}

// AddSubscriber add subscriber into watcher's subscribers
func (watcher *PolarisServiceWatcher) AddSubscriber(
	subscriber func(remoting.EventType, []model.Instance),
) {
	state := &subscriberState{
		notify:     item(subscriber),
		reconciled: true,
		kind:       applicationSubscriber,
	}

	func() {
		watcher.lock.Lock()
		defer watcher.lock.Unlock()

		watcher.subscribers = append(watcher.subscribers, state)
	}()

	watcher.lazyRun()
}

// lazyRun Delayed execution, only triggered when AddSubscriber is called, and will only be executed once
func (watcher *PolarisServiceWatcher) lazyRun() {
	watcher.execOnce.Do(func() {
		go watcher.startWatch()
	})
}

func (watcher *PolarisServiceWatcher) addRegistrySubscriber(
	initialSnapshot []model.Instance,
	mode initialReconcileMode,
	subscriber item,
	fullSnapshotSubscriber func([]model.Instance),
) {
	state := &subscriberState{
		notify:             subscriber,
		notifyFullSnapshot: fullSnapshotSubscriber,
		initialSnapshot:    copyInstances(initialSnapshot),
		initialMode:        mode,
		kind:               registrySubscriber,
	}

	func() {
		watcher.lock.Lock()
		defer watcher.lock.Unlock()

		watcher.subscribers = append(watcher.subscribers, state)
		if watcher.snapshotReady {
			watcher.reconcileSubscriberLocked(state, notifiableInstances(watcher.currentInstances, true))
		}
	}()

	// Start only after the subscriber is registered and any available current
	// snapshot has been replayed.
	watcher.lazyRun()
}

// missingInstances returns previous - current by model.InstanceKey while
// preserving the order of the previous snapshot.
func missingInstances(previous, current []model.Instance) []model.Instance {
	currentInstances := make(map[model.InstanceKey]struct{}, len(current))
	for _, instance := range current {
		currentInstances[instance.GetInstanceKey()] = struct{}{}
	}

	missing := make([]model.Instance, 0, len(previous))
	for _, instance := range previous {
		if _, ok := currentInstances[instance.GetInstanceKey()]; !ok {
			missing = append(missing, instance)
		}
	}
	return missing
}

func notifiableInstances(instances []model.Instance, reportInvalid bool) []model.Instance {
	valid := make([]model.Instance, 0, len(instances))
	for _, instance := range instances {
		validationError := polarisInstanceURLValidationError(instance)
		if validationError == "" {
			valid = append(valid, instance)
		} else if reportInvalid {
			logger.Errorf("[Registry][Polaris] %s, instance=%+v", validationError, instance)
		}
	}
	return valid
}

// handleWatchSnapshot replaces the watcher's current state and reconciles each
// subscriber's own synchronous-load baseline exactly once.
func (watcher *PolarisServiceWatcher) handleWatchSnapshot(current []model.Instance) {
	watcher.lock.Lock()
	defer watcher.lock.Unlock()

	next := copyInstances(current)
	registryCurrent := notifiableInstances(next, watcher.hasRegistrySubscriberLocked())
	previous := watcher.currentInstances
	hadPreviousSnapshot := watcher.snapshotReady
	var removedSincePrevious []model.Instance
	if hadPreviousSnapshot {
		registryPrevious := notifiableInstances(previous, false)
		removedSincePrevious = missingInstances(registryPrevious, registryCurrent)
	}
	watcher.currentInstances = next
	watcher.snapshotReady = true
	for _, subscriber := range watcher.subscribers {
		if subscriber.kind == applicationSubscriber {
			watcher.notifySubscriberLocked(subscriber, remoting.EventTypeAdd, watcher.currentInstances)
			continue
		}
		if subscriber.reconciled {
			if len(removedSincePrevious) > 0 {
				watcher.notifySubscriberLocked(subscriber, remoting.EventTypeDel, removedSincePrevious)
			}
			watcher.notifySubscriberLocked(subscriber, remoting.EventTypeAdd, registryCurrent)
			continue
		}
		watcher.reconcileSubscriberLocked(subscriber, registryCurrent)
	}
}

func (watcher *PolarisServiceWatcher) reconcileSubscriberLocked(
	subscriber *subscriberState,
	current []model.Instance,
) {
	switch subscriber.initialMode {
	case reconcileWithBaseline:
		missing := missingInstances(subscriber.initialSnapshot, current)
		if len(missing) > 0 {
			watcher.notifySubscriberLocked(subscriber, remoting.EventTypeDel, missing)
		}
		watcher.notifySubscriberLocked(subscriber, remoting.EventTypeAdd, current)
	case reconcileWithFullSnapshot:
		subscriber.notifyFullSnapshot(copyInstances(current))
	}
	subscriber.reconciled = true
	subscriber.initialSnapshot = nil
}

// startWatch start run work to watch target service by polaris
func (watcher *PolarisServiceWatcher) startWatch() {
	for {
		if err := watcher.watchOnce(); err != nil {
			time.Sleep(time.Duration(500 * time.Millisecond))
		}
	}
}

func (watcher *PolarisServiceWatcher) watchOnce() error {
	resp, err := watcher.consumer.WatchService(watcher.subscribeParam)
	if err != nil {
		return err
	}
	watcher.handleWatchSnapshot(resp.GetAllInstancesResp.Instances)
	for event := range resp.EventChannel {
		if event.GetSubScribeEventType() == internalapi.EventInstance {
			watcher.handleInstanceEvent(event.(*model.InstanceEvent))
		}
	}
	return nil
}

func (watcher *PolarisServiceWatcher) handleInstanceEvent(event *model.InstanceEvent) {
	if event == nil {
		return
	}

	watcher.lock.Lock()
	defer watcher.lock.Unlock()

	if event.AddEvent != nil {
		instances := copyInstances(event.AddEvent.Instances)
		for _, instance := range instances {
			watcher.upsertCurrentInstanceLocked(instance)
		}
		watcher.notifyReconciledSubscribersLocked(remoting.EventTypeAdd, instances)
	}
	if event.UpdateEvent != nil {
		reportInvalid := watcher.hasRegistrySubscriberLocked()
		applicationUpdates := make([]model.Instance, 0, len(event.UpdateEvent.UpdateList))
		registryUpdates := make([]model.Instance, 0, len(event.UpdateEvent.UpdateList))
		registryDeletes := make([]model.Instance, 0, len(event.UpdateEvent.UpdateList))
		changedKeyBefore := make([]model.Instance, 0, len(event.UpdateEvent.UpdateList))
		updates := make([]model.OneInstanceUpdate, 0, len(event.UpdateEvent.UpdateList))
		for _, update := range event.UpdateEvent.UpdateList {
			before := newPolarisInstanceSnapshot(update.Before)
			after := newPolarisInstanceSnapshot(update.After)
			updates = append(updates, model.OneInstanceUpdate{Before: before, After: after})
			if before.GetInstanceKey() != after.GetInstanceKey() {
				changedKeyBefore = append(changedKeyBefore, before)
			}
		}
		watcher.removeCurrentInstancesLocked(changedKeyBefore)
		for _, update := range updates {
			beforeValid := polarisInstanceURLValidationError(update.Before) == ""
			afterValidationError := polarisInstanceURLValidationError(update.After)
			afterValid := afterValidationError == ""
			keyChanged := update.Before.GetInstanceKey() != update.After.GetInstanceKey()
			watcher.upsertCurrentInstanceLocked(update.After)
			applicationUpdates = append(applicationUpdates, update.After)
			if keyChanged && beforeValid {
				registryDeletes = append(registryDeletes, update.Before)
			}
			if afterValid {
				registryUpdates = append(registryUpdates, update.After)
				continue
			}
			if reportInvalid {
				logger.Errorf("[Registry][Polaris] %s, instance=%+v", afterValidationError, update.After)
			}
			if !keyChanged && beforeValid {
				registryDeletes = append(registryDeletes, update.Before)
			}
		}
		watcher.notifyReconciledSubscribersByKindLocked(applicationSubscriber, remoting.EventTypeUpdate, applicationUpdates)
		watcher.notifyReconciledSubscribersByKindLocked(registrySubscriber, remoting.EventTypeDel, registryDeletes)
		watcher.notifyReconciledSubscribersByKindLocked(registrySubscriber, remoting.EventTypeUpdate, registryUpdates)
	}
	if event.DeleteEvent != nil {
		instances := copyInstances(event.DeleteEvent.Instances)
		watcher.removeCurrentInstancesLocked(instances)
		watcher.notifyReconciledSubscribersLocked(remoting.EventTypeDel, instances)
	}
}

func (watcher *PolarisServiceWatcher) upsertCurrentInstanceLocked(instance model.Instance) {
	key := instance.GetInstanceKey()
	for i := range watcher.currentInstances {
		if watcher.currentInstances[i].GetInstanceKey() == key {
			watcher.currentInstances[i] = instance
			return
		}
	}
	watcher.currentInstances = append(watcher.currentInstances, instance)
}

func (watcher *PolarisServiceWatcher) removeCurrentInstancesLocked(instances []model.Instance) {
	keys := make(map[model.InstanceKey]struct{}, len(instances))
	for _, instance := range instances {
		keys[instance.GetInstanceKey()] = struct{}{}
	}

	old := watcher.currentInstances
	current := old[:0]
	for _, instance := range old {
		if _, remove := keys[instance.GetInstanceKey()]; !remove {
			current = append(current, instance)
		}
	}
	clear(old[len(current):])
	watcher.currentInstances = current
}

func (watcher *PolarisServiceWatcher) notifyReconciledSubscribersLocked(eventType remoting.EventType, instances []model.Instance) {
	for _, subscriber := range watcher.subscribers {
		if subscriber.reconciled {
			watcher.notifySubscriberLocked(subscriber, eventType, instances)
		}
	}
}

func (watcher *PolarisServiceWatcher) hasRegistrySubscriberLocked() bool {
	for _, subscriber := range watcher.subscribers {
		if subscriber.kind == registrySubscriber {
			return true
		}
	}
	return false
}

func (watcher *PolarisServiceWatcher) notifyReconciledSubscribersByKindLocked(
	kind subscriberKind,
	eventType remoting.EventType,
	instances []model.Instance,
) {
	if len(instances) == 0 {
		return
	}
	for _, subscriber := range watcher.subscribers {
		if subscriber.reconciled && subscriber.kind == kind {
			watcher.notifySubscriberLocked(subscriber, eventType, instances)
		}
	}
}

func (watcher *PolarisServiceWatcher) notifySubscriberLocked(
	subscriber *subscriberState,
	eventType remoting.EventType,
	instances []model.Instance,
) {
	subscriber.notify(eventType, copyInstances(instances))
}

func copyInstances(instances []model.Instance) []model.Instance {
	if len(instances) == 0 {
		return nil
	}

	copied := make([]model.Instance, len(instances))
	for i, instance := range instances {
		copied[i] = newPolarisInstanceSnapshot(instance)
	}
	return copied
}

type polarisInstanceSnapshot struct {
	instanceKey          model.InstanceKey
	namespace            string
	service              string
	id                   string
	host                 string
	port                 uint32
	vpcID                string
	protocol             string
	version              string
	weight               int
	priority             uint32
	metadata             map[string]string
	logicSet             string
	circuitBreakerStatus model.CircuitBreakerStatus
	healthy              bool
	isolated             bool
	enableHealthCheck    bool
	region               string
	zone                 string
	idc                  string
	campus               string
	revision             string
}

func newPolarisInstanceSnapshot(instance model.Instance) model.Instance {
	if instance == nil {
		return nil
	}
	return &polarisInstanceSnapshot{
		instanceKey:          instance.GetInstanceKey(),
		namespace:            instance.GetNamespace(),
		service:              instance.GetService(),
		id:                   instance.GetId(),
		host:                 instance.GetHost(),
		port:                 instance.GetPort(),
		vpcID:                instance.GetVpcId(),
		protocol:             instance.GetProtocol(),
		version:              instance.GetVersion(),
		weight:               instance.GetWeight(),
		priority:             instance.GetPriority(),
		metadata:             copyStringMap(instance.GetMetadata()),
		logicSet:             instance.GetLogicSet(),
		circuitBreakerStatus: instance.GetCircuitBreakerStatus(),
		healthy:              instance.IsHealthy(),
		isolated:             instance.IsIsolated(),
		enableHealthCheck:    instance.IsEnableHealthCheck(),
		region:               instance.GetRegion(),
		zone:                 instance.GetZone(),
		idc:                  instance.GetIDC(),
		campus:               instance.GetCampus(),
		revision:             instance.GetRevision(),
	}
}

func copyStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	copied := make(map[string]string, len(source))
	for key, value := range source {
		copied[key] = value
	}
	return copied
}

func (instance *polarisInstanceSnapshot) GetInstanceKey() model.InstanceKey {
	return instance.instanceKey
}

func (instance *polarisInstanceSnapshot) GetNamespace() string {
	return instance.namespace
}

func (instance *polarisInstanceSnapshot) GetService() string {
	return instance.service
}

func (instance *polarisInstanceSnapshot) GetId() string {
	return instance.id
}

func (instance *polarisInstanceSnapshot) GetHost() string {
	return instance.host
}

func (instance *polarisInstanceSnapshot) GetPort() uint32 {
	return instance.port
}

func (instance *polarisInstanceSnapshot) GetVpcId() string {
	return instance.vpcID
}

func (instance *polarisInstanceSnapshot) GetProtocol() string {
	return instance.protocol
}

func (instance *polarisInstanceSnapshot) GetVersion() string {
	return instance.version
}

func (instance *polarisInstanceSnapshot) GetWeight() int {
	return instance.weight
}

func (instance *polarisInstanceSnapshot) GetPriority() uint32 {
	return instance.priority
}

func (instance *polarisInstanceSnapshot) GetMetadata() map[string]string {
	return instance.metadata
}

func (instance *polarisInstanceSnapshot) GetLogicSet() string {
	return instance.logicSet
}

func (instance *polarisInstanceSnapshot) GetCircuitBreakerStatus() model.CircuitBreakerStatus {
	return instance.circuitBreakerStatus
}

func (instance *polarisInstanceSnapshot) IsHealthy() bool {
	return instance.healthy
}

func (instance *polarisInstanceSnapshot) IsIsolated() bool {
	return instance.isolated
}

func (instance *polarisInstanceSnapshot) IsEnableHealthCheck() bool {
	return instance.enableHealthCheck
}

func (instance *polarisInstanceSnapshot) GetRegion() string {
	return instance.region
}

func (instance *polarisInstanceSnapshot) GetZone() string {
	return instance.zone
}

func (instance *polarisInstanceSnapshot) GetIDC() string {
	return instance.idc
}

func (instance *polarisInstanceSnapshot) GetCampus() string {
	return instance.campus
}

func (instance *polarisInstanceSnapshot) GetRevision() string {
	return instance.revision
}
