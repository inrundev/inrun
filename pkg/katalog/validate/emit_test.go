package validate

import (
	"testing"

	orktypes "github.com/orkspace/orkestra/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func katalogWithEmit(crdName string, emit *orktypes.EmitConfig) *executor {
	return newKatalogExec(map[string]orktypes.CRDEntry{
		crdName: {OperatorBox: &orktypes.OperatorBoxConfig{
			Emit: emit,
		}},
	})
}

func TestValidateEmit_NilEmit(t *testing.T) {
	k := katalogWithEmit("myapp", nil)
	assert.NoError(t, k.validateEmit())
}

func TestValidateEmit_NoEvents(t *testing.T) {
	k := katalogWithEmit("myapp", &orktypes.EmitConfig{})
	assert.NoError(t, k.validateEmit())
}

func TestValidateEmit_ValidNormal(t *testing.T) {
	k := katalogWithEmit("myapp", &orktypes.EmitConfig{
		Events: map[string]*orktypes.EmitEventEntry{
			"DatabaseReady": {
				Type:    orktypes.EmitEventTypeNormal,
				Reason:  "Ready",
				Message: "{{ .spec.name }} is ready",
			},
		},
	})
	assert.NoError(t, k.validateEmit())
}

func TestValidateEmit_ValidWarning(t *testing.T) {
	k := katalogWithEmit("myapp", &orktypes.EmitConfig{
		Events: map[string]*orktypes.EmitEventEntry{
			"DatabaseFailed": {
				Type:    orktypes.EmitEventTypeWarning,
				Reason:  "SyncFailed",
				Message: "{{ .spec.name }} failed to sync",
			},
		},
	})
	assert.NoError(t, k.validateEmit())
}

func TestValidateEmit_InvalidType(t *testing.T) {
	k := katalogWithEmit("myapp", &orktypes.EmitConfig{
		Events: map[string]*orktypes.EmitEventEntry{
			"BadEvent": {
				Type:    "Critical",
				Reason:  "Oops",
				Message: "something went wrong",
			},
		},
	})
	err := k.validateEmit()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "type")
	assert.Contains(t, err.Error(), "Critical")
	assert.Contains(t, err.Error(), orktypes.EmitEventTypesJoined())
}

func TestValidateEmit_EmptyType(t *testing.T) {
	k := katalogWithEmit("myapp", &orktypes.EmitConfig{
		Events: map[string]*orktypes.EmitEventEntry{
			"NoType": {
				Reason:  "SomeReason",
				Message: "some message",
			},
		},
	})
	err := k.validateEmit()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "type")
}

func TestValidateEmit_EmptyReason(t *testing.T) {
	k := katalogWithEmit("myapp", &orktypes.EmitConfig{
		Events: map[string]*orktypes.EmitEventEntry{
			"NoReason": {
				Type:    orktypes.EmitEventTypeNormal,
				Message: "some message",
			},
		},
	})
	err := k.validateEmit()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reason must not be empty")
}

func TestValidateEmit_EmptyMessage(t *testing.T) {
	k := katalogWithEmit("myapp", &orktypes.EmitConfig{
		Events: map[string]*orktypes.EmitEventEntry{
			"NoMessage": {
				Type:   orktypes.EmitEventTypeWarning,
				Reason: "SyncFailed",
			},
		},
	})
	err := k.validateEmit()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "message must not be empty")
}

func TestValidateEmitEventEntry_DirectCall(t *testing.T) {
	err := validateEmitEventEntry("myapp", "TestEvent", &orktypes.EmitEventEntry{
		Type:    orktypes.EmitEventTypeNormal,
		Reason:  "Ready",
		Message: "ready",
	}, nil)
	assert.NoError(t, err)
}

func TestValidateEmit_InvalidMessageTemplate(t *testing.T) {
	k := katalogWithEmit("myapp", &orktypes.EmitConfig{
		Events: map[string]*orktypes.EmitEventEntry{
			"BadTemplate": {
				Type:    orktypes.EmitEventTypeNormal,
				Reason:  "Ready",
				Message: "{{ .spec.name",
			},
		},
	})
	err := k.validateEmit()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid template")
}

func TestValidateEmit_InvalidWhenFieldTemplate(t *testing.T) {
	k := katalogWithEmit("myapp", &orktypes.EmitConfig{
		Events: map[string]*orktypes.EmitEventEntry{
			"BadWhen": {
				Type:    orktypes.EmitEventTypeWarning,
				Reason:  "Failed",
				Message: "something failed",
				When: []orktypes.Condition{
					{Field: "{{ .status.phase"},
				},
			},
		},
	})
	err := k.validateEmit()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid template")
}

func TestValidateEmit_MultipleEntries_OneInvalid(t *testing.T) {
	k := katalogWithEmit("myapp", &orktypes.EmitConfig{
		Events: map[string]*orktypes.EmitEventEntry{
			"GoodEvent": {
				Type:    orktypes.EmitEventTypeNormal,
				Reason:  "Ready",
				Message: "{{ .spec.name }} is ready",
			},
			"BadEvent": {
				Type:    orktypes.EmitEventTypeWarning,
				Reason:  "Failed",
				Message: "",
			},
		},
	})
	err := k.validateEmit()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "message must not be empty")
}

func TestValidateEmit_WithWhenCondition(t *testing.T) {
	k := katalogWithEmit("myapp", &orktypes.EmitConfig{
		Events: map[string]*orktypes.EmitEventEntry{
			"PhaseReady": {
				Type:    orktypes.EmitEventTypeNormal,
				Reason:  "Ready",
				Message: "{{ .spec.name }} is ready",
				When: []orktypes.Condition{
					{Field: ".status.phase", Equals: "Ready"},
				},
			},
		},
	})
	assert.NoError(t, k.validateEmit())
}
