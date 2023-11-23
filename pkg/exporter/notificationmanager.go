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
	"bytes"
	"fmt"
	"github.com/prometheus/alertmanager/template"
	"io"
	"io/ioutil"

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
func NewNotificationManagerExporter(receiver *v1alpha1.Receiver) (AuditExporter, error) {

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

func (nm *NotificationManagerExporter) Export(e *auditing.Event) error {

	data := template.Data{
		Alerts: template.Alerts{
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
func (nm *NotificationManagerExporter) Reconnect(receiver *v1alpha1.Receiver) error {

	err := nm.GetHttpConfig(receiver)
	if err != nil {
		return err
	}

	return nil
}

func (nm *NotificationManagerExporter) GetHttpConfig(receiver *v1alpha1.Receiver) error {

	if receiver == nil || receiver.ReceiverType != constant.NotificationManagerReciver {
		fmt.Println(receiver)
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

func (nm *NotificationManagerExporter) DeepEqual(_ *v1alpha1.Receiver) bool {

	return false
}
