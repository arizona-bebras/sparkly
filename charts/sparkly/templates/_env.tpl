{{- define "sparkly.env" -}}
{{- $v := .root.Values -}}
{{- $name := .component -}}
{{- if eq $name "envoy" }}
[]
{{- else if eq $name "frontend" }}
- {name: HOST, value: "0.0.0.0"}
- {name: PORT, value: "3000"}
- {name: ORIGIN, value: {{ printf "https://%s" $v.routing.appHost | quote }}}
{{- else if eq $name "ml" }}
- {name: PORT, value: "8082"}
{{- else }}
- name: RABBITMQ_URL
  valueFrom:
    secretKeyRef: {name: {{ $v.rabbitmq.connectionSecret }}, key: {{ $v.rabbitmq.urlKey }}}
{{- if eq $name "backend" }}
- {name: MAIN_PORT, value: "8080"}
- {name: POSTGRES_HOST, value: {{ $v.postgres.host | quote }}}
- {name: POSTGRES_PORT, value: {{ $v.postgres.port | quote }}}
- {name: POSTGRES_DB, value: {{ $v.postgres.database | quote }}}
- name: POSTGRES_USER
  valueFrom:
    secretKeyRef: {name: {{ $v.postgres.credentialsSecret }}, key: {{ $v.postgres.usernameKey }}}
- name: POSTGRES_PASSWORD
  valueFrom:
    secretKeyRef: {name: {{ $v.postgres.credentialsSecret }}, key: {{ $v.postgres.passwordKey }}}
- name: JWKS_MASTER_KEY
  valueFrom:
    secretKeyRef: {name: {{ $v.backend.jwksSecret }}, key: {{ $v.backend.jwksMasterKeyKey }}}
- {name: STEP_CA_URL, value: {{ $v.stepCA.url | quote }}}
- {name: STEP_CA_PROVISIONER, value: {{ $v.stepCA.provisioner | quote }}}
- {name: STEP_CA_ROOT_FILE, value: /run/step-ca/root_ca.crt}
- {name: STEP_CA_KEY_FILE, value: /run/step-ca/provisioner.key}
- {name: STEP_CA_KEY_ID_FILE, value: /run/step-ca/provisioner.kid}
{{- else if eq $name "analytics" }}
- {name: PORT, value: "8081"}
- {name: RAW_PERSISTENCE_WORKERS, value: {{ $v.analytics.rawPersistenceWorkers | quote }}}
- {name: CLICKHOUSE_HOST, value: {{ $v.clickhouse.host | quote }}}
- {name: CLICKHOUSE_NATIVE_PORT, value: {{ $v.clickhouse.port | quote }}}
- {name: CLICKHOUSE_DB, value: {{ $v.clickhouse.database | quote }}}
- name: CLICKHOUSE_USER
  valueFrom:
    secretKeyRef: {name: {{ $v.clickhouse.credentialsSecret }}, key: {{ $v.clickhouse.usernameKey }}}
- name: CLICKHOUSE_PASSWORD
  valueFrom:
    secretKeyRef: {name: {{ $v.clickhouse.credentialsSecret }}, key: {{ $v.clickhouse.passwordKey }}}
{{- end }}
{{- end }}
{{- end -}}
