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

package rule

import (
	"encoding/json"
	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	"whizard-telemetry-ruler/pkg/utils"
)

type Event struct {
	Event *corev1.Event
	// The message send to user,formatted by th rule output.
	Message string
	// The workspace which this audit event happened.
	Workspace string
	//custom message
	Annotations map[string]string
	// name of rule which triggered alert.
	alertRuleName string
}

func NewEvents(data []byte) ([]*Event, error) {

	var eventList []*Event

	err := json.Unmarshal(data, &eventList)
	if err != nil {
		glog.Errorf("unmarshal failed with:%v,body is: %s", err, string(data))
		return nil, err
	}

	return eventList, nil
}

func (e *Event) ToString() string {

	s, err := utils.ToJsonString(e)
	if err != nil {
		glog.Error(err)
		return ""
	}

	return s
}

func (e *Event) GetAlertRuleName() string {
	return e.alertRuleName
}

func (e *Event) SetAlertRuleName(n string) {
	e.alertRuleName = n
}
