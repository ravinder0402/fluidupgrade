{{/*
Helper template to generate the frontend URL or fallback to /auth
*/}}
{{- define "helper.keycloakFrontendUrl" -}}
{{- if and (not .Values.global.keycloak.internal) ( .Values.global.keycloak.override.frontendUrl) -}}
  {{- printf "%s" .Values.global.keycloak.override.frontendUrl }}
{{- else -}}
  {{- printf "/auth"}}
{{- end -}}
{{- end -}}

{{/*
Helper template to generate Keycloak endpoints
*/}}
{{- define "helper.keycloakHttpEndpoint" -}}
{{- if and (not .Values.global.keycloak.internal) ( .Values.global.keycloak.override.service.name) -}}
  {{- printf "http://%s:%d" .Values.global.keycloak.override.service.name (int .Values.global.keycloak.override.service.httpPort) -}}
{{- else -}}
  {{- printf "http://keycloak:8080"}}
{{- end -}}
{{- end -}}

{{- define "helper.keycloakHttpsEndpoint" -}}
{{- if and (not .Values.global.keycloak.internal) ( .Values.global.keycloak.override.service.name) -}}
  {{- printf "https://%s:%d" .Values.global.keycloak.override.service.name (int .Values.global.keycloak.override.service.httpsPort) -}}
{{- else -}}
  {{- printf "https://keycloak:8443"}}
{{- end -}}
{{- end -}}

{{/*
Helper template to generate keycloak secret name
*/}}
{{- define "helper.keycloakSecretName" -}}
{{- if and (not .Values.global.keycloak.internal) ( .Values.global.keycloak.override.creds) -}}
  {{- printf "keycloak-override-creds" }}
{{- else -}}
  {{- printf "keycloak-admin" }}
{{- end -}}
{{- end -}}
