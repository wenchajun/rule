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

package constant

const (
	KeyFile  = "/etc/kube/auditing/tls.key"
	CertFile = "/etc/kube/auditing/tls.crt"
)

const (
	ChannelLenMax     = 10000
	GoroutinesNumMax  = 10
	GoroutinesTimeOut = 5
)

const (
	Warning = "WARNING"
	Info    = "INFO"
	Debug   = "DEBUG"
)

const (
	WebhookReceiver            = "webhook"
	AlertManagerReciver        = "alertmanager"
	NotificationManagerReciver = "notificationmanager"
)

const (
	DefaultNamespace      = "kubesphere-logging-system"
	DefaultWebhookImage   = "kubespheredev/kube-auditing-webhook:latest"
	DefaultWebhook        = "kube-auditing-webhook"
	DefaultServiceAccount = "kube-auditing-operator"
)

const (
	Alerting  = "alerting"
	Archiving = "archiving"
)

const (
	FluentBitLogLenMax = 16384
)
