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

package constant

const (
	KeyFile  = "/etc/kube/rule/tls.key"
	CertFile = "/etc/kube/rule/tls.crt"
)

const (
	ChannelLenMax     = 1000
	GoroutinesNumMax  = 10
	GoroutinesTimeOut = 5
)

const (
	Info     = "INFO"
	Warning  = "WARNING"
	ERROR    = "ERROR"
	CRITICAL = "CRITICAL"
)

const (
	Event    = "Event"
	Logging  = "Logging"
	Auditing = "Auditing"
)

const (
	WhizardTelemetryRuler      = "whizard-telemetry-ruler"
	WebhookReceiver            = "webhook"
	AlertManagerReciver        = "alertmanager"
	NotificationManagerReciver = "notificationmanager"
)

const (
	DefaultNamespace = "kubesphere-logging-system"
)
