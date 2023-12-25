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
	"fmt"
	"github.com/golang/glog"
	"sigs.k8s.io/yaml"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os"
	kubeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"whizard-telemetry-ruler/pkg/constant"
	"whizard-telemetry-ruler/pkg/exporter"
	"whizard-telemetry-ruler/pkg/utils"
)

func LoadSinks() (*exporter.Sink, error) {

	// Load Kubernetes config
	k8sConfig, err := kubeconfig.GetConfig()
	if err != nil {
		fmt.Printf("Error loading kubeconfig: %v\n", err)
		return nil, err
	}

	// Create Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		fmt.Printf("Error creating Kubernetes clientset: %v\n", err)
		return nil, err
	}

	ns := os.Getenv("NAMESPACE")
	if len(ns) == 0 {
		ns = constant.DefaultNamespace
	}

	configmap, err := clientset.CoreV1().ConfigMaps(ns).Get(context.Background(), "whizard-telemetry-ruler", metav1.GetOptions{})
	data, ok := configmap.Data["config"]
	if !ok {
		glog.Errorf("Failed to get configmap : %v", err)
		return nil, fmt.Errorf("failed to get configmap")
	}

	if data == "" {
		fmt.Printf("receiver is empty, please check configmap")
		return nil, fmt.Errorf("failed to get configmap")
	}
	fmt.Println("xxxxxxxxxxx")
	fmt.Println(data)

	var sink *exporter.Sink
	err = yaml.Unmarshal([]byte(data), &sink)
	if err != nil {
		glog.Errorf("json decode failed : %v", err)
		fmt.Println("-----------")
		fmt.Println(err)
		return nil, fmt.Errorf("failed to get configmap")
	}
	fmt.Println(utils.ToJsonString(sink.Receivers))

	return sink, err
}
