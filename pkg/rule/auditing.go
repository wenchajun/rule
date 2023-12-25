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
	"whizard-telemetry-ruler/pkg/utils"

	"strings"

	"github.com/golang/glog"
	"k8s.io/apiserver/pkg/apis/audit"
)

type Auditing struct {
	// audit event.
	audit.Event
	// Devops project.
	Devops string
	// The workspace which this audit event happened.
	Workspace string
	// The message send to user,formatted by th rule output.
	Message string
	// name of rule which triggered alert.
	alertRuleName string
	//custom message
	Annotations map[string]string
}

func NewAuditing(data []byte) ([]*Auditing, error) {

	var auditingList []*Auditing

	err := json.Unmarshal(data, &auditingList)
	if err != nil {
		glog.Errorf("unmarshal failed with:%v,body is: %s", err, string(data))
		return nil, err
	}

	var es []*Auditing
	for _, event := range auditingList {
		e := event
		e.Verb = strings.ToLower(e.Verb)
		es = append(es, e)
	}

	return es, nil
}

func (a *Auditing) ToString() string {

	m, err := utils.StructToMap(a)
	if err != nil {
		return ""
	}

	req := ""
	if a.RequestObject != nil {
		req, err = utils.ToJsonString(a.RequestObject)
		if err != nil {
			glog.Error(err)
			req = ""
		}
	}
	m["RequestObject"] = req

	resp := ""
	if a.ResponseObject != nil {
		resp, err = utils.ToJsonString(a.ResponseObject)
		if err != nil {
			glog.Error(err)
			resp = ""
		}
	}
	m["ResponseObject"] = resp

	ip := ""
	if a.SourceIPs != nil && len(a.SourceIPs) > 0 {
		ip = utils.SliceToString(a.SourceIPs, ",")
	}

	m["SourceIPs"] = ip

	u := make(map[string]interface{})
	u["Username"] = a.User.Username
	u["UID"] = a.User.UID
	u["Groups"] = utils.SliceToString(a.User.Groups, ",")
	m["User"] = u
	delete(m, "ImpersonatedUser")
	delete(m, "Annotations")
	delete(m, "UserAgent")

	delete(m, "RequestReceivedTimestamp")
	delete(m, "StageTimestamp")
	m["RequestReceivedTimestamp"] = a.RequestReceivedTimestamp.Local()
	m["StageTimestamp"] = a.StageTimestamp.Local()

	s, err := utils.ToJsonString(m)
	if err != nil {
		glog.Error(err)
		return ""
	}

	return s
}

func (a *Auditing) GetAlertRuleName() string {
	return a.alertRuleName
}

func (a *Auditing) SetAlertRuleName(n string) {
	a.alertRuleName = n
}
