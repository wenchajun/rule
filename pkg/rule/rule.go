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
	"bytes"
	"context"
	"fmt"
	"whizard-telemetry-ruler/pkg/apis/logging.whizard.io/v1alpha1"
	"whizard-telemetry-ruler/pkg/cache"
	"whizard-telemetry-ruler/pkg/constant"
	"whizard-telemetry-ruler/pkg/utils"

	"regexp"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/kubesphere/event-rule-engine/visitor"
)

const (
	KindRule     = "rule"
	KindMacro    = "macro"
	KindList     = "list"
	KindAlias    = "alias"
	AuditingType = "auditing"
	EventsType   = "events"
	LoggingType  = "logging"
)

type WhizardEvent struct {
	Kind     string
	Event    *Event
	Auditing *Auditing
}

type Group struct {
	// Group name, also the name of the instance of CRD Rule which this rule in.
	Name string
	// Group type, archiving or alerting.
	Type string
}

type Rule struct {
	// The name of group which this rule in.
	Group string
	v1alpha1.Rule
	labels      map[string]string
	whizardEventType string
	format      string
	params      []string
}

var resourceInWorkSpace = []string{
	"devops",
	"namespaces",
	"federatednamespaces",
	"workspaceroles",
	"federatedworkspaceroles",
	"workspacemembers",
}

func (r *Rule) GetAuditingAlertMessage(a *Auditing, m map[string]interface{}, rs map[string]Rule) (string, map[string]string) {

	var msg string
	if len(r.Rule.Alerts.Message) == 0 {
		if len(a.Workspace) > 0 && utils.IsExist(resourceInWorkSpace, a.ObjectRef.Resource) {
			msg = fmt.Sprintf("%s %s %s '%s' in Workspace %s", a.User.Username, a.Verb, a.ObjectRef.Resource, a.ObjectRef.Name, a.Workspace)
		} else if len(a.Devops) > 0 {
			msg = fmt.Sprintf("%s %s %s '%s' in Devops %s", a.User.Username, a.Verb, a.ObjectRef.Resource, a.ObjectRef.Name, a.Devops)
		} else if len(a.ObjectRef.Namespace) > 0 && a.ObjectRef.Resource != "namespaces" && a.ObjectRef.Resource != "federatednamespaces" {
			msg = fmt.Sprintf("%s %s %s '%s' in Namespace %s", a.User.Username, a.Verb, a.ObjectRef.Resource, a.ObjectRef.Name, a.ObjectRef.Namespace)
		} else {
			msg = fmt.Sprintf("%s %s %s '%s'", a.User.Username, a.Verb, a.ObjectRef.Resource, a.ObjectRef.Name)
		}
	} else {
		var ps []interface{}
		for _, p := range r.params {
			if strings.HasPrefix(p, "$") {
				index, err := strconv.Atoi(p[1:])
				if err != nil {
					glog.Error(err)
					ps = append(ps, "")
				}

				ss := strings.Split(m["ObjectRef.Name"].(string), ":")
				if index-1 < len(ss) {
					ps = append(ps, ss[index-1])
				} else {
					ps = append(ps, "")
				}
			} else {
				key := p
				mr, ok := rs[fmt.Sprintf("%s.%s", r.Group, p)]
				if !ok || mr.Expr.Kind != KindAlias {
					mr, ok = rs[key]
				}
				if ok && mr.Expr.Kind == KindAlias {
					key = mr.Expr.Alias
				}
				ps = append(ps, m[key])
			}
		}
		msg = fmt.Sprintf(r.format, ps...)
	}

	an := make(map[string]string)
	for k, v := range r.Alerts.Annotations {
		an[k] = v
	}
	return msg, an
}

func (r *Rule) GetEventAlertMessage(e *Event, m map[string]interface{}, rs map[string]Rule) (string, map[string]string) {

	var msg string
	if len(r.Alerts.Message) == 0 {
		msg = fmt.Sprintf("%s'", e.Event.Message)
	} else {
		var ps []interface{}
		for _, p := range r.params {
			if strings.HasPrefix(p, "$") {
				index, err := strconv.Atoi(p[1:])
				if err != nil {
					glog.Error(err)
					ps = append(ps, "")
				}

				ss := strings.Split(m["involvedObject.Name"].(string), ":")
				if index-1 < len(ss) {
					ps = append(ps, ss[index-1])
				} else {
					ps = append(ps, "")
				}
			} else {
				key := p
				mr, ok := rs[fmt.Sprintf("%s.%s", r.Group, p)]
				if !ok || mr.Expr.Kind != KindAlias {
					mr, ok = rs[key]
				}
				if ok && mr.Expr.Kind == KindAlias {
					key = mr.Expr.Alias
				}
				ps = append(ps, m[key])
			}
		}
		msg = fmt.Sprintf(r.format, ps...)
	}

	an := make(map[string]string)
	for k, v := range r.Alerts.Annotations {
		an[k] = v
	}
	return msg, an
}

func (r *Rule) GetCondition(rs map[string]Rule) (string, error) {

	c := r.Rule.Expr.Condition
	regex, err := regexp.Compile("\\${(.*?)}")
	if err != nil {
		return c, err
	}

	ss := regex.FindAllString(c, -1)
	if ss == nil || len(ss) == 0 {
		return c, nil
	}

	for _, s := range ss {
		key := strings.TrimPrefix(s, "${")
		key = strings.TrimSuffix(key, "}")
		rule, ok := rs[fmt.Sprintf("%s.%s", r.Group, key)]
		if !ok {
			rule, ok = rs[key]
		}
		if ok {
			switch rule.Expr.Kind {
			case KindMacro:
				c = strings.ReplaceAll(c, s, rule.Expr.Macro)
			case KindAlias:
				c = strings.ReplaceAll(c, s, rule.Expr.Alias)
			case KindList:
				buf := bytes.Buffer{}
				buf.WriteString("(")
				for index, l := range rule.Expr.List {
					buf.WriteString("\"")
					buf.WriteString(l)
					buf.WriteString("\"")
					if index != len(rule.Expr.List)-1 {
						buf.WriteString(",")
					}
				}
				buf.WriteString(")")
				c = strings.ReplaceAll(c, s, buf.String())
			}
		} else {
			return c, fmt.Errorf("rule %s not correct, %s is not found", r.Name, key)
		}
	}

	return c, nil
}

func (r *Rule) Print() map[string]interface{} {

	m, err := utils.StructToMap(r)
	if err != nil {
		return nil
	}

	if r.Expr.Kind != KindRule {
		delete(m, "enable")

	}
	return m
}

func (r *Rule) SetParams() {

	regex, err := regexp.Compile("\\${(.*?)}")
	if err != nil {
		return
	}

	op := r.Alerts.Message
	if len(op) == 0 {
		return
	}

	ss := regex.FindAllString(op, -1)
	if ss == nil || len(ss) == 0 {
		r.format = op
		return
	}

	var ps []string
	for _, s := range ss {
		op = strings.ReplaceAll(op, s, "%s")
		s = strings.TrimPrefix(s, "${")
		s = strings.TrimSuffix(s, "}")
		ps = append(ps, s)
	}

	r.format = op
	r.params = ps
}

func (r *Rule) SeverityHigherThan(severity string) bool {
	return ordPriority(r.Alerts.Severity) > ordPriority(severity)
}

func (r *Rule) SeverityHigherOrEqualTo (severity string) bool {
	return ordPriority(r.Alerts.Severity) >= ordPriority(severity)
}

func ordPriority(priority string) int {
	switch priority {
	case constant.Info:
		return 1
	case constant.Warning:
		return 2
	case constant.ERROR:
		return 3
	case constant.CRITICAL:
		return 4
	default:
		return 0
	}
}

func (r *Rule) GetEventType() string {
	return r.whizardEventType
}


// LoadRule load rule policy from Rules.
func LoadRule() (map[string]Rule, error) {

	rl := &v1alpha1.ClusterRuleGroupList{}
	if err := cache.Cache().List(context.Background(), rl); err != nil {
		return nil, err
	}

	rules := make(map[string]Rule)
	for _, item := range rl.Items {
		outputType := item.Spec.Type
		for _, pr := range item.Spec.Rules {
			r := Rule{}
			r.Rule = pr
			r.whizardEventType = outputType
			r.Group = item.Name
			rules[fmt.Sprintf("%s.%s", r.Group, r.Name)] = r
		}
	}

	for name, r := range rules {
		if r.Expr.Kind == KindRule {
			// If the condition of item is incorrect, delete this item.
			c, err := r.GetCondition(rules)
			if err != nil {
				glog.Error(err)
				delete(rules, name)
				continue
			}

			// If the condition of item is not grammatical, delete this item.
			if ok, err := visitor.CheckRule(c); !ok {
				glog.Errorf("item %s is not correct, conditions(%s), err(%s)", name, c, err)
				delete(rules, name)
			}
		}

		r.SetParams()
		rules[name] = r
	}

	return rules, nil
}
