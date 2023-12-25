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

package cache

import (
	"context"
	"sync"
	"whizard-telemetry-ruler/pkg/apis/logging.whizard.io/v1alpha1"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	once          sync.Once
	cacheInformer cache.Cache
)

func init() {
	once.Do(doOnce)
}

func doOnce() {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	k8sConfig, err := config.GetConfig()
	if err != nil {
		glog.Fatalln(err)
	}

	cacheInformer, err = cache.New(k8sConfig, cache.Options{
		Scheme: scheme,
	})
	if err != nil {
		glog.Fatalln(err)
	}

	go func() {
		err := cacheInformer.Start(context.Background())
		if err != nil {
			glog.Fatalln(err)
		}
	}()
}

func Cache() cache.Cache {
	return cacheInformer
}
