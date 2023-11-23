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

package config

import (
	"context"
	"flag"
	"rule/pkg/apis/logging.whizard.io/v1alpha1"
	"rule/pkg/cache"
	"rule/pkg/constant"
	"rule/pkg/exporter"
	"rule/pkg/rule"

	"sync"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/types"
	kcache "k8s.io/client-go/tools/cache"
)

type Config struct {
	// The alert which receivers config.
	Receivers []v1alpha1.Receiver
	// The labels to select the rule of archiving.
	Archiving map[string]string
	// The labels to select the rule of alerting.
	Alerting          map[string]string
	ArchivingPriority string
	AlertingPriority  string
	// Map of rules.
	Rules map[string]rule.Rule
}

var webhookName string
var config *Config
var once sync.Once

func init() {
	flag.StringVar(&webhookName, "auditing-webhook-name", "", "webhook name")
}

// LoadConfig load  config from webhook and rule crd.
func LoadConfig() error {

	once.Do(func() {
		// Add event handler, to reload config when crd change
		ruleInf, err := cache.Cache().GetInformer(context.Background(), &v1alpha1.ClusterRuleGroup{})
		if err != nil {
			glog.Fatal(err)
		}
		ruleInf.AddEventHandler(kcache.ResourceEventHandlerFuncs{
			AddFunc: onChange,
			UpdateFunc: func(oldObj, newObj interface{}) {
				onChange(newObj)
			},
			DeleteFunc: onChange,
		})

		webhookInf, err := cache.Cache().GetInformer(context.Background(), &v1alpha1.Webhook{})
		if err != nil {
			glog.Fatal(err)
		}
		webhookInf.AddEventHandler(kcache.ResourceEventHandlerFuncs{
			AddFunc: onChange,
			UpdateFunc: func(oldObj, newObj interface{}) {
				onChange(newObj)
			},
			DeleteFunc: onChange,
		})
	})

	// Load config
	conf, err := loadConfig()
	if err != nil {
		return err
	}

	config = conf

	// Init the alert receivers.
	// The receiver must determine whether it will reconnect or not when config reload.
	errs := exporter.Connect(conf.Receivers)
	if errs != nil && len(errs) > 0 {
		glog.Errorf("init receivers err")
		for _, e := range errs {
			glog.Error(e)
		}
	}
	//fmt.Println(utils.OutputAsJson(config))
	return nil
}

func GetConfig() *Config {
	return config
}

func onChange(_ interface{}) {
	// On crd change, reload config
	if err := LoadConfig(); err != nil {
		glog.Errorf("reload config error, %s", err)
	}
	glog.Errorf("reload config")
}

func loadConfig() (*Config, error) {

	wh := &v1alpha1.Webhook{}
	if err := cache.Cache().Get(context.Background(), types.NamespacedName{Name: webhookName}, wh); err != nil {
		return nil, err
	}

	conf := &Config{}
	conf.ArchivingPriority = wh.Spec.ArchivingPriority
	if conf.ArchivingPriority != constant.Debug &&
		conf.ArchivingPriority != constant.Info &&
		conf.ArchivingPriority != constant.Warning {
		conf.ArchivingPriority = constant.Debug
	}

	conf.AlertingPriority = wh.Spec.AlertingPriority
	if conf.AlertingPriority != constant.Debug &&
		conf.AlertingPriority != constant.Info &&
		conf.AlertingPriority != constant.Warning {
		conf.AlertingPriority = constant.Warning
	}

	conf.Receivers = wh.Spec.Receivers

	if wh.Spec.Sinks != nil &&
		wh.Spec.Sinks.AlertingRuleSelector != nil {
		conf.Alerting = wh.Spec.Sinks.AlertingRuleSelector.MatchLabels
	}

	// If the labels of alerting not set, use default
	if conf.Alerting == nil || len(conf.Alerting) == 0 {
		conf.Alerting = map[string]string{}
		conf.Alerting["type"] = constant.Alerting
	}

	if wh.Spec.Sinks != nil &&
		wh.Spec.Sinks.ArchivingRuleSelector != nil {
		conf.Archiving = wh.Spec.Sinks.ArchivingRuleSelector.MatchLabels
	}

	// If the labels of archiving not set, use default
	if conf.Archiving == nil || len(conf.Archiving) == 0 {
		conf.Archiving = map[string]string{}
		conf.Archiving["type"] = constant.Archiving
	}


	// Load rules
	var err error
	conf.Rules, err = rule.LoadRule(conf.ArchivingPriority, conf.AlertingPriority, conf.Alerting, conf.Archiving)
	if err != nil {
		return nil, err
	}

	return conf, nil
}
