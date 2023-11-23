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
	"io/ioutil"
	"rule/pkg/config"
	"rule/pkg/constant"
	"rule/pkg/rule"

	"net/http"
	"sync"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	port          int
	tls           bool
	goroutinesNum int

	waitHandlerGroup sync.WaitGroup
	//eventChan        chan *rule.
	auditingchan    chan *rule.Auditing
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

	eventChan = make(chan *auditing.Event, constant.ChannelLenMax)
	go work()

	return httpServer()
}

func httpServer() error {

	container := restful.NewContainer()
	ws := new(restful.WebService)
	ws.Path("").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)
	ws.Route(ws.POST("/audit/webhook/event").To(handlerEvents))
	//  Events received through this API are only used for alerting
	ws.Route(ws.POST("/audit/webhook/event/alerting").To(handlerAlertingEvents))
	ws.Route(ws.GET("/readiness").To(readiness))
	ws.Route(ws.GET("/liveness").To(readiness))
	ws.Route(ws.GET("/preStop").To(preStop))

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

func work() {
	routinesChan := make(chan interface{}, goroutinesNum)

	for {
		event := <-eventChan
		if event == nil {
			break
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*constant.GoroutinesTimeOut)
		select {
		case routinesChan <- struct{}{}:
			cancel()
		case <-ctx.Done():
			glog.Errorf("get goroutines for audit %s timeout", event.AuditID)
			cancel()
			continue
		}

		go func() {
			stopCh := make(chan interface{}, 1)
			go func() {
				eventMatch(event)
				close(stopCh)
			}()

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*constant.GoroutinesTimeOut)
			defer cancel()
			select {
			case <-stopCh:
				break
			case <-ctx.Done():
				glog.Errorf("match audit %s timeout", event.AuditID)
			}

			<-routinesChan
		}()
	}
}

func handlerEvents(req *restful.Request, resp *restful.Response) {
	handler(req, resp, false)
}

func handlerAlertingEvents(req *restful.Request, resp *restful.Response) {
	handler(req, resp, true)
}

func handler(req *restful.Request, resp *restful.Response, alertOnly bool) {
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

	var events []*auditing.Event
	if alertOnly == false {
		events, err = auditing.NewEvent(body)
	} else {
		events, err = auditing.NewAlertEvent(body)
	}
	if err != nil {
		err := resp.WriteHeaderAndEntity(http.StatusBadRequest, "")
		if err != nil {
			glog.Errorf("response error %s", err)
		}
		return
	}

	for _, event := range events {
		if len(event.Workspace) == 0 && event.ObjectRef != nil && len(event.ObjectRef.Namespace) > 0 {
			ns := &corev1.Namespace{}
			if err := cache.Cache().Get(context.Background(), types.NamespacedName{Name: event.ObjectRef.Namespace}, ns); err == nil {
				ws, ok := ns.Labels["kubesphere.io/workspace"]
				if ok {
					event.Workspace = ws
				}
			}
		}

		event.SetAlertOnly(alertOnly)

		eventChan <- event
	}

	err = resp.WriteHeaderAndEntity(http.StatusOK, "")
	if err != nil {
		glog.Errorf("response error %s", err)
	}
}

func Close() {
	waitHandlerGroup.Wait()
	glog.Errorf("msg handler close, wait pool close")
	close(eventChan)
}

// readiness
func readiness(_ *restful.Request, resp *restful.Response) {

	responseWithHeaderAndEntity(resp, http.StatusOK, "")
}

// preStop
func preStop(_ *restful.Request, resp *restful.Response) {

	Close()
	responseWithHeaderAndEntity(resp, http.StatusOK, "")
	glog.Flush()
}

func responseWithHeaderAndEntity(resp *restful.Response, status int, value interface{}) {
	e := resp.WriteHeaderAndEntity(status, value)
	if e != nil {
		glog.Errorf("response error %s", e)
	}
}
