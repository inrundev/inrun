# providers

Declares external provider dependencies — cloud services, databases, or any system Orkestra integrates with via a provider plugin. Providers must be registered at operator startup.

```yaml
providers:
  - name: aws
    required: true
    version: v1.2.0
    auth:
      region: us-east-1
      roleArn: $AWS_ROLE_ARN
      accessKeyId: $AWS_ACCESS_KEY_ID
      secretAccessKey: $AWS_SECRET_ACCESS_KEY

  - name: mongodb
    required: false              # warn if not registered, don't fail
    auth:
      uri: $MONGODB_URI
```

## Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Provider identifier — must match a registered provider's name. |
| `required` | no | `true` — hard error if the provider is not registered at startup. `false` — log a warning and continue. |
| `auth` | no | Provider credentials. Map of key-value pairs. Values support `$ENV_VAR` expansion. |
| `version` | no | Expected provider library version. Checked at startup if set. |
| `library` | no | OCI artifact reference for the provider library. |

## How providers are used

Once declared at the Katalog level, provider blocks appear in `operatorBox`:

```yaml
operatorBox:
  reconcile:
    providers:
      aws:
        - action: createBucket
          input:
            name: "{{ .Name }}-data"
            region: "{{ .Spec.Region }}"
```

The provider declaration at the top level is the dependency claim. The provider block in `operatorBox` is where calls are made.

---
