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

package controllers

import (
	"context"
	"fmt"
	"os"
	"rule/pkg/apis/logging.whizard.io/v1alpha1"
	"rule/pkg/constant"
	"rule/pkg/utils"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// WebhookReconciler reconciles a Webhook object
type WebhookReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=logging.whizard.io,resources=webhooks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=logging.whizard.io,resources=webhooks/status,verbs=get;update;patch

func (r *WebhookReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("Webhook", req.NamespacedName)
	log.Info("Reconciling Webhook")

	// Only one webhook will create related deployment.

	ns := os.Getenv("NAMESPACE")
	if len(ns) == 0 {
		ns = constant.DefaultNamespace
	}

	// Fetch the Webhook instance
	webhook := &v1alpha1.Webhook{}
	err := r.Get(ctx, req.NamespacedName, webhook)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue

			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Fetch the secret associated to Webhook.
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-secret", webhook.Name),
			Namespace: ns,
		},
	}
	err = r.Get(ctx,
		types.NamespacedName{
			Name:      fmt.Sprintf("%s-secret", webhook.Name),
			Namespace: ns,
		},
		secret)
	if err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		err = controllerutil.SetControllerReference(webhook, secret, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.CreateSecret(ctx, secret, webhook.Name, ns)
		if err != nil {
			return ctrl.Result{}, err
		}
		log.Info("create secret", fmt.Sprintf("%s.%s", secret.Name, secret.Namespace), controllerutil.OperationResultCreated)
	} else {
		log.Info("", fmt.Sprintf("%s.%s", secret.Name, secret.Namespace), controllerutil.OperationResultNone)
	}

	// Fetch the deployment associated to Webhook.
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-deploy", webhook.Name),
			Namespace: ns,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, deploy,
		r.deployMutate(webhook, deploy))
	if err != nil {
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}
	log.Info("create webhook", fmt.Sprintf("%s.%s", deploy.Name, deploy.Namespace), op)

	// Fetch the service associated to Webhook.
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-svc", webhook.Name),
			Namespace: ns,
		},
	}

	op, err = controllerutil.CreateOrUpdate(ctx, r.Client, svc,
		r.svcMutate(webhook, svc))
	if err != nil {
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}
	log.Info("", fmt.Sprintf("%s.%s", svc.Name, svc.Namespace), op)

	return ctrl.Result{}, nil
}

func (r *WebhookReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Webhook{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

func (r *WebhookReconciler) deployMutate(webhook *v1alpha1.Webhook,
	deploy *appsv1.Deployment) controllerutil.MutateFn {

	return func() error {
		deploy.Labels = webhook.Labels
		deploy.Spec.Replicas = webhook.Spec.Replicas
		deploy.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app": webhook.Name,
			},
		}
		deploy.Spec.Template.ObjectMeta = metav1.ObjectMeta{
			Labels: map[string]string{
				"app": webhook.Name,
			},
		}
		deploy.Spec.Template.Spec.ImagePullSecrets = webhook.Spec.ImagePullSecrets
		deploy.Spec.Template.Spec.ServiceAccountName = constant.DefaultServiceAccount
		if deploy.Spec.Template.Spec.Containers == nil {
			deploy.Spec.Template.Spec.Containers = make([]corev1.Container, 1)
		} else if len(deploy.Spec.Template.Spec.Containers) > 1 {
			deploy.Spec.Template.Spec.Containers = deploy.Spec.Template.Spec.Containers[0:1]
		}
		deploy.Spec.Template.Spec.Containers[0].Name = webhook.Name
		deploy.Spec.Template.Spec.Containers[0].Image = webhook.Spec.Image
		if len(deploy.Spec.Template.Spec.Containers[0].Image) == 0 {
			deploy.Spec.Template.Spec.Containers[0].Image = constant.DefaultWebhookImage
		}
		deploy.Spec.Template.Spec.Containers[0].ImagePullPolicy = webhook.Spec.ImagePullPolicy


		port := webhook.Spec.Port
		if port == 0 {
			if webhook.Spec.UseHTTPS != nil && *webhook.Spec.UseHTTPS {
				port = 6443
			} else {
				port = 8080
			}
		}
		deploy.Spec.Template.Spec.Containers[0].Args = webhook.Spec.Args
		deploy.Spec.Template.Spec.Containers[0].Args = append(deploy.Spec.Template.Spec.Containers[0].Args,
			fmt.Sprintf("--port=%d", port))

		if webhook.Spec.UseHTTPS != nil {
			deploy.Spec.Template.Spec.Containers[0].Args = append(deploy.Spec.Template.Spec.Containers[0].Args,
				fmt.Sprintf("--tls=%v", *webhook.Spec.UseHTTPS))
		}

		deploy.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
			{
				Name:      "cert",
				MountPath: "/etc/kube/auditing/",
				ReadOnly:  true,
			},
			{
				Name:      "host-time",
				MountPath: "/etc/localtime",
				ReadOnly:  true,
			},
		}

		scheme := corev1.URISchemeHTTP
		if webhook.Spec.UseHTTPS != nil && *webhook.Spec.UseHTTPS == true {
			scheme = corev1.URISchemeHTTPS

		}
		deploy.Spec.Template.Spec.Containers[0].ReadinessProbe = &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/readiness",
					Port:   intstr.IntOrString{IntVal: port},
					Scheme: scheme,
				},
			},
			InitialDelaySeconds: 10,
			TimeoutSeconds:      3,
			PeriodSeconds:       10,
			SuccessThreshold:    1,
			FailureThreshold:    3,
		}
		deploy.Spec.Template.Spec.Containers[0].LivenessProbe = deploy.Spec.Template.Spec.Containers[0].ReadinessProbe
		deploy.Spec.Template.Spec.Containers[0].Lifecycle = &corev1.Lifecycle{
			PreStop: &corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/preStop",
					Port:   intstr.IntOrString{IntVal: port},
					Scheme: scheme,
				},
			},
		}
		if webhook.Spec.Resources != nil {
			deploy.Spec.Template.Spec.Containers[0].Resources = *webhook.Spec.Resources
		}

		var mode int32 = 420
		deploy.Spec.Template.Spec.Volumes = []corev1.Volume{
			{
				Name: "cert",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  fmt.Sprintf("%s-secret", webhook.Name),
						DefaultMode: &mode,
					},
				},
			},
			{
				Name: "host-time",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/etc/localtime",
					},
				},
			},
		}
		deploy.Spec.Template.Spec.Affinity = webhook.Spec.Affinity
		deploy.Spec.Template.Spec.Tolerations = webhook.Spec.Tolerations
		deploy.Spec.Template.Spec.NodeSelector = webhook.Spec.NodeSelector
		return controllerutil.SetControllerReference(webhook, deploy, r.Scheme)
	}
}

func (r *WebhookReconciler) svcMutate(webhook *v1alpha1.Webhook,
	svc *corev1.Service) controllerutil.MutateFn {

	return func() error {
		svc.Labels = webhook.Labels
		svc.Spec.Selector = map[string]string{
			"app": webhook.Name,
		}

		port := webhook.Spec.Port
		if port == 0 {
			if webhook.Spec.UseHTTPS != nil && *webhook.Spec.UseHTTPS {
				port = 6443
			} else {
				port = 8080
			}
		}
		svc.Spec.Ports = []corev1.ServicePort{
			{
				Name: "whizard-logging",
				Port: port,
				TargetPort: intstr.IntOrString{
					IntVal: port,
				},
				Protocol: corev1.ProtocolTCP,
			},
		}
		svc.Spec.Type = corev1.ServiceTypeClusterIP
		return controllerutil.SetControllerReference(webhook, svc, r.Scheme)
	}
}

func (r *WebhookReconciler) CreateSecret(ctx context.Context, secret *corev1.Secret, name, namespace string) error {

	ca, key, crt, err := utils.CreateCa(fmt.Sprintf("%s-svc.%s.svc", name, namespace))
	if err != nil {
		return err
	}

	secret.Data = make(map[string][]byte)
	secret.Data["tls.key"] = key
	secret.Data["tls.crt"] = crt
	secret.Data["caBundle"] = ca
	return r.Create(ctx, secret)
}
