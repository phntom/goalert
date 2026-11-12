{{/*
Expand the name of the chart.
*/}}
{{- define "goalert.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "goalert.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "goalert.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "goalert.labels" -}}
helm.sh/chart: {{ include "goalert.chart" . }}
{{ include "goalert.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "goalert.selectorLabels" -}}
app.kubernetes.io/name: {{ include "goalert.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Container image reference: repository:tag (tag defaults to the chart appVersion),
with an @<hash> digest suffix appended only when image.hash is set.
*/}}
{{- define "goalert.image" -}}
{{- $tag := default .Chart.AppVersion .Values.image.tag -}}
{{- printf "%s:%s" .Values.image.repository $tag -}}
{{- with .Values.image.hash }}
{{- printf "@%s" . }}
{{- end }}
{{- end }}

{{/*
Name of the Secret holding AUTH_TOKEN and APP_HASH:
config.existingSecret when set, the chart fullname otherwise.
*/}}
{{- define "goalert.secretName" -}}
{{- if .Values.config.existingSecret -}}
{{- .Values.config.existingSecret -}}
{{- else -}}
{{- include "goalert.fullname" . -}}
{{- end -}}
{{- end }}
