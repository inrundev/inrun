//go:build ignore

// reconciler/webapp_reconciler.go
//
// The ConstructorApp reconciler lifted from controller-runtime into Orkestra.
//
// Compare with 00-controller-runtime-baseline/controller/webapp_controller.go.
// The reconcile logic is identical — same Deployment spec, same Service spec,
// same status patch, same IsNotFound guard. The only change is the signature:
//
//	Before: Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)
//	After:  Reconcile(ctx context.Context, key string) error
//
// key is namespace/name — the same string as req.String(). Everything inside
// the method is unchanged.
//
// What was removed (owned by Orkestra now):
//   - ctrl.NewManager and all setup in main.go
//   - ctrl.NewControllerManagedBy / SetupWithManager
//   - scheme registration
//   - ctrl.Result retry semantics (return nil = done, return error = requeue)
//
// Orkestra provides (without you writing any of it):
//   - Informer watching the ConstructorApp CRD
//   - Workqueue with deduplication and backoff
//   - Worker pool (configurable in Katalog: workers: N)
//   - safeReconcile panic recovery
//   - Prometheus metrics (reconcile total, duration, queue depth)
//   - Per-CRD health tracking
//   - Leader election
//
// You own (same as before):
//   - Reading objects from the informer cache
//   - Finalizer management
//   - Kubernetes events
//   - Status updates
//   - All reconcile logic
package reconciler

import (
	"context"
	"fmt"

	apiv1 "github.com/orkspace/from-controller-runtime-all-options/options/constructor/api/v1alpha1"
	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	sigs "sigs.k8s.io/controller-runtime/pkg/client"
)

// WebAppReconciler implements domain.Reconciler for the ConstructorApp CRD.
type WebAppReconciler struct {
	kube kubeclient.Interface
}

// NewWebAppReconciler is the constructor function registered in the Katalog.
func NewWebAppReconciler(kube kubeclient.Interface) domain.Reconciler {
	return &WebAppReconciler{kube: kube}
}

// Reconcile is called by Orkestra's worker pool for every queued ConstructorApp key.
func (r *WebAppReconciler) Reconcile(ctx context.Context, req domain.Request) (domain.Result, error) {
	if req.Prepared == nil {
		return domain.Result{}, nil
	}

	webapp, err := domain.ToTyped[apiv1.ConstructorApp](req.Prepared)
	if err != nil {
		return domain.Result{}, fmt.Errorf("toTyped: %w", err)
	}

	if webapp.DeletionTimestamp != nil {
		// Owner references clean up Deployment and Service automatically.
		return domain.Result{}, nil
	}

	if err := r.reconcileDeployment(ctx, webapp); err != nil {
		return domain.Result{}, err
	}
	if err := r.reconcileService(ctx, webapp); err != nil {
		return domain.Result{}, err
	}

	r.kube.GetEventRecorder().Eventf(webapp, corev1.EventTypeNormal, "WebAppReconciled",
		"ConstructorApp %s/%s reconciled", webapp.Namespace, webapp.Name)

	return domain.Result{}, r.kube.PatchStatus(ctx, webapp, map[string]interface{}{
		"phase":    "Running",
		"endpoint": fmt.Sprintf("%s-svc.%s.svc.cluster.local", webapp.Name, webapp.Namespace),
		"replicas": webapp.Spec.Replicas,
	}, metav1.PatchOptions{})
}

// reconcileDeployment — same logic as the controller-runtime baseline.
// StrategicMergeFrom is used here because Deployment's container list carries
// patchMergeKey:"name" annotations — the API server merges containers by name
// rather than replacing the list wholesale. sigs.StrategicMergeFrom works here too.
func (r *WebAppReconciler) reconcileDeployment(ctx context.Context, webapp *apiv1.ConstructorApp) error {
	replicas := webapp.Spec.Replicas
	desired := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webapp.Name,
			Namespace: webapp.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(webapp, apiv1.GroupVersionKind),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": webapp.Name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": webapp.Name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  webapp.Name,
							Image: webapp.Spec.Image,
							Ports: []corev1.ContainerPort{
								{ContainerPort: webapp.Spec.Port},
							},
						},
					},
				},
			},
		},
	}

	existing := &appsv1.Deployment{}
	err := r.kube.Get(ctx, domain.ObjectKey{Namespace: webapp.Namespace, Name: webapp.Name}, existing, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return r.kube.Create(ctx, desired, metav1.CreateOptions{})
	}
	if err != nil {
		return err
	}
	patch := sigs.StrategicMergeFrom(existing.DeepCopy())
	existing.Spec = desired.Spec
	return r.kube.Patch(ctx, existing, patch, metav1.PatchOptions{})
}

// reconcileService — same logic as the controller-runtime baseline.
// MergeFrom (JSON merge patch) is correct here — Service ports have no strategic
// merge key, so replace semantics are what the API server applies anyway.
// sigs.MergeFrom works here too.
func (r *WebAppReconciler) reconcileService(ctx context.Context, webapp *apiv1.ConstructorApp) error {
	desired := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webapp.Name + "-svc",
			Namespace: webapp.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(webapp, apiv1.GroupVersionKind),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": webapp.Name},
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(int(webapp.Spec.Port)),
				},
			},
		},
	}

	existing := &corev1.Service{}
	err := r.kube.Get(ctx, domain.ObjectKey{Namespace: webapp.Namespace, Name: webapp.Name + "-svc"}, existing, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return r.kube.Create(ctx, desired, metav1.CreateOptions{})
	}
	if err != nil {
		return err
	}
	patch := sigs.MergeFrom(existing.DeepCopy())
	existing.Spec.Ports = desired.Spec.Ports
	return r.kube.Patch(ctx, existing, patch, metav1.PatchOptions{})
}
