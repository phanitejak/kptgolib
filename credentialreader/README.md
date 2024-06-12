# This library is intended to use by services who requests the OEM specific credentials.

The service uses this library should define below parameters:

1. URL of the central service.

2. Kubernetes service account for pod to verify service authentication.

This library interacts with central service url (system-credential-orchestrator) to fetch OEM credentials.

Currently this library uses kubernetes service account for service authentication.