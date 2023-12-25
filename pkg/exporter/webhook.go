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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/golang/glog"
	alertType "github.com/prometheus/alertmanager/types"
	"github.com/prometheus/common/model"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
	"whizard-telemetry-ruler/pkg/constant"
	"whizard-telemetry-ruler/pkg/rule"
	"whizard-telemetry-ruler/pkg/utils"
)

const (
	MaxIdleConns        = 8
	MaxConnsPerHost     = 8
	MaxIdleConnsPerHost = 8
	IdleConnTimeout     = 30 * time.Second
)

// WebhookExporter export alerts to webhook
type WebhookExporter struct {
	// Exporter name, if the receiver name is set, use receiver name,
	// else use the url of webhook
	name     string
	Client   http.Client
	URL      string
	CABundle []byte
}

func init() {
	// Register before used.
	RegisterPlugin(constant.WebhookReceiver, NewWebhookClient)
}

// NewWebhookClient create a webhook exporter.
func NewWebhookClient(receiver *Receiver) (Exporters, error) {

	wh := &WebhookExporter{}

	err := wh.GetHttpConfig(receiver)
	if err != nil {
		return nil, err
	}

	return wh, nil
}

func (wh *WebhookExporter) Connect() error {
	return nil
}

func (wh *WebhookExporter) ExportAuditingAlerts(a *rule.Auditing) error {
	msg := a.Annotations
	msgKey := "message"
	msgValue := a.Message
	if existingValue, exists := msg[msgKey]; exists {
		fmt.Printf("Key '%s' already exists with value: %s,Please change the annotation field to another field \n", msg, existingValue)
	} else {
		msg[msgKey] = msgValue
	}
	var alert alertType.Alert
	msgSet := make(model.LabelSet)
	for k, v := range msg {
		msgSet[model.LabelName(k)] = model.LabelValue(v)
	}
	alert.Annotations = msgSet
	alert.Labels = map[model.LabelName]model.LabelValue{
		"namespace":                model.LabelValue(a.ObjectRef.Namespace),
		"resource":                 model.LabelValue(a.ObjectRef.Resource),
		"name":                     model.LabelValue(a.ObjectRef.Name),
		"user":                     model.LabelValue(a.User.Username),
		"group":                    model.LabelValue(utils.OutputAsJson(a.User.Groups)),
		"verb":                     model.LabelValue(a.Verb),
		"alerttype":                "auditing",
		"alertname":                model.LabelValue(a.GetAlertRuleName()),
		"requestReceivedTimestamp": model.LabelValue(a.RequestReceivedTimestamp.String()),
	}

	request, err := http.NewRequest(http.MethodPost, wh.URL, bytes.NewBuffer([]byte(alert.String())))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := wh.Client.Do(request)
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

func (wh *WebhookExporter) ExportEventAlerts(e *rule.Event) error {

	msg := e.Annotations
	msgKey := "message"
	msgValue := e.Message
	if existingValue, exists := msg[msgKey]; exists {
		fmt.Printf("Key '%s' already exists with value: %s,Please change the annotation field to another field \n", msg, existingValue)
	} else {
		msg[msgKey] = msgValue
	}
	var alert alertType.Alert
	msgSet := make(model.LabelSet)
	for k, v := range msg {
		msgSet[model.LabelName(k)] = model.LabelValue(v)
	}
	alert.Annotations = msgSet
	alert.Labels = map[model.LabelName]model.LabelValue{
		"namespace": model.LabelValue(e.Event.Namespace),
		"reason":    model.LabelValue(e.Event.Reason),
		"name":      model.LabelValue(e.Event.Name),
		"user":      model.LabelValue(e.Event.Source.Host),
		"group":     model.LabelValue(utils.OutputAsJson(e.Event.Series)),
		"alerttype": "events",
		"alertname": model.LabelValue(e.GetAlertRuleName()),
	}

	request, err := http.NewRequest(http.MethodPost, wh.URL, bytes.NewBuffer([]byte(alert.String())))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := wh.Client.Do(request)
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

// Reconnect only reset the webhook url and ca.
func (wh *WebhookExporter) Reconnect(receiver *Receiver) error {

	err := wh.GetHttpConfig(receiver)
	if err != nil {
		return err
	}

	return nil
}

func (wh *WebhookExporter) GetHttpConfig(receiver *Receiver) error {

	if receiver == nil || receiver.ReceiverType != constant.WebhookReceiver {
		glog.Error(receiver)
		return fmt.Errorf("no webhook receiver config")
	}

	wh.CABundle = receiver.ReceiverConfig.CABundle

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
		return fmt.Errorf("no webhook receiver config")
	}
	wh.URL = url

	wh.name = receiver.ReceiverName
	if len(wh.name) == 0 {
		wh.name = wh.URL
	}

	if strings.HasPrefix(url, "https") {
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(wh.CABundle)

		wh.Client = http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: pool,
				},
				MaxIdleConns:        MaxIdleConns,
				MaxConnsPerHost:     MaxConnsPerHost,
				MaxIdleConnsPerHost: MaxIdleConnsPerHost,
				IdleConnTimeout:     IdleConnTimeout,
			},
			Timeout: time.Second,
		}
	} else {
		wh.Client = http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        MaxIdleConns,
				MaxConnsPerHost:     MaxConnsPerHost,
				MaxIdleConnsPerHost: MaxIdleConnsPerHost,
				IdleConnTimeout:     IdleConnTimeout,
			},
			Timeout: time.Second,
		}
	}

	return nil
}

func (wh *WebhookExporter) Name() string {
	return wh.name
}

// Type is "webhook"
func (wh *WebhookExporter) Type() string {
	return constant.WebhookReceiver
}

func (wh *WebhookExporter) DeepEqual(_ *Receiver) bool {

	return false
}
