{{/*
Returns the PostgreSQL port
*/}}
{{- define "ccspostgresql.port" -}}
{{- if .Values.global.postgresql.external.port -}}
{{- .Values.global.postgresql.external.port -}}
{{- else -}}
5432
{{- end -}}
{{- end -}}

{{/*
Returns the PostgreSQL user
*/}}
{{- define "ccspostgresql.user" -}}
{{- if .Values.global.postgresql.external.user -}}
{{- .Values.global.postgresql.external.user -}}
{{- else if not .Values.global.postgresql.architecture.standalone -}}
{{- .Values.postgresqlHA.creds.user -}}
{{- else -}}
admin
{{- end -}}
{{- end -}}

{{/*
Returns the PostgreSQL host
*/}}
{{- define "ccspostgresql.host" -}}
{{- if .Values.global.postgresql.external.host -}}
{{- .Values.global.postgresql.external.host -}}
{{- else if not .Values.global.postgresql.architecture.standalone -}}
ccs-postgresql-cluster
{{- else -}}
ccs-postgres
{{- end -}}
{{- end -}}

{{/*
Returns the PostgreSQL password
*/}}
{{- define "ccspostgresql.password" -}}
{{- if .Values.global.postgresql.external.password -}}
{{- .Values.global.postgresql.external.password -}}
{{- else if not .Values.global.postgresql.architecture.standalone -}}
{{- .Values.postgresqlHA.creds.pass -}}
{{- else -}}
admin
{{- end -}}
{{- end -}}
---
{{/*
Returns the AuditDB port
*/}}
{{- define "ccsauditdb.port" -}}
{{- if .Values.global.auditdb.port -}}
{{- .Values.global.auditdb.port -}}
{{- else -}}
{{- include "ccspostgresql.port" . -}}
{{- end -}}
{{- end -}}

{{/*
Returns the AuditDB user
*/}}
{{- define "ccsauditdb.username" -}}
{{- if .Values.global.auditdb.username -}}
{{- .Values.global.auditdb.username -}}
{{- else -}}
{{- include "ccspostgresql.user" . -}}
{{- end -}}
{{- end -}}

{{/*
Returns the AuditDB host
*/}}
{{- define "ccsauditdb.host" -}}
{{- if .Values.global.auditdb.host -}}
{{- .Values.global.auditdb.host -}}
{{- else -}}
{{- include "ccspostgresql.host" . -}}
{{- end -}}
{{- end -}}

{{/*
Returns the AuditDB password
*/}}
{{- define "ccsauditdb.password" -}}
{{- if .Values.global.auditdb.password -}}
{{- .Values.global.auditdb.password -}}
{{- else -}}
{{- include "ccspostgresql.password" . -}}
{{- end -}}
{{- end -}}

{{/*
Returns the AuditDB database
*/}}
{{- define "ccsauditdb.database" -}}
{{- if .Values.global.auditdb.database -}}
{{- .Values.global.auditdb.database -}}
{{- else -}}
audit-db
{{- end -}}
{{- end -}}