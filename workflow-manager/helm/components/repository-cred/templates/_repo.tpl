{{- define "repoRegistryKey.secret" -}}
  {{- $repoCreds := "" }}
  {{- $repo := .Values.repository }}
  {{- $cred := .Values.repositoryCred }}
  {{- $mail := default "@" $cred.mail }}
  {{- $auth := printf "%s:%s" $cred.user $cred.password | b64enc }}
  {{- $repoCreds = printf "\"%s\": {\"username\":\"%s\",\"password\":\"%s\",\"email\":\"%s\",\"auth\":\"%s\"}" $repo $cred.user $cred.password $mail $auth }}
  {{- printf "{%s}" $repoCreds | b64enc -}}
{{- end -}}
