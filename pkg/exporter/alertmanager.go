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
	"context"
	"fmt"

	"reflect"
	"time"

	kit "github.com/kubesphere/alertmanager-kit"
)

const (
	DefaultNamespace = "kubesphere-monitoring-system"
	DefaultName      = "alertmanager-main"
	DefaultPort      = 9093
)

// Exporter to AlertManager
type AlertManagerExporter struct {
	// Exporter name, if the receiver name is set, use receiver name,
	// else use the url of AlertManagerExporter
	name string
	// AlertManagerExporter kit config
	kit.ClientConfig
	client *kit.AlertmanagerClient
}

func init() {
	// Register before used.
	RegisterPlugin(constant.AlertManagerReciver, NewAlertManager)
}

func NewAlertManager(receiver *v1alpha1.Receiver) (AuditExporter, error) {

	if receiver == nil || receiver.ReceiverType != constant.AlertManagerReciver {
		return nil, fmt.Errorf("no alertmanager config")
	}

	am := &AlertManagerExporter{}
	am.initFromReceiver(receiver)

	return am, nil
}

func (am *AlertManagerExporter) Connect() error {

	// Create client for alertmanager service
	var err error
	am.client, err = kit.NewClient(am.ClientConfig)
	return err
}

func (am *AlertManagerExporter) Export(e *auditing.Event) error {

	ctx, cancle := context.WithTimeout(context.Background(), time.Second*10)
	defer cancle()
	err := am.client.PostAlerts(ctx, []*kit.RawAlert{
		{
			Labels: map[string]string{
				"namespace":                e.ObjectRef.Namespace,
				"resource":                 e.ObjectRef.Resource,
				"name":                     e.ObjectRef.Name,
				"user":                     e.User.Username,
				"group":                    utils.OutputAsJson(e.User.Groups),
				"verb":                     e.Verb,
				"alerttype":                "auditing",
				"alertname":                e.GetAlertRuleName(),
				"requestReceivedTimestamp": e.RequestReceivedTimestamp.String(),
			},
			Annotations: map[string]string{"message": e.Message},
		},
	})

	return err
}

func (am *AlertManagerExporter) Reconnect(receiver *v1alpha1.Receiver) error {

	if receiver == nil || receiver.ReceiverType != constant.AlertManagerReciver {
		return fmt.Errorf("no alertmanager config")
	}

	am.initFromReceiver(receiver)
	return am.Connect()
}

func (am *AlertManagerExporter) Name() string {
	return am.name
}

func (am *AlertManagerExporter) Type() string {
	return constant.AlertManagerReciver
}

func (am *AlertManagerExporter) DeepEqual(receiver *v1alpha1.Receiver) bool {

	nam := &AlertManagerExporter{}
	nam.initFromReceiver(receiver)
	if reflect.DeepEqual(nam, am) {
		return true
	}

	return false

}
func (am *AlertManagerExporter) initFromReceiver(receiver *v1alpha1.Receiver) {

	port := DefaultPort
	am.name = receiver.ReceiverName
	am.ClientConfig = kit.ClientConfig{
		Service: &kit.ServiceReference{
			Namespace: DefaultNamespace,
			Name:      DefaultName,
			Port:      &port,
		},
	}

	if receiver.ReceiverConfig.Service != nil {
		am.Service.Namespace = receiver.ReceiverConfig.Service.Namespace
		am.Service.Name = receiver.ReceiverConfig.Service.Name
		if &receiver.ReceiverConfig.Service.Port != nil {
			port := int(*receiver.ReceiverConfig.Service.Port)
			am.Service.Port = &port
		}
	} else if receiver.ReceiverConfig.URL != nil {
		am.URL = *receiver.ReceiverConfig.URL
	}

	return
}
