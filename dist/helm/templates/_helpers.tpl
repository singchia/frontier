{{/*
Expand the name of the chart.
*/}}
{{- define "frontier.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "frontier.fullname" -}}
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

{{/* Per-component fullnames. */}}
{{- define "frontier.frontier.fullname" -}}
{{- printf "%s-frontier" (include "frontier.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "frontier.frontlas.fullname" -}}
{{- printf "%s-frontlas" (include "frontier.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/* Chart label. */}}
{{- define "frontier.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/* Common labels (shared by both components). */}}
{{- define "frontier.commonLabels" -}}
helm.sh/chart: {{ include "frontier.chart" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: {{ include "frontier.name" . }}
{{- end }}

{{/* Frontier labels + selector. */}}
{{- define "frontier.frontier.labels" -}}
{{ include "frontier.commonLabels" . }}
app.kubernetes.io/name: {{ include "frontier.name" . }}
app.kubernetes.io/component: frontier
{{- end }}

{{- define "frontier.frontier.selectorLabels" -}}
app.kubernetes.io/name: {{ include "frontier.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: frontier
{{- end }}

{{/* Frontlas labels + selector. */}}
{{- define "frontier.frontlas.labels" -}}
{{ include "frontier.commonLabels" . }}
app.kubernetes.io/name: {{ include "frontier.name" . }}
app.kubernetes.io/component: frontlas
{{- end }}

{{- define "frontier.frontlas.selectorLabels" -}}
app.kubernetes.io/name: {{ include "frontier.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: frontlas
{{- end }}

{{/* Service account names. */}}
{{- define "frontier.frontier.serviceAccountName" -}}
{{- if .Values.frontier.serviceAccount.create }}
{{- default (include "frontier.frontier.fullname" .) .Values.frontier.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.frontier.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "frontier.frontlas.serviceAccountName" -}}
{{- if .Values.frontlas.serviceAccount.create }}
{{- default (include "frontier.frontlas.fullname" .) .Values.frontlas.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.frontlas.serviceAccount.name }}
{{- end }}
{{- end }}

{{/* Resolve image pull policy (per-component override > global default). */}}
{{- define "frontier.pullPolicy" -}}
{{- if .comp.image.pullPolicy }}{{ .comp.image.pullPolicy }}{{ else }}{{ .global.imagePullPolicy }}{{ end }}
{{- end }}

{{/* Resolve image tag (per-component override > Chart.AppVersion). */}}
{{- define "frontier.imageTag" -}}
{{- if .comp.image.tag }}{{ .comp.image.tag }}{{ else }}{{ .root.Chart.AppVersion }}{{ end }}
{{- end }}

{{/* Frontlas Service FQDN — used by frontier to dial the frontier-plane port. */}}
{{- define "frontier.frontlas.serviceFQDN" -}}
{{ include "frontier.frontlas.fullname" . }}.{{ .Release.Namespace }}.svc.cluster.local
{{- end }}

{{/*
Resolve Redis connection details for Frontlas. Returns a YAML dict that
deployment_frontlas.yaml fromYaml-decodes to build env vars.

When redis.enabled is true, target the bundled `<release>-redis-master`
service and the auto-generated `<release>-redis` Secret.
Otherwise read from .Values.frontlas.externalRedis.
*/}}
{{- define "frontier.frontlas.redis" -}}
{{- if .Values.redis.enabled -}}
addrs: "{{ .Release.Name }}-redis-master.{{ .Release.Namespace }}.svc.cluster.local:6379"
user: ""
redisType: standalone
masterName: ""
db: 0
passwordSecretName: "{{ .Release.Name }}-redis"
passwordSecretKey: "redis-password"
{{- else -}}
addrs: "{{ join "," .Values.frontlas.externalRedis.addrs }}"
user: "{{ .Values.frontlas.externalRedis.user }}"
redisType: {{ .Values.frontlas.externalRedis.redisType }}
masterName: "{{ .Values.frontlas.externalRedis.masterName }}"
db: {{ .Values.frontlas.externalRedis.db }}
passwordSecretName: "{{ .Values.frontlas.externalRedis.passwordSecret.name }}"
passwordSecretKey: "{{ .Values.frontlas.externalRedis.passwordSecret.key }}"
{{- end -}}
{{- end }}
