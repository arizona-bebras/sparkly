{{- define "sparkly.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 40 | trimSuffix "-" -}}
{{- end -}}
{{- define "sparkly.fullname" -}}
{{- default (printf "%s-%s" .Release.Name (include "sparkly.name" .)) .Values.fullnameOverride | trunc 40 | trimSuffix "-" -}}
{{- end -}}
{{- define "sparkly.labels" -}}
app.kubernetes.io/name: {{ include "sparkly.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | quote }}
{{- end -}}
{{- define "sparkly.selector" -}}
app.kubernetes.io/name: {{ include "sparkly.name" .root }}
app.kubernetes.io/instance: {{ .root.Release.Name }}
app.kubernetes.io/component: {{ .component }}
{{- end -}}
{{- define "sparkly.image" -}}
{{- $tag := default .root.Chart.AppVersion .image.tag -}}
{{- printf "%s/%s:%s" (trimSuffix "/" .root.Values.global.imageRegistry) .image.repository (required "Chart.appVersion or an explicit component image tag is required" $tag) -}}
{{- end -}}
