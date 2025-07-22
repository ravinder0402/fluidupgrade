{{- define "helper.jobIdentifier" -}}
  {{- printf "%s" now | date "20060102150405" -}}
{{- end -}}
