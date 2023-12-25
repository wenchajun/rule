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

package app

import (
	"whizard-telemetry-ruler/pkg/config"
	"whizard-telemetry-ruler/pkg/exporter"
	"whizard-telemetry-ruler/pkg/rule"
	"whizard-telemetry-ruler/pkg/utils"

	"github.com/golang/glog"
	"github.com/kubesphere/event-rule-engine/visitor"
)

func processAuditingEvent (a *rule.Auditing) {

	err := evaluateAuditingRule(a)
	if err != nil {
		glog.Errorf("match rule error %s", err)
		return
	}

	if len(a.Message) > 0 {
		go exporter.ExportAuditingAlerts(a)
	}

}

func processKubeEvent(e *rule.Event) {
	err := evaluateKubeEventRule(e)
	if err != nil {
		glog.Errorf("match rule error %s", err)
		return
	}

	if len(e.Message) > 0 {
		go exporter.ExportEventAlerts(e)
	}

}

func evaluateAuditingRule(a *rule.Auditing) error {

	auditingMap, err := utils.StructToMap(a.Event)
	if err != nil {
		return err
	}

	flattenAuditing := utils.Flatten(auditingMap)
	rs := config.GetConfig().Rules
	severity := ""
	for _, r := range rs {
		if !r.Rule.Enable || r.Rule.Expr.Kind != rule.KindRule || r.GetEventType() != rule.AuditingType {
			continue
		}

		if !r.SeverityHigherThan(severity)  {
			continue
		}

		c, _ := r.GetCondition(rs)
		err, ok := visitor.EventRuleEvaluate(flattenAuditing, c)
		if err != nil {
			glog.Errorf("match rule[%s] error %s", r.Name, err)
			continue
		}

		if ok {
			// When the event matched multiple rules, the message is generated by the rule with the highest priority
			a.Message, a.Annotations = r.GetAuditingAlertMessage(a, flattenAuditing, rs)
			a.SetAlertRuleName(r.Name)
			severity = r.Alerts.Severity
		}
	}

	return nil
}

func evaluateKubeEventRule(e *rule.Event) error {

	m, err := utils.StructToMap(e.Event)
	if err != nil {
		return err
	}

	fm := utils.Flatten(m)
	rs := config.GetConfig().Rules
	severity := ""
	for _, r := range rs {
		if !r.Enable || r.Rule.Expr.Kind != rule.KindRule || r.GetEventType() != rule.EventsType {
			continue
		}

		if !r.SeverityHigherOrEqualTo(severity) {
			continue
		}

		c, _ := r.GetCondition(rs)
		err, ok := visitor.EventRuleEvaluate(fm, c)
		if err != nil {
			glog.Errorf("match rule[%s] error %s", r.Name, err)
			continue
		}

		if ok {
			// When the event matched multiple rules, the message is generated by the rule with the highest priority
			e.Message, e.Annotations = r.GetEventAlertMessage(e, fm, rs)
			e.SetAlertRuleName(r.Name)
			severity = r.Alerts.Severity
		}
	}

	return nil
}