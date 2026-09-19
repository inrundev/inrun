//go:build ignore

// hooks/database_hooks.go
//
// Typed Go hooks for the Database CRD.
//
// This hook runs inside the Orkestra reconcile loop. Orkestra still provides:
//   - Informer watching the Database CRD
//   - Workqueue with deduplication and backoff
//   - Worker pool
//   - Finalizer management
//   - Kubernetes events
//   - Status management (Layer 1 Ready condition)
//   - Prometheus metrics
//
// The hook provides:
//   - Type-safe struct access (obj.Spec.Engine, not unstructured map navigation)
//   - The business logic that cannot be expressed in templates
//   - OrkestraRegistry calls for Kubernetes child resources
package hooks

import (
	"context"
	"fmt"

	apiv1 "github.com/orkspace/orkestra-mixed-operator-pattern/09-hooks/api/v1alpha1"
	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	orkcron "github.com/orkspace/orkestra/pkg/resources/cronjobs"
	orksvc "github.com/orkspace/orkestra/pkg/resources/services"
	orkstatefulset "github.com/orkspace/orkestra/pkg/resources/statefulsets"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

const (
	// Test values
	postgresUser     = "postgres"
	postgresPassword = "postgres-password"
	defaultReplicas  = "1"
)

// DatabaseHooks returns the hook implementation for the Database CRD.
// Registered in the Katalog under reconciler.hooks.function.
func DatabaseHooks() domain.AnyReconcileHooks {
	return domain.ReconcileHooks[*apiv1.Database]{
		OnReconcile: onDatabaseReconcile,
		OnDelete:    onDatabaseDelete,
	}
}

// onDatabaseReconcile runs on every reconcile cycle for a Database CR.
// It creates the StatefulSet, Service, and optionally a backup CronJob.
func onDatabaseReconcile(ctx context.Context, obj *apiv1.Database) error {
	kube, ok := kubeclient.FromContext(ctx)
	if !ok {
		return fmt.Errorf("kubeclient not in context")
	}

	// ── Type-safe access to spec fields ───────────────────────────────────
	// This is why typed hooks exist: obj.Spec.Engine, not
	// unstructured map navigation with type assertions.
	engine := obj.Spec.Engine
	version := obj.Spec.Version
	storage := obj.Spec.Storage
	image := engineImage(engine, version)

	// ── Create the StatefulSet via OrkestraRegistry ────────────────────────
	// OrkestraRegistry handles: owner references, idempotency, system labels.
	// The hook only provides the resolved spec.
	spec := orkstatefulset.Resolve(
		orktypes.StatefulSetTemplateSource{
			Name:               obj.Name,
			Namespace:          obj.Namespace,
			Image:              image,
			Replicas:           defaultReplicas,
			Port:               dbPort(engine),
			ServiceAccountName: obj.Name + "-sa", // declared in the Katalog onCreate
			Env: []orktypes.EnvVar{
				{Name: "POSTGRES_USER", Value: postgresUser},
				{Name: "POSTGRES_PASSWORD", Value: postgresPassword},
			},
			Labels: orktypes.Labels{
				"db-engine":    engine,
				"db-version":   version,
				"storage-size": storage,
			},
		},
		obj.Name,
		nil,
	)
	if err := orkstatefulset.Update(ctx, kube, obj, spec); err != nil {
		return fmt.Errorf("database statefulSet: %w", err)
	}

	// ── Create the Service ─────────────────────────────────────────────────
	svcSpec := orksvc.Resolve(
		orktypes.ServiceTemplateSource{
			Name:       obj.Name + "-svc",
			Namespace:  obj.Namespace,
			Port:       dbPort(engine),
			TargetPort: dbPort(engine),
			Type:       "ClusterIP",
		},
		obj.Name,
	)
	if err := orksvc.Update(ctx, kube, obj, svcSpec); err != nil {
		return fmt.Errorf("database service: %w", err)
	}

	// ── Conditional: backup CronJob ────────────────────────────────────────
	// This conditional is in Go rather than a when: block because the
	// logic depends on obj.Spec.Backup — a boolean field that when:
	// conditions can handle, but the CronJob spec also varies by engine.
	if obj.Spec.Backup {
		cronSpec := orkcron.Resolve(
			orktypes.CronJobTemplateSource{
				Name:      obj.Name + "-backup",
				Namespace: obj.Namespace,
				Schedule:  "0 2 * * *", // daily at 2am
				Image:     backupImage(engine),
				Command:   []string{"/bin/backup.sh"},
				Args:      []string{obj.Name, obj.Namespace},
			},
			obj.Name,
			nil,
		)
		if err := orkcron.Update(ctx, kube, obj, cronSpec); err != nil {
			return fmt.Errorf("database backup cronjob: %w", err)
		}
	}

	return nil
}

// onDatabaseDelete runs when a Database CR is being deleted.
// Owner references handle StatefulSet, Service, and CronJob cleanup automatically.
// This hook handles external cleanup — nothing in this example, but the
// pattern is here for operators that create external resources.
func onDatabaseDelete(ctx context.Context, obj *apiv1.Database) error {
	// For this example the database is purely in-cluster.
	// In a real external provisioner, you would:
	//   return externalDB.Delete(ctx, obj.Spec.InstanceID)
	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────

func engineImage(engine, version string) string {
	if version == "" {
		version = "latest"
	}
	switch engine {
	case "mysql":
		return fmt.Sprintf("mysql:%s", version)
	default:
		return fmt.Sprintf("postgres:%s", version)
	}
}

func dbPort(engine string) string {
	if engine == "mysql" {
		return "3306"
	}
	return "5432"
}

func backupImage(engine string) string {
	if engine == "mysql" {
		return "myorg/mysql-backup:latest"
	}
	return "myorg/postgres-backup:latest"
}
