{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "<<< .ServiceName >>>.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "<<< .ServiceName >>>.fullname" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- /*
<<< .ServiceName >>>.chartref prints a chart name and version.
It does minimal escaping for use in Kubernetes labels.
*/ -}}
{{- define "<<< .ServiceName >>>.chartref" -}}
  {{- replace "+" "_" .Chart.Version | printf "%s-%s" .Chart.Name -}}
{{- end -}}

{{/*
<<< .ServiceName >>>.labels.standard prints the standard Helm labels.
The standard labels are frequently used in metadata.
*/}}
{{- define "<<< .ServiceName >>>.labels.standard" -}}
app: {{template "<<< .ServiceName >>>.name" .}}
chart: {{template "<<< .ServiceName >>>.chartref" . }}
app.kubernetes.io/name: {{template "<<< .ServiceName >>>.name" .}}
helm.sh/chart: {{template "<<< .ServiceName >>>.chartref" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/version: {{ .Chart.Version }}
com.nokia.neo/commitId: ${COMMIT_ID}
{{- end -}}

{{/*
<<< .ServiceName >>>.template.labels prints the template metadata labels.
*/}}
{{- define "<<< .ServiceName >>>.template.labels" -}}
app: {{template "<<< .ServiceName >>>.name" .}}
release: {{ .Release.Name }}
{{- end -}}

{{- define "<<< .ServiceName >>>.app" -}}
app: {{template "<<< .ServiceName >>>.name" .}}
{{- end -}}

{{- define "<<< .ServiceName >>>.release" -}}
release: {{ .Release.Name }}
{{- end -}}
