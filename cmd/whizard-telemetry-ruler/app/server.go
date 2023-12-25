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
	"context"
	"flag"
	"fmt"
	"github.com/emicklei/go-restful"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/types"
	"whizard-telemetry-ruler/pkg/cache"
	"whizard-telemetry-ruler/pkg/config"
	"whizard-telemetry-ruler/pkg/constant"
	"whizard-telemetry-ruler/pkg/rule"

	"net/http"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
)

var (
	port                  int
	tls                   bool
	goroutinesNum         int

	waitHandlerGroup sync.WaitGroup
	whizardChan      chan *rule.WhizardEvent
)

func AddFlags(fs *pflag.FlagSet) {
	fs.IntVar(&port, "port", 8080, "The port which the server listen, default 8080")
	fs.BoolVar(&tls, "tls", true, "Use https, default false")
	fs.IntVar(&goroutinesNum, "goroutines-num", constant.GoroutinesNumMax, "the num of goroutine to match rule,default 200")
}

func NewServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "rule-webhook",
		Long: `The rule webhook to receive audit/event/log`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return Run()
		},
	}
	AddFlags(cmd.Flags())
	cmd.Flags().AddGoFlagSet(flag.CommandLine)

	return cmd
}

func Run() error {

	pflag.VisitAll(func(flag *pflag.Flag) {
		glog.Errorf("FLAG: --%s=%q", flag.Name, flag.Value)
	})

	if err := config.LoadConfig(); err != nil {
		glog.Fatal(err)
	}

	glog.Info("Run start")
	whizardChan = make(chan *rule.WhizardEvent, constant.ChannelLenMax)

	go whizardEventsWorker()

	glog.Info("Run function completed")
	return httpServer()
}

func httpServer() error {

	container := restful.NewContainer()
	ws := new(restful.WebService)
	ws.Path("").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)
	ws.Route(ws.POST("/webhook/auditing").To(handlerAudits))
	ws.Route(ws.POST("/webhook/events").To(handlerEvents))
	ws.Route(ws.GET("/readiness").To(readiness))
	ws.Route(ws.GET("/liveness").To(readiness))
	ws.Route(ws.GET("/prestop").To(preStop))

	container.Add(ws)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: container,
	}

	var err error
	if tls {
		err = server.ListenAndServeTLS(constant.CertFile, constant.KeyFile)
	} else {
		err = server.ListenAndServe()
	}

	return err
}

func handlerEvents(request *restful.Request, response *restful.Response) {
	waitHandlerGroup.Add(1)
	defer waitHandlerGroup.Done()

	body, err := ioutil.ReadAll(request.Request.Body)
	if err != nil {
		err := response.WriteHeaderAndEntity(http.StatusBadRequest, "")
		if err != nil {
			glog.Errorf("response error %s", err)
		}
		return
	}
	var events []*rule.Event
	events, err = rule.NewEvents(body)
	if err != nil {
		err := response.WriteHeaderAndEntity(http.StatusBadRequest, "")
		if err != nil {
			glog.Errorf("response error %s", err)
		}
		return
	}

	for _, event := range events {
		whizardEvent := &rule.WhizardEvent{
			Kind:  constant.Event,
			Event: event,
		}
		whizardChan <- whizardEvent
	}

	err = response.WriteHeaderAndEntity(http.StatusOK, "")
	if err != nil {
		glog.Errorf("response error %s", err)
	}
}

func whizardEventsWorker() {
	glog.Info("Entering whizardEventsWorker")
	routinesChan := make(chan interface{}, goroutinesNum)
	for {
		whizardEvents := <-whizardChan
		if whizardEvents == nil {
			break
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*constant.GoroutinesTimeOut)
		select {
		case routinesChan <- struct{}{}:
			cancel()
		case <-ctx.Done():
			glog.Errorf("get goroutines for  %s timeout", whizardEvents.Kind)
			cancel()
			continue
		}

		go func() {
			stopCh := make(chan interface{}, 1)
			go func() {
				if whizardEvents.Kind == constant.Auditing {
					processAuditingEvent(whizardEvents.Auditing)
					close(stopCh)
				} else if whizardEvents.Kind == constant.Event {
					processKubeEvent(whizardEvents.Event)
					close(stopCh)
				}
			}()

			ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second*constant.GoroutinesTimeOut)
			defer cancel2()
			select {
			case <-stopCh:
				break
			case <-ctx2.Done():
				glog.Errorf("match %s timeout", whizardEvents.Kind)
			}

			<-routinesChan
		}()
	}
}

func handlerAudits(req *restful.Request, resp *restful.Response) {
	waitHandlerGroup.Add(1)
	defer waitHandlerGroup.Done()

	body, err := ioutil.ReadAll(req.Request.Body)
	if err != nil {
		err := resp.WriteHeaderAndEntity(http.StatusBadRequest, "")
		if err != nil {
			glog.Errorf("response error %s", err)
		}
		return
	}
	var audits []*rule.Auditing

	audits, err = rule.NewAuditing(body)

	if err != nil {
		err := resp.WriteHeaderAndEntity(http.StatusBadRequest, "")
		if err != nil {
			glog.Errorf("response error %s", err)
		}
		return
	}

	// Iterate through audits, check and populate missing Workspace information based on namespace labels.
	for _, audit := range audits {
		if len(audit.Workspace) == 0 && audit.ObjectRef != nil && len(audit.ObjectRef.Namespace) > 0 {
			ns := &corev1.Namespace{}
			if err := cache.Cache().Get(context.Background(), types.NamespacedName{Name: audit.ObjectRef.Namespace}, ns); err == nil {
				ws, ok := ns.Labels["kubesphere.io/workspace"]
				if ok {
					audit.Workspace = ws
				}
			}
		}
		whizardAudit := &rule.WhizardEvent{
			Kind:     constant.Auditing,
			Auditing: audit,
		}
		whizardChan <- whizardAudit
	}

	err = resp.WriteHeaderAndEntity(http.StatusOK, "")
	if err != nil {
		glog.Errorf("response error %s", err)
	}
}

func Close() {
	waitHandlerGroup.Wait()
	glog.Errorf("msg handler close, wait pool close")
	close(whizardChan)
}

// preStop
func preStop(_ *restful.Request, resp *restful.Response) {

	Close()
	responseWithHeaderAndEntity(resp, http.StatusOK, "")
	glog.Flush()
}

// readiness
func readiness(_ *restful.Request, resp *restful.Response) {

	responseWithHeaderAndEntity(resp, http.StatusOK, "")
}

func responseWithHeaderAndEntity(resp *restful.Response, status int, value interface{}) {
	e := resp.WriteHeaderAndEntity(status, value)
	if e != nil {
		glog.Errorf("response error %s", e)
	}
}
