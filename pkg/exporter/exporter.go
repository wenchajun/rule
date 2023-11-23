/*
Copyright 2020 The KubeSphere Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package exporter

import (
	"fmt"
	"github.com/golang/glog"

	"sync"
)

type Factory func(receiver *v1alpha1.Receiver) (AuditExporter, error)

// AuditExporter used to send alert to receiver.
type AuditExporter interface {
	// Connect to alert receiver.
	Connect() error
	// Reconnect to the alert receiver.
	Reconnect(receiver *v1alpha1.Receiver) error
	// Name return name of export. usually it same with the name of receiver.
	Name() string
	// Type return The type of receiver which this exporter will send alert to.
	Type() string
	// DeepEqual compare the receiver of this export is same as the given receiver.
	DeepEqual(receiver *v1alpha1.Receiver) bool
	// Export alert
	Export(event *auditing.Event) error
}

var mutex sync.Mutex
var plugins map[string]Factory
var exporters map[string]AuditExporter

// RegisterPlugin used to register a new type of export must.
// Name is the type of receiver, factory must be return an
// instance of the export.
func RegisterPlugin(name string, factory Factory) {
	if plugins == nil {
		plugins = make(map[string]Factory)
	}

	plugins[name] = factory
}

// Export will send alert to all receivers.
func Export(e *auditing.Event) {

	for _, exporter := range exporters {
		err := exporter.Export(e)
		if err != nil {
			glog.Errorf("output e(%s) to(%s) error, %s", e.AuditID, exporter.Name(), err)
		}
	}

	return
}

// Connect will structure exporters from receivers.
// It will refactor all exporters from new receivers when
// call this function when receivers changed.
func Connect(receivers []v1alpha1.Receiver) []error {

	mutex.Lock()
	defer mutex.Unlock()

	var errs []error
	m := make(map[string]AuditExporter)
	for _, receiver := range receivers {

		// Reconnect if the exporter of this receiver exist.
		exporter := getExporter(receiver)
		if exporter != nil {
			if err := exporter.Reconnect(&receiver); err != nil {
				errs = append(errs, fmt.Errorf("receiver %s reconnect error, %s", exporter.Name(), err))
				continue
			}
			m[exporter.Name()] = exporter
		}

		// Get the factory of this receiver
		factory, ok := plugins[receiver.ReceiverType]
		if !ok {
			errs = append(errs, fmt.Errorf("unregister plugin %s", receiver.ReceiverType))
			continue
		}

		// Structure exporter
		exporter, err := factory(&receiver)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// Connect to the receiver
		err = exporter.Connect()
		if err != nil {
			errs = append(errs, fmt.Errorf("connect to receiver %s error, %s", receiver.ReceiverName, err))
			continue
		}

		// Add the exporter to maps, key is the name of exporter
		m[exporter.Name()] = exporter
	}

	exporters = m
	return errs
}

// Get exporter by receiver
func getExporter(receiver v1alpha1.Receiver) AuditExporter {

	// Get exporter from maps with receiver name
	exporter, ok := exporters[receiver.ReceiverName]
	if ok {
		return exporter
	}

	// If the exporter name is not same as receiver name, or receiver name
	// changed, we will traversal the map to find the exporter.
	for _, exp := range exporters {
		if exp.Type() != receiver.ReceiverType {
			continue
		}

		if exp.DeepEqual(&receiver) {
			return exp
		}
	}

	return nil
}
