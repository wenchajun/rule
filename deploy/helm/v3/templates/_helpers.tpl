{{/*
Common labels
*/}}
{{- define "whizard-telemetry-ruler.labels" -}}
app.kubernetes.io/name: {{ "whizard-telemetry-ruler" }}
helm.sh/chart: {{ .Chart.Version }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}