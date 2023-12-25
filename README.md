# Whizard-telemetry-ruler

WhizardTelemetryRuler provides one K8s webhook (whizard-telemetry-ruler), one cluster CRDs (ClusterRuleGroup),a k8s configmap(sink) for send message.

## Requirements

You must have a Kubernetes cluster (v1.13+), and config the kube-apiserver follow [this](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/). The webhook URL is:

````bash
https://${webhook}-svc.${namespace}:${port}/webhook/auditing/
https://${webhook}-svc.${namespace}:${port}/webhook/events/
````

You can get the ca from secret ${webhook}-secret in the namespace which WhizardTelemetryRuler deployed.
- ${webhook} is the name of service , default is `whizard-telemetry-ruler-svc`.
- ${namespace} is the namespace which the WhizardTelemetryRuler deployed.
- ${port} is the webhook port. 

## Install

### Install with yaml

Install the latest stable version

```shell
kubectl apply -f https://raw.githubusercontent.com/WhizardTelemetry/WhizardTelemetryRuler/releae-0.1/deploy/yaml/bundle.yam

# You can change the namespace in deploy/yaml/kustomization.yaml in corresponding release branch 
# and then use command below to install to another namespace
# kubectl kustomize deploy/yaml | kubectl apply -f -
```

Install the development version

```shell
kubectl apply -f https://raw.githubusercontent.com/WhizardTelemetry/WhizardTelemetryRuler/master/deploy/yaml/bundle.yam

# You can change the namespace in deploy/yaml/kustomization.yaml 
# and then use command below to install to another namespace
# kubectl kustomize deploy/yaml | kubectl apply -f -
```

### Install with helm

```shell
 helm upgrade --install WhizardTelemetryRuler deploy/helm/v3 -n kubesphere-logging-system 
```

## Introduction
WhizardTelemetryRuler provides one K8s webhook (Whizard-telemetry-ruler), one cluster CRDs (ClusterRuleGroup),a k8s configmap(sink) for send message.


#### WhizardTelemetryRuler Severity
WhizardTelemetryRuler rule has an attribute named severity, the known value of priority from low to high are INFO,WARNING,ERROR,CRITICAL. 

