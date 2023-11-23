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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"

	"net/http"
	"strings"
	"time"
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
func NewWebhookClient(receiver *v1alpha1.Receiver) (AuditExporter, error) {

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

func (wh *WebhookExporter) Export(e *auditing.Event) error {

	request, err := http.NewRequest(http.MethodPost, wh.URL, bytes.NewBuffer([]byte(e.ToString())))
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
func (wh *WebhookExporter) Reconnect(receiver *v1alpha1.Receiver) error {

	err := wh.GetHttpConfig(receiver)
	if err != nil {
		return err
	}

	return nil
}

func (wh *WebhookExporter) GetHttpConfig(receiver *v1alpha1.Receiver) error {

	if receiver == nil || receiver.ReceiverType != constant.WebhookReceiver {
		fmt.Println(receiver)
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

func (wh *WebhookExporter) DeepEqual(_ *v1alpha1.Receiver) bool {

	return false
}
