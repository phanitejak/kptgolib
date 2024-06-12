#
# Subchart/helper chart calls mainchart.app from this file.
# All NEO applications which use the subchart need to have mainchart.app.
# See more: https://confluence.int.net.nokia.com/display/NEO/Helm+Helper+Chart  

{{- define "mainchart.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}
 
{{- define "mainchart.app" -}}
{{template "mainchart.name" .}}
{{- end -}}
