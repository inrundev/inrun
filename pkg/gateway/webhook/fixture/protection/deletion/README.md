# deletion-protection (webhook)

Verifies `security.deletionProtection`: a `ValidatingWebhookConfiguration`
intercepts `DELETE` on CRDs managed by the Catalog and rejects the request
before the API server removes it. `LogStream` (`crd-unprotected.yaml`) is
deliberately left out of the Catalog as the contrast case — it deletes freely.

The housekeeper recreates the `ValidatingWebhookConfiguration` immediately if
it's deleted directly, so the only clean way to unblock cleanup in the test
is to push `security.deletionProtection.enabled: false` via the catalog
ConfigMap and restart the gateway — the fixture does exactly that as its
last step before asserting cleanup.

## Run

```sh
inrun e2e pkg/gateway/webhook/fixture/protection/deletion/e2e.yaml
```
