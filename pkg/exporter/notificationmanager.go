/*
Copyright 2023 The KubeSphere Authors.

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
	"bytes"
	"fmt"
	"github.com/golang/glog"
	"github.com/prometheus/alertmanager/template"
	"io"
	"io/ioutil"
	"whizard-telemetry-ruler/pkg/constant"
	"whizard-telemetry-ruler/pkg/rule"
	"whizard-telemetry-ruler/pkg/utils"

	"net/http"
)

// NotificationExporter export alerts to Notification Manager
type NotificationManagerExporter struct {
	// Exporter name, if the receiver name is set, use receiver name,
	// else use the url of notificationmanager
	name   string
	URL    string
	client *http.Client
}

func init() {
	// Register before used.
	RegisterPlugin(constant.NotificationManagerReciver, NewNotificationManagerExporter)
}

// NewNotificationManagerExporter create a notification manager exporter.
func NewNotificationManagerExporter(receiver *Receiver) (Exporters, error) {

	wh := &NotificationManagerExporter{}

	err := wh.GetHttpConfig(receiver)
	if err != nil {
		return nil, err
	}

	return wh, nil
}

func (nm *NotificationManagerExporter) Connect() error {
	return nil
}

func (nm *NotificationManagerExporter) ExportAuditingAlerts(a *rule.Auditing) error {

	msg := a.Annotations
	msgKey := "message"
	msgValue := a.Message
	if existingValue, exists := msg[msgKey]; exists {
		fmt.Printf("Key '%s' already exists with value: %s,Please change the annotation field to another field \n", msg, existingValue)
	} else {
		msg[msgKey] = msgValue
	}
	data := template.Data{
		Alerts: template.Alerts{
			{
				Labels: map[string]string{
					"namespace":                a.ObjectRef.Namespace,
					"resource":                 a.ObjectRef.Resource,
					"name":                     a.ObjectRef.Name,
					"user":                     a.User.Username,
					"group":                    utils.OutputAsJson(a.User.Groups),
					"verb":                     a.Verb,
					"alerttype":                "auditing",
					"alertname":                a.GetAlertRuleName(),
					"requestReceivedTimestamp": a.RequestReceivedTimestamp.String(),
				},
				Annotations: msg,
			},
		},
	}

	s, err := utils.ToJsonString(data)
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPost, nm.URL, bytes.NewBuffer([]byte(s)))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := nm.client.Do(request)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%s", response.Status)
	}

	// Discard the body
	_, _ = io.Copy(ioutil.Discard, response.Body)

	return nil
}

func (nm *NotificationManagerExporter) ExportEventAlerts(e *rule.Event) error {

	msg := e.Annotations
	msgKey := "message"
	msgValue := e.Message
	if existingValue, exists := msg[msgKey]; exists {
		fmt.Printf("Key '%s' already exists with value: %s,Please change the annotation field to another field \n", msg, existingValue)
	} else {
		msg[msgKey] = msgValue
	}
	data := template.Data{
		Alerts: template.Alerts{
			{
				Labels: map[string]string{
					"namespace": e.Event.Namespace,
					"reason":    e.Event.Reason,
					"name":      e.Event.Name,
					"user":      e.Event.Source.Host,
					"group":     utils.OutputAsJson(e.Event.Series),
					"alerttype": "events",
					"alertname": e.GetAlertRuleName(),
				},
				Annotations: msg,
			},
		},
	}

	s, err := utils.ToJsonString(data)
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPost, nm.URL, bytes.NewBuffer([]byte(s)))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := nm.client.Do(request)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%s", response.Status)
	}

	// Discard the body
	_, _ = io.Copy(ioutil.Discard, response.Body)

	return nil
}

// Reconnect only reset the notification manager url.
func (nm *NotificationManagerExporter) Reconnect(receiver *Receiver) error {

	err := nm.GetHttpConfig(receiver)
	if err != nil {
		return err
	}

	return nil
}

func (nm *NotificationManagerExporter) GetHttpConfig(receiver *Receiver) error {

	if receiver == nil || receiver.ReceiverType != constant.NotificationManagerReciver {
		glog.Error(receiver)
		return fmt.Errorf("no notification manager receiver config")
	}

	url := ""
	if receiver.ReceiverConfig.URL != nil {
		url = *receiver.ReceiverConfig.URL
	} else if receiver.ReceiverConfig.Service != nil {
		service := receiver.ReceiverConfig.Service
		if len(receiver.ReceiverConfig.CABundle) > 0 {
			url = fmt.Sprintf("https://%s.%s", service.Name, service.Namespace)
		} else {
			url = fmt.Sprintf("http://%s.%s", service.Name, service.Namespace)
		}

		if service.Port != nil {
			url = fmt.Sprintf("%s:%d", url, *service.Port)
		}

		if service.Path != nil {
			url = fmt.Sprintf("%s%s", url, *service.Path)
		}
	} else {
		return fmt.Errorf("no notification manager receiver config")
	}
	nm.URL = url

	nm.name = receiver.ReceiverName
	if len(nm.name) == 0 {
		nm.name = nm.URL
	}

	nm.client = &http.Client{}

	return nil
}

func (nm *NotificationManagerExporter) Name() string {
	return nm.name
}

// Type is "notificationmanager"
func (nm *NotificationManagerExporter) Type() string {
	return constant.NotificationManagerReciver
}

func (nm *NotificationManagerExporter) DeepEqual(_ *Receiver) bool {

	return false
}
