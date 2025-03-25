{{- define "helper.imageTag" -}}
  {{ if and .Values.global.releaseTag (not .Values.global.UseDailyBuilds) }}
    {{- printf "%s" .Values.global.releaseTag -}}
  {{ else }}
    {{- printf "%s" now | date "2006-01-02" -}}
  {{ end }}
{{- end -}}
