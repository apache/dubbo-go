/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package application

const (
	chartYaml = `apiVersion: v1
name: dubbo-go-app
description: dubbo-go-app

# A chart can be either an 'application' or a 'library' chart.
#
# Application charts are a collection of templates that can be packaged into versioned archives
# to be deployed.
#
# Library charts provide useful utilities or functions for the chart developer. They're included as
# a dependency of application charts to inject those utilities and functions into the rendering
# pipeline. Library charts do not define any templates and therefore cannot be deployed.
type: application

# This is the chart version. This version number should be incremented each time you make changes
# to the chart and its templates, including the app version.
# Versions are expected to follow Semantic Versioning (https://semver.org/)
version: 0.0.1

# This is the version number of the application being deployed. This version number should be
# incremented each time you make changes to the application. Versions are not expected to
# follow Semantic Versioning. They should reflect the version the application is using.
# It is recommended to use it with quotes.
appVersion: "1.16.0"
`
	valuesYaml = `replicaCount: 1

image:
  repository: $(your_repo)/$(namespace)/$(image_name)
  pullPolicy: Always
  tag: "1.0.0"

# Dubbo-go-mesh version control labels
version:
  labels:
    dubbogoAppVersion: 0.0.1

container:
  env:
    - name: POD_NAME
      valueFrom:
        fieldRef:
          fieldPath: metadata.name
    - name: POD_NAMESPACE
      valueFrom:
        fieldRef:
          fieldPath: metadata.namespace
  ports:
    - name: triple
      containerPort: 20000
      protocol: TCP
  volumeMounts:
    - mountPath: /var/run/secrets/token
      name: istio-token

volumes:
  - name: istio-token
    projected:
      sources:
        - serviceAccountToken:
            audience: istio-ca
            path: istio-token

imagePullSecrets: []
nameOverride: ""
fullnameOverride: ""

serviceAccount:
  # Specifies whether a service account should be created
  create: true
  # Annotations to add to the service account
  annotations: {}
  # The name of the service account to use.
  # If not set and create is true, a name is generated using the fullname template
  name: ""

podAnnotations: {}

podSecurityContext: {}
# fsGroup: 2000

securityContext: {}
  # capabilities:
  #   drop:
  #   - ALL
  # readOnlyRootFilesystem: true
  # runAsNonRoot: true
# runAsUser: 1000

service:
  type: ClusterIP
  port: 20000
  portName: triple

nodeSelector: {}

tolerations: []

affinity: {}
`

	helpersTPL = `{{/*
Expand the name of the chart.
*/}}
{{- define "app.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
Can we fix line 'if .Values.version.labels.dubbogoAppVersion' if user doesn't want to set app version?
*/}}
{{- define "app.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- if .Values.version.labels.dubbogoAppVersion }}
{{- $version := .Values.version.labels.dubbogoAppVersion }}
{{- printf "%s-%s" .Chart.Name $version  | trimSuffix "-" }}
{{- else }}
{{- printf "%s" .Chart.Name }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "app.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "app.labels" -}}
helm.sh/chart: {{ include "app.chart" . }}
{{ include "app.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}


{{/*
Selector labels
*/}}
{{- define "app.selectorLabels" -}}
app.kubernetes.io/name: {{ include "app.name" . }}
{{- end }}

{{/*
AppVersioned Selector labels
Used by Deployment and Pod
To management version control.
*/}}
{{- define "app.versionedSelectorLabels" -}}
{{- include "app.labels" . }}
{{- with .Values.version.labels.dubbogoAppVersion }}
dubbogoAppVersion: {{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "app.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "app.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
`
	deploymentYaml = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "app.fullname" . }}
  labels:
    {{- include "app.labels" . | nindent 4 }}
    {{- toYaml .Values.version.labels | nindent 4 }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      {{- include "app.versionedSelectorLabels" . | nindent 8 }}
  template:
    metadata:
      {{- with .Values.podAnnotations }}
      annotations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      labels:
        {{- include "app.versionedSelectorLabels" . | nindent 8 }}
    spec:
      {{- with .Values.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.serviceAccountName }}
      serviceAccountName: {{ .Values.serviceAccountName }}
      {{- end }}
      securityContext:
        {{- toYaml .Values.podSecurityContext | nindent 8 }}
      containers:
        - name: {{ .Chart.Name }}
          securityContext:
            {{- toYaml .Values.securityContext | nindent 12 }}
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          env:
            {{- toYaml .Values.container.env | nindent 12}}
          ports:
            {{- toYaml .Values.container.ports | nindent 12 }}
          resources:
            {{- toYaml .Values.resources | nindent 12 }}
          {{- with .Values.container.volumeMounts }}
          volumeMounts:
            {{- toYaml . | nindent 12 }}
          {{- end }}
      {{- with .Values.nodeSelector }}
      nodeSelector:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.affinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.tolerations }}
      tolerations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.volumes }}
      volumes:
        {{- toYaml . | nindent 8 }}
      {{- end }}
`
	serviceYaml = `# Dubbo-go version control, we do not update service if there is exsiting service, because
# service is an app-level resource, helm install service with a different helmName again to add an app
# version would cause failed.
{{- $svc := lookup "v1" "Service" .Release.Namespace  .Chart.Name }}
{{- if not $svc }}
apiVersion: v1
kind: Service
metadata:
  name: {{ include "app.name" . }}
  labels:
    {{- include "app.labels" . | nindent 4 }}
spec:
  type: {{ .Values.service.type }}
  ports:
    - port: {{ .Values.service.port }}
      targetPort: {{ .Values.service.portName }}
      protocol: TCP
      name: {{ .Values.service.portName }}
  selector:
    {{- include "app.selectorLabels" . | nindent 4 }}
{{- end }}
`
	serviceAccountYaml = `{{- if .Values.serviceAccount.create -}}
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ include "app.serviceAccountName" . }}
  labels:
    {{- include "app.labels" . | nindent 4 }}
  {{- with .Values.serviceAccount.annotations }}
  annotations:
    {{- toYaml . | nindent 4 }}
  {{- end }}
{{- end }}
`
)

const (
	nacosEnvChartFile = `apiVersion: v1
name: nacos
description: nacos environment

# A chart can be either an 'application' or a 'library' chart.
#
# Application charts are a collection of templates that can be packaged into versioned archives
# to be deployed.
#
# Library charts provide useful utilities or functions for the chart developer. They're included as
# a dependency of application charts to inject those utilities and functions into the rendering
# pipeline. Library charts do not define any templates and therefore cannot be deployed.
type: application

# This is the chart version. This version number should be incremented each time you make changes
# to the chart and its templates, including the app version.
# Versions are expected to follow Semantic Versioning (https://semver.org/)
version: 0.0.1

# This is the version number of the application being deployed. This version number should be
# incremented each time you make changes to the application. Versions are not expected to
# follow Semantic Versioning. They should reflect the version the application is using.
# It is recommended to use it with quotes.
appVersion: "1.16.0"
`
	nacosEnvValuesFile = `replicaCount: 1

image:
  repository: nacos/nacos-server
  pullPolicy: IfNotPresent
  tag: "2.0.1"

version:
  labels:
    dubbogoAppVersion: latest

container:
  env:
    - name: MODE
      value: "standalone"
  ports:
    - name: http
      containerPort: 8848
      protocol: TCP


imagePullSecrets: []
nameOverride: ""
fullnameOverride: ""

serviceAccount:
  # Specifies whether a service account should be created
  create: true
  # Annotations to add to the service account
  annotations: {}
  # The name of the service account to use.
  # If not set and create is true, a name is generated using the fullname template
  name: ""

podAnnotations: {}

podSecurityContext: {}
  # fsGroup: 2000

securityContext: {}
  # capabilities:
  #   drop:
  #   - ALL
  # readOnlyRootFilesystem: true
  # runAsNonRoot: true
  # runAsUser: 1000

service:
  type: ClusterIP
  port: 8848
  portName: http

nodeSelector: {}

tolerations: []

affinity: {}

`
)

func init() {
	// App chart
	fileMap["chartYaml"] = &fileGenerator{
		path:    "./chart/app",
		file:    "Chart.yaml",
		context: chartYaml,
	}

	fileMap["valuesYaml"] = &fileGenerator{
		path:    "./chart/app",
		file:    "values.yaml",
		context: valuesYaml,
	}

	fileMap["helpersTPL"] = &fileGenerator{
		path:    "./chart/app/templates",
		file:    "_helpers.tpl",
		context: helpersTPL,
	}
	fileMap["deploymentYaml"] = &fileGenerator{
		path:    "./chart/app/templates",
		file:    "deployment.yaml",
		context: deploymentYaml,
	}
	fileMap["serviceYaml"] = &fileGenerator{
		path:    "./chart/app/templates",
		file:    "service.yaml",
		context: serviceYaml,
	}
	fileMap["serviceAccountYaml"] = &fileGenerator{
		path:    "./chart/app/templates",
		file:    "serviceaccount.yaml",
		context: serviceAccountYaml,
	}

	// Nacos env chart
	fileMap["nacosEnvchartYaml"] = &fileGenerator{
		path:    "./chart/nacos_env",
		file:    "Chart.yaml",
		context: nacosEnvChartFile,
	}

	fileMap["nacosEnvvaluesYaml"] = &fileGenerator{
		path:    "./chart/nacos_env",
		file:    "values.yaml",
		context: nacosEnvValuesFile,
	}

	fileMap["nacosEnvHelpersTPL"] = &fileGenerator{
		path:    "./chart/nacos_env/templates",
		file:    "_helpers.tpl",
		context: helpersTPL,
	}
	fileMap["nacosEnvDeploymentYaml"] = &fileGenerator{
		path:    "./chart/nacos_env/templates",
		file:    "deployment.yaml",
		context: deploymentYaml,
	}
	fileMap["nacosEnvServiceYaml"] = &fileGenerator{
		path:    "./chart/nacos_env/templates",
		file:    "service.yaml",
		context: serviceYaml,
	}
}
