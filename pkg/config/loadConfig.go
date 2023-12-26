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

package config

import (
	"context"
	"flag"
	"whizard-telemetry-ruler/pkg/apis/logging.whizard.io/v1alpha1"
	"whizard-telemetry-ruler/pkg/cache"
	"whizard-telemetry-ruler/pkg/exporter"
	"whizard-telemetry-ruler/pkg/rule"

	"sync"

	"github.com/golang/glog"
	kcache "k8s.io/client-go/tools/cache"
)

type Config struct {
	// The alert which receivers config.
	Receivers []exporter.Receiver
	// Map of rules.
	Rules map[string]rule.Rule
}

var webhookName string
var config *Config
var once sync.Once

func init() {
	flag.StringVar(&webhookName, "rule-webhook-name", "", "webhook name")
}

// LoadConfig load  config from webhook and rule crd.
func LoadConfig() error {
	//Resolve the circular dependency, assign the local variable to a global variable.
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
	})

	//load sink
	sink, err := LoadSinks()

	// Load config
	conf, err := loadConfig(sink)
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

func loadConfig(sink *exporter.Sink) (*Config, error) {

	conf := &Config{}

	if sink == nil  {
		conf.Receivers =nil
	} else {
			conf.Receivers = sink.Receivers
	}

	// Load rules
	var err error
	conf.Rules, err = rule.LoadRule()
	if err != nil {
		return nil, err
	}

	return conf, nil
}
