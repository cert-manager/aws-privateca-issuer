# AWS Private CA Issuer

AWS Private CA is an AWS service that can setup and manage private CAs, as well as issue private certifiates.

cert-manager is a Kubernetes add-on to automate the management and issuance of TLS certificates from various issuing sources.
It will ensure certificates are valid and up to date periodically, and attempt to renew certificates at an appropriate time before expiry.

This project acts as an addon (see https://cert-manager.io/docs/configuration/external/) to cert-manager that signs off certificate requests using AWS Private CA.

## Values

<!-- AUTO-GENERATED -->

### AWS Private CA Issuer


<table>
<tr>
<th>Property</th>
<th>Description</th>
<th>Type</th>
<th>Default</th>
</tr>
<tr>

<td>replicaCount</td>
<td>

Number of replicas to run of the issuer

</td>
<td>number</td>
<td>

```yaml
2
```

</td>
</tr>
<tr>

<td>image.repository</td>
<td>

Image repository

</td>
<td>string</td>
<td>

```yaml
public.ecr.aws/k1n1h4h4/cert-manager-aws-privateca-issuer
```

</td>
</tr>
<tr>

<td>image.pullPolicy</td>
<td>

Image pull policy

</td>
<td>string</td>
<td>

```yaml
IfNotPresent
```

</td>
</tr>
<tr>

<td>image.tag</td>
<td>

Image tag (used only when digest is empty)

</td>
<td>string</td>
<td>

```yaml
""
```

</td>
</tr>
<tr>

<td>image.digest</td>
<td>

Image digest (overrides tag when set). Example: sha256:aaaaaa...

</td>
<td>string</td>
<td>

```yaml
""
```

</td>
</tr>
<tr>

<td>disableApprovedCheck</td>
<td>

Disable waiting for CertificateRequests to be Approved before signing

</td>
<td>bool</td>
<td>

```yaml
false
```

</td>
</tr>
<tr>

<td>disableClientSideRateLimiting</td>
<td>

Disables Kubernetes client-side rate limiting (only use if API Priority & Fairness is enabled on the cluster).

</td>
<td>bool</td>
<td>

```yaml
false
```

</td>
</tr>
<tr>

<td>imagePullSecrets</td>
<td>

Optional secrets used for pulling the container image  
  
For example:

```yaml
imagePullSecrets:
- name: secret-name
```

</td>
<td>array</td>
<td>

```yaml
[]
```

</td>
</tr>
<tr>

<td>nameOverride</td>
<td>

Override the name of the objects created by this chart

</td>
<td>string</td>
<td>

```yaml
""
```

</td>
</tr>
<tr>

<td>fullnameOverride</td>
<td>

Override the name of the objects created by this chart

</td>
<td>string</td>
<td>

```yaml
""
```

</td>
</tr>
<tr>

<td>revisionHistoryLimit</td>
<td>

Number deployment revisions to keep

</td>
<td>number</td>
<td>

```yaml
10
```

</td>
</tr>
<tr>

<td>serviceAccount.create</td>
<td>

Specifies whether a service account should be created

</td>
<td>bool</td>
<td>

```yaml
true
```

</td>
</tr>
<tr>

<td>serviceAccount.annotations</td>
<td>

Annotations to add to the service account

</td>
<td>object</td>
<td>

```yaml
{}
```

</td>
</tr>
<tr>

<td>serviceAccount.name</td>
<td>

The name of the service account to use.  
If not set and create is true, a name is generated using the fullname template

</td>
<td>string</td>
<td>

```yaml
""
```

</td>
</tr>
<tr>

<td>rbac.create</td>
<td>

Specifies whether RBAC should be created

</td>
<td>bool</td>
<td>

```yaml
true
```

</td>
</tr>
<tr>

<td>service.type</td>
<td>

Type of service to create

</td>
<td>string</td>
<td>

```yaml
ClusterIP
```

</td>
</tr>
<tr>

<td>service.port</td>
<td>

Port the service should listen on

</td>
<td>number</td>
<td>

```yaml
8080
```

</td>
</tr>
<tr>

<td>podAnnotations</td>
<td>

Annotations to add to the issuer Pod

</td>
<td>object</td>
<td>

```yaml
{}
```

</td>
</tr>
<tr>

<td>podSecurityContext</td>
<td>

Pod security context


</td>
<td>object</td>
<td>

```yaml
runAsUser: 65532
```

</td>
</tr>
<tr>

<td>securityContext</td>
<td>

Container security context


</td>
<td>object</td>
<td>

```yaml
allowPrivilegeEscalation: false
```

</td>
</tr>
<tr>

<td>resources</td>
<td>

Kubernetes pod resources requests/limits


</td>
<td>object</td>
<td>

```yaml
limits:
  cpu: 50m
  memory: 64Mi
requests:
  cpu: 50m
  memory: 64Mi
```

</td>
</tr>
<tr>

<td>nodeSelector</td>
<td>

Kubernetes node selector: node labels for pod assignment

</td>
<td>object</td>
<td>

```yaml
{}
```

</td>
</tr>
<tr>

<td>tolerations</td>
<td>

Kubernetes pod tolerations for cert-manager-csi-driver  
  
For example:

```yaml
tolerations:
- operator: "Exists"
```

</td>
<td>array</td>
<td>

```yaml
[]
```

</td>
</tr>
<tr>

<td>affinity</td>
<td>

A Kubernetes Affinity; see https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#affinity-v1-core  
  
For example:

```yaml
affinity:
  nodeAffinity:
   requiredDuringSchedulingIgnoredDuringExecution:
     nodeSelectorTerms:
     - matchExpressions:
       - key: foo.bar.com/role
         operator: In
         values:
         - master
```


</td>
<td>object</td>
<td>

```yaml
podAntiAffinity:
  preferredDuringSchedulingIgnoredDuringExecution:
    - podAffinityTerm:
        labelSelector:
          matchExpressions:
            - key: app.kubernetes.io/name
              operator: In
              values:
                - aws-privateca-issuer
        topologyKey: kubernetes.io/hostname
      weight: 100
```

</td>
</tr>
<tr>

<td>topologySpreadConstraints</td>
<td>

List of Kubernetes TopologySpreadConstraints; see https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#topologyspreadconstraint-v1-core


</td>
<td>array</td>
<td>

```yaml
- labelSelector:
    matchLabels:
      app.kubernetes.io/name: aws-privateca-issuer
  maxSkew: 1
  topologyKey: topology.kubernetes.io/zone
  whenUnsatisfiable: ScheduleAnyway
```

</td>
</tr>
<tr>

<td>priorityClassName</td>
<td>

Priority class name for the issuer pods  
If specified, this will set the priority class on pods, which can influence scheduling decisions  
  
For example:

```yaml
priorityClassName: high-priority
```

</td>
<td>string</td>
<td>

```yaml
""
```

</td>
</tr>
<tr>

<td>env</td>
<td>

Additional environment variables to set in the Pod


</td>
<td>object</td>
<td>

```yaml
null
```

</td>
</tr>
<tr>

<td>podLabels</td>
<td>

Additional labels to add to the Pod

</td>
<td>object</td>
<td>

```yaml
{}
```

</td>
</tr>
<tr>

<td>volumes</td>
<td>

Additional volumes on the operator container.

</td>
<td>array</td>
<td>

```yaml
[]
```

</td>
</tr>
<tr>

<td>volumeMounts</td>
<td>

Additional VolumeMounts on the operator container.

</td>
<td>array</td>
<td>

```yaml
[]
```

</td>
</tr>
<tr>

<td>podDisruptionBudget.maxUnavailable</td>
<td>

</td>
<td>number</td>
<td>

```yaml
1
```

</td>
</tr>
</table>

### Autoscaling


<table>
<tr>
<th>Property</th>
<th>Description</th>
<th>Type</th>
<th>Default</th>
</tr>
<tr>

<td>autoscaling.enabled</td>
<td>

Enable auto scaling using a HorizontalPodAutoscaler

</td>
<td>bool</td>
<td>

```yaml
false
```

</td>
</tr>
<tr>

<td>autoscaling.minReplicas</td>
<td>

Minimum number of replicas to deploy

</td>
<td>number</td>
<td>

```yaml
1
```

</td>
</tr>
<tr>

<td>autoscaling.maxReplicas</td>
<td>

Maximum number of replicas to deploy

</td>
<td>number</td>
<td>

```yaml
100
```

</td>
</tr>
<tr>

<td>autoscaling.targetCPUUtilizationPercentage</td>
<td>

CPU threshold to scale at as a percentage of the requested CPUs

</td>
<td>number</td>
<td>

```yaml
80
```

</td>
</tr>
<tr>

<td>autoscaling.targetMemoryUtilizationPercentage</td>
<td>

Memory threshold to scale at as a percentage of the requested memory


</td>
<td>number</td>
<td>

```yaml

```

</td>
</tr>
</table>

### Approver Role


Options for configuring a target ServiceAccount with the role to approve all awspca.cert-manager.io requests.

<table>
<tr>
<th>Property</th>
<th>Description</th>
<th>Type</th>
<th>Default</th>
</tr>
<tr>

<td>approverRole.enabled</td>
<td>

Create the ClusterRole to allow the issuer to approve certificate requests

</td>
<td>bool</td>
<td>

```yaml
true
```

</td>
</tr>
<tr>

<td>approverRole.serviceAccountName</td>
<td>

Service account give approval permission

</td>
<td>string</td>
<td>

```yaml
cert-manager
```

</td>
</tr>
<tr>

<td>approverRole.namespace</td>
<td>

Namespace the service account resides in

</td>
<td>string</td>
<td>

```yaml
cert-manager
```

</td>
</tr>
</table>

### Monitoring


<table>
<tr>
<th>Property</th>
<th>Description</th>
<th>Type</th>
<th>Default</th>
</tr>
<tr>

<td>serviceMonitor.create</td>
<td>

Create Prometheus ServiceMonitor

</td>
<td>bool</td>
<td>

```yaml
false
```

</td>
</tr>
<tr>

<td>serviceMonitor.annotations</td>
<td>

Annotations to add to the Prometheus ServiceMonitor

</td>
<td>object</td>
<td>

```yaml
{}
```

</td>
</tr>
<tr>

<td>serviceMonitor.labels</td>
<td>

Labels to add to the Prometheus ServiceMonitor

</td>
<td>object</td>
<td>

```yaml
{}
```

</td>
</tr>
</table>

<!-- /AUTO-GENERATED -->
