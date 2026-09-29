package validate

import (
	"testing"

	"github.com/orkspace/orkestra/domain"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func katalogWithQueue(crdName string, q *orktypes.Queue) *executor {
	if q.Empty() {
		q = &orktypes.Queue{}
	}
	return newKatalogExec(map[string]orktypes.CRDEntry{
		crdName: {OperatorBox: &orktypes.OperatorBoxConfig{
			Reconcile: &orktypes.ReconcileConfig{Queue: *q},
		}},
	})
}

func TestValidateQueue_Nil(t *testing.T) {
	k := katalogWithQueue("app", nil)
	assert.NoError(t, k.validateQueue())
}

func TestValidateQueue_OnLimitValid(t *testing.T) {
	k := katalogWithQueue("app", &orktypes.Queue{
		MaxDepth: 100,
		Cfg: &orktypes.QueueBehaviour{
			OnLimit: &orktypes.QueueBehaviourSetting{
				Drop: boolPtr(true),
			},
		},
	})
	assert.NoError(t, k.validateQueue())
}

func TestValidateQueue_UnLimitedValid(t *testing.T) {
	k := katalogWithQueue("app", &orktypes.Queue{MaxDepth: 0})
	assert.NoError(t, k.validateQueue())
}

func TestValidateQueue_LimitedValid(t *testing.T) {
	k := katalogWithQueue("app", &orktypes.Queue{
		MaxDepth: 100,
		Cfg: &orktypes.QueueBehaviour{
			OnLimit: &orktypes.QueueBehaviourSetting{
				Drop: boolPtr(true),
			},
		},
	})
	assert.NoError(t, k.validateQueue())
}

func TestValidateQueue_UnLimitedInValid(t *testing.T) {
	k := katalogWithQueue("app", &orktypes.Queue{
		MaxDepth: 0,
		Cfg: &orktypes.QueueBehaviour{
			OnLimit: &orktypes.QueueBehaviourSetting{
				Drop: boolPtr(true),
			},
		},
	})

	err := k.validateQueue()
	require.Error(t, err)
	assert.ErrorContains(t, err, "'queue.behaviour' configuration is only valid when 'queue.maxDepth' is greater than 0")
}

func TestValidateQueue_OnLimitDropValid(t *testing.T) {
	k := katalogWithQueue("app", &orktypes.Queue{
		MaxDepth: 100,
		Cfg: &orktypes.QueueBehaviour{
			OnLimit: &orktypes.QueueBehaviourSetting{
				Drop: boolPtr(false),
			},
		},
	})
	assert.NoError(t, k.validateQueue())
}

func TestValidateQueue_OnLimitWithValueInValid(t *testing.T) {
	k := katalogWithQueue("app", &orktypes.Queue{
		MaxDepth: 100,
		Cfg: &orktypes.QueueBehaviour{
			OnLimit: &orktypes.QueueBehaviourSetting{
				Drop:  boolPtr(true),
				Value: 5,
			},
		},
	})

	err := k.validateQueue()
	require.Error(t, err)
	assert.ErrorContains(t, err, "is not allowed in onLimit configuration")
}

func TestValidateQueue_onThresholdWarning(t *testing.T) {
	k := katalogWithQueue("app", &orktypes.Queue{
		MaxDepth: 100,
		Cfg: &orktypes.QueueBehaviour{
			OnThreshold: &orktypes.QueueBehaviourSetting{
				Drop:  boolPtr(false),
				Value: 5,
			},
		},
	})

	err := k.validateQueue()
	assert.NoError(t, err)

	entry := k.k.EnabledCRDs()["app"]

	if !entry.Warnings.HasWarnings() {
		t.Fatal("expected crd to have warning")
	}

	// Check if drop is set correctly
	cfg := entry.QueueConfig().Behaviour()
	// got := cfg.onThreshold.ShouldDrop()
	got := *cfg.OnThreshold.Drop

	if !got {
		t.Fatalf("expected onThreshold.drop to be true. got: %v", got)
	}
}

// ── Remote reconciler validation ─────────────────────────────────────────────

func remoteResources() []domain.ManagedResource {
	return []domain.ManagedResource{{Group: "apps", Version: "v1", Plural: "deployments"}}
}

func katalogWithRemote(endpoint string, protocol orktypes.RemoteReconcileProtocol, resources []domain.ManagedResource) *executor {
	return newKatalogExec(map[string]orktypes.CRDEntry{
		"app": {OperatorBox: &orktypes.OperatorBoxConfig{
			Reconcile: &orktypes.ReconcileConfig{
				Remote: &orktypes.RemoteReconcilerDeclaration{
					Protocol:         protocol,
					Endpoint:         endpoint,
					ManagedResources: resources,
				},
			},
		}},
	})
}

func TestValidateRemote_Valid(t *testing.T) {
	k := katalogWithRemote("https://svc.internal/reconcile", orktypes.RemoteReconcileProtocolHTTP, remoteResources())
	require.NoError(t, k.validateRemote("app", crdPtr(k, "app")))
}

func TestValidateRemote_NoEndpoint(t *testing.T) {
	k := katalogWithRemote("", orktypes.RemoteReconcileProtocolHTTP, remoteResources())
	err := k.validateRemote("app", crdPtr(k, "app"))
	assert.ErrorContains(t, err, "endpoint is required")
}

func TestValidateRemote_InvalidType(t *testing.T) {
	k := katalogWithRemote("https://svc.internal/reconcile", "websocket", remoteResources())
	err := k.validateRemote("app", crdPtr(k, "app"))
	assert.ErrorContains(t, err, "not valid")
}

func TestValidateRemote_NoManagedResources_Info(t *testing.T) {
	k := katalogWithRemote("https://svc.internal/reconcile", orktypes.RemoteReconcileProtocolHTTP, nil)
	crd := crdPtr(k, "app")
	require.NoError(t, k.validateManagedResources("app", crd))
	assert.True(t, crd.Info.HasInfo())
	assert.True(t, crd.Info.Contains("RBAC will not be generated"))
}

func TestValidateRemote_ConflictsWithHooks(t *testing.T) {
	crd := orktypes.CRDEntry{OperatorBox: &orktypes.OperatorBoxConfig{
		Reconcile: &orktypes.ReconcileConfig{
			Hooks: &orktypes.HookDeclaration{Location: "github.com/test/rec", Function: "New"},
			Remote: &orktypes.RemoteReconcilerDeclaration{
				Endpoint:         "https://svc.internal/reconcile",
				ManagedResources: remoteResources(),
			},
		},
	}}
	k := katalogWith(map[string]orktypes.CRDEntry{"app": crd})
	err := k.validateRemote("app", crdPtr(k, "app"))
	assert.ErrorContains(t, err, "cannot declare both")
}

func TestValidateRemote_ConflictsWithConstructor(t *testing.T) {
	crd := orktypes.CRDEntry{OperatorBox: &orktypes.OperatorBoxConfig{
		Reconcile: &orktypes.ReconcileConfig{
			ConstructorDecl: &orktypes.ConstructorDeclaration{Location: "github.com/test/rec", Function: "New"},
			Remote: &orktypes.RemoteReconcilerDeclaration{
				Endpoint:         "https://svc.internal/reconcile",
				ManagedResources: remoteResources(),
			},
		},
	}}
	k := katalogWith(map[string]orktypes.CRDEntry{"app": crd})
	err := k.validateRemote("app", crdPtr(k, "app"))
	assert.ErrorContains(t, err, "cannot declare both")
}

func TestValidateRemote_TypeOmitted_Valid(t *testing.T) {
	k := katalogWithRemote("https://svc.internal/reconcile", "", remoteResources())
	require.NoError(t, k.validateRemote("app", crdPtr(k, "app")))
}

func TestValidateRemote_TemplateEndpoint_Valid(t *testing.T) {
	k := katalogWithRemote("http://{{ .metadata.namespace }}-svc.svc.cluster.local/reconcile", orktypes.RemoteReconcileProtocolHTTP, remoteResources())
	require.NoError(t, k.validateRemote("app", crdPtr(k, "app")))
}

func TestValidateRemote_TemplateEndpoint_InvalidTemplate(t *testing.T) {
	k := katalogWithRemote("http://{{ .metadata.namespace }-svc/reconcile", orktypes.RemoteReconcileProtocolHTTP, remoteResources())
	err := k.validateRemote("app", crdPtr(k, "app"))
	assert.ErrorContains(t, err, "invalid template")
}

func TestValidateRemote_Args_ValidTemplate(t *testing.T) {
	k := newKatalogExec(map[string]orktypes.CRDEntry{
		"app": {OperatorBox: &orktypes.OperatorBoxConfig{
			Reconcile: &orktypes.ReconcileConfig{
				Remote: &orktypes.RemoteReconcilerDeclaration{
					Endpoint:         "https://svc.internal/reconcile",
					ManagedResources: remoteResources(),
					Args:             map[string]interface{}{"replicas": "{{ .spec.replicas }}"},
				},
			},
		}},
	})
	require.NoError(t, k.validateRemote("app", crdPtr(k, "app")))
}

func TestValidateRemote_Args_InvalidTemplate(t *testing.T) {
	k := newKatalogExec(map[string]orktypes.CRDEntry{
		"app": {OperatorBox: &orktypes.OperatorBoxConfig{
			Reconcile: &orktypes.ReconcileConfig{
				Remote: &orktypes.RemoteReconcilerDeclaration{
					Endpoint:         "https://svc.internal/reconcile",
					ManagedResources: remoteResources(),
					Args:             map[string]interface{}{"replicas": "{{ .spec.replicas }"},
				},
			},
		}},
	})
	err := k.validateRemote("app", crdPtr(k, "app"))
	assert.ErrorContains(t, err, "invalid template")
}

func TestValidateRemote_DefaultFalse_Valid(t *testing.T) {
	k := newKatalogExec(map[string]orktypes.CRDEntry{
		"app": {OperatorBox: &orktypes.OperatorBoxConfig{
			Reconcile: &orktypes.ReconcileConfig{
				Default: boolPtr(false),
				Remote: &orktypes.RemoteReconcilerDeclaration{
					Endpoint:         "https://svc.internal/reconcile",
					ManagedResources: remoteResources(),
				},
			},
		}},
	})
	crd := crdPtr(k, "app")
	require.NoError(t, k.validateRemote("app", crd))
	require.NoError(t, k.validateManagedResources("app", crd))
}

// ── Remote auth validation ────────────────────────────────────────────────────

func remoteWithAuth(auth *orktypes.ExternalAuth) *executor {
	return newKatalogExec(map[string]orktypes.CRDEntry{
		"app": {OperatorBox: &orktypes.OperatorBoxConfig{
			Reconcile: &orktypes.ReconcileConfig{
				Remote: &orktypes.RemoteReconcilerDeclaration{
					Endpoint: "https://svc.internal/reconcile",
					Auth:     auth,
				},
			},
		}},
	})
}

func TestValidateRemote_Auth_SecretRef_Valid(t *testing.T) {
	k := remoteWithAuth(&orktypes.ExternalAuth{
		SecretRef: &orktypes.APISecretRef{Name: "my-token", Key: "token", Namespace: "default"},
	})
	require.NoError(t, k.validateRemote("app", crdPtr(k, "app")))
}

func TestValidateRemote_Auth_SecretRef_MissingName(t *testing.T) {
	k := remoteWithAuth(&orktypes.ExternalAuth{
		SecretRef: &orktypes.APISecretRef{Key: "token", Namespace: "default"},
	})
	err := k.validateRemote("app", crdPtr(k, "app"))
	assert.ErrorContains(t, err, "name")
}

func TestValidateRemote_Auth_SecretRef_MissingKey(t *testing.T) {
	k := remoteWithAuth(&orktypes.ExternalAuth{
		SecretRef: &orktypes.APISecretRef{Name: "my-token", Namespace: "default"},
	})
	err := k.validateRemote("app", crdPtr(k, "app"))
	assert.ErrorContains(t, err, "key")
}

func TestValidateRemote_Auth_SecretRef_NoNamespace_Warning(t *testing.T) {
	k := remoteWithAuth(&orktypes.ExternalAuth{
		SecretRef: &orktypes.APISecretRef{Name: "my-token", Key: "token"},
	})
	crd := crdPtr(k, "app")
	require.NoError(t, k.validateRemote("app", crd))
	assert.True(t, crd.Warnings.Contains("namespace is empty"))
}

func TestValidateRemote_Auth_Env_NoValidation(t *testing.T) {
	k := remoteWithAuth(&orktypes.ExternalAuth{Env: "RECONCILER_TOKEN"})
	require.NoError(t, k.validateRemote("app", crdPtr(k, "app")))
}

// crdPtr returns a pointer to the named CRD entry from the executor's katalog.
func crdPtr(k *executor, name string) *orktypes.CRDEntry {
	e := k.k.EnabledCRDs()[name]
	return &e
}

// ── Remote payload validation ─────────────────────────────────────────────────

// remoteResourcesWithKind returns managed resources with Kind set so LookupBuiltIn resolves them.
func remoteResourcesWithKind() []domain.ManagedResource {
	return []domain.ManagedResource{
		{Kind: "Deployment", Group: "apps", Version: "v1", Plural: "deployments"},
		{Kind: "Service", Group: "", Version: "v1", Plural: "services"},
	}
}

func katalogWithRemotePayload(payload *orktypes.RemotePayloadConfig) *executor {
	return newKatalogExec(map[string]orktypes.CRDEntry{
		"app": {OperatorBox: &orktypes.OperatorBoxConfig{
			Reconcile: &orktypes.ReconcileConfig{
				Remote: &orktypes.RemoteReconcilerDeclaration{
					Endpoint:         "https://svc.internal/reconcile",
					ManagedResources: remoteResourcesWithKind(),
					Payload:          payload,
				},
			},
		}},
	})
}

func TestValidateRemotePayload_Nil(t *testing.T) {
	k := katalogWithRemotePayload(nil)
	require.NoError(t, k.validateRemote("app", crdPtr(k, "app")))
}

func TestValidateRemotePayload_ChildrenDisabled_SkipsChildrenValidation(t *testing.T) {
	k := katalogWithRemotePayload(&orktypes.RemotePayloadConfig{
		Children: &orktypes.RemotePayloadChildrenConfig{
			Enabled: boolPtr(false),
			// invalid paths that would normally error — should be skipped
			Resources: map[string]*orktypes.RemotePayloadChildrenResourceConfig{
				"nonexistent": {},
			},
		},
	})
	require.NoError(t, k.validateRemote("app", crdPtr(k, "app")))
}

func TestValidateRemotePayload_ObjectExclude_Valid(t *testing.T) {
	k := katalogWithRemotePayload(&orktypes.RemotePayloadConfig{
		Object: &orktypes.RemotePayloadObjectConfig{
			Exclude: []string{"metadata.managedFields", "metadata.annotations"},
		},
	})
	require.NoError(t, k.validateRemote("app", crdPtr(k, "app")))
}

func TestValidateRemotePayload_ObjectExclude_InvalidPath(t *testing.T) {
	k := katalogWithRemotePayload(&orktypes.RemotePayloadConfig{
		Object: &orktypes.RemotePayloadObjectConfig{
			Exclude: []string{"metadata..managedFields"},
		},
	})
	err := k.validateRemote("app", crdPtr(k, "app"))
	assert.ErrorContains(t, err, "payload.object.exclude")
	assert.ErrorContains(t, err, "empty segment")
}

func TestValidateRemotePayload_ObjectExclude_LeadingDot(t *testing.T) {
	k := katalogWithRemotePayload(&orktypes.RemotePayloadConfig{
		Object: &orktypes.RemotePayloadObjectConfig{
			Exclude: []string{".metadata.managedFields"},
		},
	})
	err := k.validateRemote("app", crdPtr(k, "app"))
	assert.ErrorContains(t, err, "payload.object.exclude")
}

func TestValidateRemotePayload_Children_RootExclude_Valid(t *testing.T) {
	k := katalogWithRemotePayload(&orktypes.RemotePayloadConfig{
		Children: &orktypes.RemotePayloadChildrenConfig{
			Exclude: []string{"metadata.managedFields"},
		},
	})
	require.NoError(t, k.validateRemote("app", crdPtr(k, "app")))
}

func TestValidateRemotePayload_Children_RootExclude_InvalidPath(t *testing.T) {
	k := katalogWithRemotePayload(&orktypes.RemotePayloadConfig{
		Children: &orktypes.RemotePayloadChildrenConfig{
			Exclude: []string{"spec.template..annotations"},
		},
	})
	err := k.validateRemote("app", crdPtr(k, "app"))
	assert.ErrorContains(t, err, "payload.children.exclude")
	assert.ErrorContains(t, err, "empty segment")
}

func TestValidateRemotePayload_Children_ResourcesFilter_Valid(t *testing.T) {
	k := katalogWithRemotePayload(&orktypes.RemotePayloadConfig{
		Children: &orktypes.RemotePayloadChildrenConfig{
			Resources: map[string]*orktypes.RemotePayloadChildrenResourceConfig{
				"deployment": {},
				"service":    nil,
			},
		},
	})
	require.NoError(t, k.validateRemote("app", crdPtr(k, "app")))
}

func TestValidateRemotePayload_Children_ResourceNotInManaged(t *testing.T) {
	k := katalogWithRemotePayload(&orktypes.RemotePayloadConfig{
		Children: &orktypes.RemotePayloadChildrenConfig{
			Resources: map[string]*orktypes.RemotePayloadChildrenResourceConfig{
				"configmap": {},
			},
		},
	})
	err := k.validateRemote("app", crdPtr(k, "app"))
	assert.ErrorContains(t, err, "payload.children.resources")
	assert.ErrorContains(t, err, "configmap")
	assert.ErrorContains(t, err, "not declared in reconcile.remote.resources")
}

func TestValidateRemotePayload_Children_PerResourceExclude_Valid(t *testing.T) {
	k := katalogWithRemotePayload(&orktypes.RemotePayloadConfig{
		Children: &orktypes.RemotePayloadChildrenConfig{
			Resources: map[string]*orktypes.RemotePayloadChildrenResourceConfig{
				"deployment": {Exclude: []string{"spec.template.metadata.annotations"}},
			},
		},
	})
	require.NoError(t, k.validateRemote("app", crdPtr(k, "app")))
}

func TestValidateRemotePayload_Children_PerResourceExclude_InvalidPath(t *testing.T) {
	k := katalogWithRemotePayload(&orktypes.RemotePayloadConfig{
		Children: &orktypes.RemotePayloadChildrenConfig{
			Resources: map[string]*orktypes.RemotePayloadChildrenResourceConfig{
				"deployment": {Exclude: []string{"spec.template."}},
			},
		},
	})
	err := k.validateRemote("app", crdPtr(k, "app"))
	assert.ErrorContains(t, err, `resources["deployment"].exclude`)
}
