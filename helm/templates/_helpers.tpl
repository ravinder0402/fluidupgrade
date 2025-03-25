{{/*
Returns the PostgreSQL port
*/}}
{{- define "postgresql.port" -}}
{{- if .Values.global.postgresql.external.port -}}
{{- .Values.global.postgresql.external.port -}}
{{- else -}}
5432
{{- end -}}
{{- end -}}

{{/*
Returns the PostgreSQL user
*/}}
{{- define "postgresql.user" -}}
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
{{- define "postgresql.host" -}}
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
{{- define "postgresql.password" -}}
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
{{- define "auditdb.port" -}}
{{- if .Values.global.auditdb.port -}}
{{- .Values.global.auditdb.port -}}
{{- else -}}
{{- include "postgresql.port" . -}}
{{- end -}}
{{- end -}}

{{/*
Returns the AuditDB user
*/}}
{{- define "auditdb.username" -}}
{{- if .Values.global.auditdb.username -}}
{{- .Values.global.auditdb.username -}}
{{- else -}}
{{- include "postgresql.user" . -}}
{{- end -}}
{{- end -}}

{{/*
Returns the AuditDB host
*/}}
{{- define "auditdb.host" -}}
{{- if .Values.global.auditdb.host -}}
{{- .Values.global.auditdb.host -}}
{{- else -}}
{{- include "postgresql.host" . -}}
{{- end -}}
{{- end -}}

{{/*
Returns the AuditDB password
*/}}
{{- define "auditdb.password" -}}
{{- if .Values.global.auditdb.password -}}
{{- .Values.global.auditdb.password -}}
{{- else -}}
{{- include "postgresql.password" . -}}
{{- end -}}
{{- end -}}

{{/*
Returns the AuditDB database
*/}}
{{- define "auditdb.database" -}}
{{- if .Values.global.auditdb.database -}}
{{- .Values.global.auditdb.database -}}
{{- else -}}
audit-db
{{- end -}}
{{- end -}}