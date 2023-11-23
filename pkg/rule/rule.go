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

package rule

import (
	"bytes"
	"context"
	"fmt"
	"rule/pkg/apis/logging.whizard.io/v1alpha1"
	"rule/pkg/constant"
	"rule/pkg/utils"

	"regexp"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/kubesphere/event-rule-engine/visitor"
)

const (
	KindRule  = "rule"
	KindMacro = "macro"
	KindList  = "list"
	KindAlias = "alias"
	AuditingType = "auditing"
	EventsType = "events"
	LoggingType = "logging"
)

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
	// Whether this rule will trigger alert or not.
	alerting bool
	// Whether the audit event matched this rule will be archived or not.
	archiving bool
	labels    map[string]string
	format    string
	params    []string
}

var resourceInWorkSpace = []string{
	"devops",
	"namespaces",
	"federatednamespaces",
	"workspaceroles",
	"federatedworkspaceroles",
	"workspacemembers",
}

func (r *Rule) GetMessage(e *Event, m map[string]interface{}, rs map[string]Rule) string {

	var msg string
	if len(r.Output) == 0 {
		if len(e.Workspace) > 0 && utils.IsExist(resourceInWorkSpace, e.ObjectRef.Resource) {
			msg = fmt.Sprintf("%s %s %s '%s' in Workspace %s", e.User.Username, e.Verb, e.ObjectRef.Resource, e.ObjectRef.Name, e.Workspace)
		} else if len(e.Devops) > 0 {
			msg = fmt.Sprintf("%s %s %s '%s' in Devops %s", e.User.Username, e.Verb, e.ObjectRef.Resource, e.ObjectRef.Name, e.Devops)
		} else if len(e.ObjectRef.Namespace) > 0 && e.ObjectRef.Resource != "namespaces" && e.ObjectRef.Resource != "federatednamespaces" {
			msg = fmt.Sprintf("%s %s %s '%s' in Namespace %s", e.User.Username, e.Verb, e.ObjectRef.Resource, e.ObjectRef.Name, e.ObjectRef.Namespace)
		} else {
			msg = fmt.Sprintf("%s %s %s '%s'", e.User.Username, e.Verb, e.ObjectRef.Resource, e.ObjectRef.Name)
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
				if !ok || mr.Type != TypeAlias {
					mr, ok = rs[key]
				}
				if ok && mr.Type == TypeAlias {
					key = mr.Alias
				}
				ps = append(ps, m[key])
			}
		}
		msg = fmt.Sprintf(r.format, ps...)
	}

	return msg
}

func (r *Rule) GetCondition(rs map[string]Rule) (string, error) {

	c := r.Condition
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
			switch rule.Type {
			case KindMacro:
				c = strings.ReplaceAll(c, s, rule.Macro)
			case KindAlias:
				c = strings.ReplaceAll(c, s, rule.Alias)
			case KindList:
				buf := bytes.Buffer{}
				buf.WriteString("(")
				for index, l := range rule.List {
					buf.WriteString("\"")
					buf.WriteString(l)
					buf.WriteString("\"")
					if index != len(rule.List)-1 {
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

func (r *Rule) IsAlerting() bool {
	return r.alerting
}

func (r *Rule) SetAlerting(b bool) {
	r.alerting = b
}

func (r *Rule) IsArchiving() bool {
	return r.archiving
}

func (r *Rule) SetArchiving(b bool) {
	r.archiving = b
}

func (r *Rule) Print() map[string]interface{} {

	m, err := utils.StructToMap(r)
	if err != nil {
		return nil
	}

	if r.Type != TypeRule {
		delete(m, "enable")

	}
	return m
}

func (r *Rule) SetParams() {

	regex, err := regexp.Compile("\\${(.*?)}")
	if err != nil {
		return
	}

	op := r.Output
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

func (r *Rule) PriorityGreater(priority string) bool {
	return ordPriority(r.Priority) > ordPriority(priority)
}

func (r *Rule) PriorityGreaterOrEqual(priority string) bool {
	return ordPriority(r.Priority) >= ordPriority(priority)
}

func ordPriority(priority string) int {
	switch priority {
	case constant.Debug:
		return 1
	case constant.Info:
		return 2
	case constant.Warning:
		return 3
	default:
		return 0
	}
}

// LoadRule load rule policy from Rules.
func LoadRule(archivingPriority, alertingPriority string, alertingLabels, archivingLabels map[string]string) (map[string]Rule, error) {

	rl := &v1alpha1.RuleList{}
	if err := cache.Cache().List(context.Background(), rl); err != nil {
		return nil, err
	}

	rules := make(map[string]Rule)
	for _, item := range rl.Items {
		for _, pr := range item.Spec.PolicyRules {
			r := Rule{}
			r.labels = make(map[string]string)
			for k, v := range item.Labels {
				r.labels[k] = v
			}
			r.PolicyRule = pr
			r.Group = item.Name
			rules[fmt.Sprintf("%s.%s", r.Group, r.Name)] = r
		}
	}

	for name, r := range rules {
		if r.Type == TypeRule {
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

		if utils.IsLabelsContains(r.labels, alertingLabels) {
			r.SetArchiving(true)
			if r.PriorityGreaterOrEqual(alertingPriority) {
				r.SetAlerting(true)
			}
		}

		if utils.IsLabelsContains(r.labels, archivingLabels) &&
			r.PriorityGreaterOrEqual(archivingPriority) {
			r.SetArchiving(true)
		}

		rules[name] = r
	}

	return rules, nil
}
