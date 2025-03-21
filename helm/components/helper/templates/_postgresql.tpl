{{/*
Returns the PostgreSQL initContainer
*/}}
{{- define "helper.postgresql.pgready" -}}
      - name: postgres-startup
        image: postgres:15
        imagePullPolicy: IfNotPresent
        command:
        - sh
        - -c
        - |
          until pg_isready -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER
          do
            echo "Waiting for postgres..."
            sleep 1;
          done
        env:
        - name: POSTGRES_HOST
          valueFrom:
            secretKeyRef:
              name: ccs-postgres-config
              key: DB_HOST
        - name: POSTGRES_USER
          valueFrom:
            secretKeyRef:
              name: ccs-postgres-config
              key: POSTGRES_USER
        - name: POSTGRES_PORT
          valueFrom:
            secretKeyRef:
              name: ccs-postgres-config
              key: DB_PORT
{{- end -}}