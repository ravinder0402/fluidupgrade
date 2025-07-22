{{- define "helper.imageTag" -}}
  {{ if and .Values.global.ReleaseTag (not .Values.global.UseDailyBuilds) }}
    {{- printf "%s" .Values.global.ReleaseTag -}}
  {{ else }}
    {{- printf "%s" now | date "2006-01-02" -}}
  {{ end }}
{{- end -}}
