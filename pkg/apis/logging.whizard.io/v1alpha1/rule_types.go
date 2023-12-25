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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type Expr struct {
	// Rule kind, rule, macro,list,alias.
	Kind string `json:"kind,omitempty"`
	// Rule condition
	// This effective When the rule kind is rule.
	Condition string `json:"condition,omitempty"`
	// This effective When the rule kind is macro.
	Macro string `json:"macro,omitempty"`
	// This effective When the rule kind is alias.
	Alias string `json:"alias,omitempty"`
	// This effective When the rule kind is list.
	List []string `json:"list,omitempty"`
}

type Alerts struct {
	// Values of Annotations can use format string with the fields of the event.
	Annotations map[string]string `json:"annotations,omitempty"`
	// The output formatter of message which send to user.
	Message string `json:"message,omitempty"`
	// Rule priority, INFO,WARNING,ERROR,CRITICAL.
	Severity string `json:"severity,omitempty"`
}

type Rule struct {
	// Rule name.
	Name string `json:"name,omitempty"`

	// Rule describe.
	Desc string `json:"desc,omitempty"`
	// Expression of the rule
	Expr   Expr   `json:"expr,omitempty"`
	Alerts Alerts `json:"alerts,omitempty"`
	// Is the rule enable.
	Enable bool `json:"enable,omitempty"`
}

// RuleSpec defines the desired state of ClusterRuleGroup.
type ClusterRuleGroupRuleSpec struct {
	// whizard log type ,auditing/events/logging
	Type  string `json:"type,omitempty"`
	Rules []Rule `json:"rules,omitempty"`
}

// RuleStatus defines the observed state of ClusterRuleGroup.
type ClusterRuleGroupRuleStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=crg

// ClusterRuleGroup is the Schema for the rules API
type ClusterRuleGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterRuleGroupRuleSpec   `json:"spec,omitempty"`
	Status ClusterRuleGroupRuleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RuleList contains a list of Rule
type ClusterRuleGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterRuleGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterRuleGroup{}, &ClusterRuleGroupList{})
}
