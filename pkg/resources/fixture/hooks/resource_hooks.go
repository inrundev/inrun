// hooks/resource_hooks.go
//
// Typed Go hooks for the ResourceProbe CRD (group: resources.orkestra.io).
//
// Purpose: exercise every pkg/resources/* Resolve() + Update() (or Create())
// code path in a single reconcile loop.
//
// Coverage map:
//
//	namespaces, configmaps, secrets, serviceaccounts (Create), clusterroles,
//	clusterrolebindings, roles, rolebindings, networkpolicies, resourcequotas,
//	limitranges, deployments, statefulsets, replicasets, pods, jobs (Create),
//	cronjobs, services, ingresses, hpas, pdbs, pvcs, pvs
package hooks

import (
	"context"
	"fmt"

	rpv1alpha1 "github.com/orkspace/orkestra-resource-probe/api/v1alpha1"
	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/kubeclient"
	orkcrb "github.com/orkspace/orkestra/pkg/resources/clusterrolebindings"
	orkcr "github.com/orkspace/orkestra/pkg/resources/clusterroles"
	orkconfigmap "github.com/orkspace/orkestra/pkg/resources/configmaps"
	orkcronjob "github.com/orkspace/orkestra/pkg/resources/cronjobs"
	orkdeployment "github.com/orkspace/orkestra/pkg/resources/deployments"
	orkhpa "github.com/orkspace/orkestra/pkg/resources/hpas"
	orkingress "github.com/orkspace/orkestra/pkg/resources/ingresses"
	orkjob "github.com/orkspace/orkestra/pkg/resources/jobs"
	orklimitrange "github.com/orkspace/orkestra/pkg/resources/limitranges"
	orknamespace "github.com/orkspace/orkestra/pkg/resources/namespaces"
	orknetpol "github.com/orkspace/orkestra/pkg/resources/networkpolicies"
	orkpdb "github.com/orkspace/orkestra/pkg/resources/pdbs"
	orkpod "github.com/orkspace/orkestra/pkg/resources/pods"
	orkpvc "github.com/orkspace/orkestra/pkg/resources/pvcs"
	orkpv "github.com/orkspace/orkestra/pkg/resources/pvs"
	orkreplicaset "github.com/orkspace/orkestra/pkg/resources/replicasets"
	orkquota "github.com/orkspace/orkestra/pkg/resources/resourcequotas"
	orkrb "github.com/orkspace/orkestra/pkg/resources/rolebindings"
	orkrole "github.com/orkspace/orkestra/pkg/resources/roles"
	orksecret "github.com/orkspace/orkestra/pkg/resources/secrets"
	orksa "github.com/orkspace/orkestra/pkg/resources/serviceaccounts"
	orksvc "github.com/orkspace/orkestra/pkg/resources/services"
	orkstatefulset "github.com/orkspace/orkestra/pkg/resources/statefulsets"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// ResourceProbeHooks returns the hook implementation for the ResourceProbe CRD.
// Registered in the Katalog under operatorBox.reconciler.hooks.function.
func ResourceProbeHooks() domain.AnyReconcileHooks {
	return domain.ReconcileHooks[*rpv1alpha1.ResourceProbe]{
		OnReconcile: onReconcile,
		OnDelete:    onDelete,
	}
}

func onReconcile(ctx context.Context, obj *rpv1alpha1.ResourceProbe) error {
	kube, ok := kubeclient.FromContext(ctx)
	if !ok {
		return fmt.Errorf("kubeclient not in context")
	}

	var reg *orktypes.ProfileRegistry
	ns := obj.Name + "-ns"

	// ── Namespace ─────────────────────────────────────────────────────────────
	nsSpec := orknamespace.Resolve(orktypes.NamespaceTemplateSource{
		Name: ns,
	}, obj.Name)
	if err := orknamespace.Update(ctx, kube, obj, nsSpec); err != nil {
		return fmt.Errorf("namespace: %w", err)
	}

	// ── ConfigMap ─────────────────────────────────────────────────────────────
	cmSpec := orkconfigmap.Resolve(orktypes.ConfigMapTemplateSource{
		Name:      obj.Name + "-config",
		Namespace: ns,
		Data: map[string]string{
			"IMAGE":    obj.Spec.Image,
			"PORT":     obj.Spec.Port,
			"REPLICAS": obj.Spec.Replicas,
		},
	}, obj.Name)
	if err := orkconfigmap.Update(ctx, kube, obj, cmSpec); err != nil {
		return fmt.Errorf("configmap: %w", err)
	}

	// ── Secret ────────────────────────────────────────────────────────────────
	secretSpec := orksecret.Resolve(orktypes.SecretTemplateSource{
		Name:      obj.Name + "-creds",
		Namespace: ns,
		Data: map[string]string{
			"token": "probe-token",
		},
	}, obj.Name)
	if err := orksecret.Update(ctx, kube, obj, secretSpec); err != nil {
		return fmt.Errorf("secret: %w", err)
	}

	// ── ServiceAccount (Create — no Update) ───────────────────────────────────
	saSpec := orksa.Resolve(orktypes.ServiceAccountTemplateSource{
		Name:      obj.Name + "-sa",
		Namespace: ns,
	}, obj.Name)
	if err := orksa.Create(ctx, kube, obj, saSpec); err != nil {
		return fmt.Errorf("serviceaccount: %w", err)
	}

	// ── ClusterRole ───────────────────────────────────────────────────────────
	crSpec := orkcr.Resolve(orktypes.ClusterRoleTemplateSource{
		Name: obj.Name + "-cr",
		Rules: []orktypes.PolicyRuleSpec{
			{APIGroups: []string{""}, Resources: []string{"namespaces"}, Verbs: []string{"get", "list", "watch"}},
		},
	}, obj.Name)
	if err := orkcr.Update(ctx, kube, obj, crSpec); err != nil {
		return fmt.Errorf("clusterrole: %w", err)
	}

	// ── ClusterRoleBinding ────────────────────────────────────────────────────
	crbSpec := orkcrb.Resolve(orktypes.ClusterRoleBindingTemplateSource{
		Name: obj.Name + "-crb",
		RoleRef: orktypes.RoleRefSpec{
			Kind: "ClusterRole",
			Name: obj.Name + "-cr",
		},
		Subjects: []orktypes.SubjectSpec{
			{Kind: "ServiceAccount", Name: obj.Name + "-sa", Namespace: ns},
		},
	}, obj.Name)
	if err := orkcrb.Update(ctx, kube, obj, crbSpec); err != nil {
		return fmt.Errorf("clusterrolebinding: %w", err)
	}

	// ── Role ──────────────────────────────────────────────────────────────────
	roleSpec := orkrole.Resolve(orktypes.RoleTemplateSource{
		Name:      obj.Name + "-role",
		Namespace: ns,
		Rules: []orktypes.PolicyRuleSpec{
			{APIGroups: []string{""}, Resources: []string{"configmaps", "secrets"}, Verbs: []string{"get", "list"}},
		},
	}, obj.Name)
	if err := orkrole.Update(ctx, kube, obj, roleSpec); err != nil {
		return fmt.Errorf("role: %w", err)
	}

	// ── RoleBinding ───────────────────────────────────────────────────────────
	rbSpec := orkrb.Resolve(orktypes.RoleBindingTemplateSource{
		Name:      obj.Name + "-rb",
		Namespace: ns,
		RoleRef:   orktypes.RoleRefSpec{Kind: "Role", Name: obj.Name + "-role"},
		Subjects:  []orktypes.SubjectSpec{{Kind: "ServiceAccount", Name: obj.Name + "-sa", Namespace: ns}},
	}, obj.Name)
	if err := orkrb.Update(ctx, kube, obj, rbSpec); err != nil {
		return fmt.Errorf("rolebinding: %w", err)
	}

	// ── NetworkPolicy ─────────────────────────────────────────────────────────
	netpolSpec := orknetpol.Resolve(orktypes.NetworkPolicyTemplateSource{
		Name:      obj.Name + "-netpol",
		Namespace: ns,
	}, obj.Name, reg)
	if err := orknetpol.Update(ctx, kube, obj, netpolSpec); err != nil {
		return fmt.Errorf("networkpolicy: %w", err)
	}

	// ── ResourceQuota ─────────────────────────────────────────────────────────
	quotaSpec := orkquota.Resolve(orktypes.ResourceQuotaTemplateSource{
		Name:      obj.Name + "-quota",
		Namespace: ns,
		Hard:      map[string]string{"cpu": "4", "memory": "8Gi", "pods": "20"},
	}, obj.Name, reg)
	if err := orkquota.Update(ctx, kube, obj, quotaSpec); err != nil {
		return fmt.Errorf("resourcequota: %w", err)
	}

	// ── LimitRange ────────────────────────────────────────────────────────────
	lrSpec := orklimitrange.Resolve(orktypes.LimitRangeTemplateSource{
		Name:      obj.Name + "-limits",
		Namespace: ns,
		Limits: []orktypes.LimitRangeItem{
			{
				Type:           "Container",
				Default:        map[string]string{"cpu": "500m", "memory": "512Mi"},
				DefaultRequest: map[string]string{"cpu": "250m", "memory": "256Mi"},
			},
		},
	}, obj.Name, reg)
	if err := orklimitrange.Update(ctx, kube, obj, lrSpec); err != nil {
		return fmt.Errorf("limitrange: %w", err)
	}

	// ── Deployment ────────────────────────────────────────────────────────────
	deploySpec := orkdeployment.Resolve(orktypes.DeploymentTemplateSource{
		Name:               obj.Name + "-deploy",
		Namespace:          ns,
		Image:              obj.Spec.Image,
		Replicas:           obj.Spec.Replicas,
		Port:               obj.Spec.Port,
		ServiceAccountName: obj.Name + "-sa",
	}, obj.Name, reg)
	if err := orkdeployment.Update(ctx, kube, obj, deploySpec); err != nil {
		return fmt.Errorf("deployment: %w", err)
	}

	// ── StatefulSet ───────────────────────────────────────────────────────────
	stsSpec := orkstatefulset.Resolve(orktypes.StatefulSetTemplateSource{
		Name:      obj.Name + "-sts",
		Namespace: ns,
		Image:     obj.Spec.Image,
		Replicas:  obj.Spec.Replicas,
		Port:      obj.Spec.Port,
	}, obj.Name, reg)
	if err := orkstatefulset.Update(ctx, kube, obj, stsSpec); err != nil {
		return fmt.Errorf("statefulset: %w", err)
	}

	// ── ReplicaSet ────────────────────────────────────────────────────────────
	rsSpec := orkreplicaset.Resolve(orktypes.ReplicaSetTemplateSource{
		Name:      obj.Name + "-rs",
		Namespace: ns,
		Image:     obj.Spec.Image,
		Replicas:  obj.Spec.Replicas,
		Port:      obj.Spec.Port,
	}, obj.Name, reg)
	if err := orkreplicaset.Update(ctx, kube, obj, rsSpec); err != nil {
		return fmt.Errorf("replicaset: %w", err)
	}

	// ── Pod ───────────────────────────────────────────────────────────────────
	podSpec := orkpod.Resolve(orktypes.PodTemplateSource{
		Name:      obj.Name + "-pod",
		Namespace: ns,
		Image:     obj.Spec.Image,
		Port:      obj.Spec.Port,
	}, obj.Name, reg)
	if err := orkpod.Update(ctx, kube, obj, podSpec); err != nil {
		return fmt.Errorf("pod: %w", err)
	}

	// ── Job (Create — no Update) ──────────────────────────────────────────────
	jobSpec := orkjob.Resolve(orktypes.JobTemplateSource{
		Name:      obj.Name + "-init",
		Namespace: ns,
		Image:     "alpine:3.19",
		Command:   []string{"/bin/sh", "-c", "echo probe-job-ok"},
	}, 3, obj.Name, reg)
	if err := orkjob.Create(ctx, kube, obj, jobSpec); err != nil {
		return fmt.Errorf("job: %w", err)
	}

	// ── CronJob ───────────────────────────────────────────────────────────────
	cronSpec := orkcronjob.Resolve(orktypes.CronJobTemplateSource{
		Name:      obj.Name + "-cron",
		Namespace: ns,
		Schedule:  obj.Spec.Schedule,
		Image:     "alpine:3.19",
		Command:   []string{"/bin/sh", "-c", "echo probe-cron-ok"},
	}, obj.Name, reg)
	if err := orkcronjob.Update(ctx, kube, obj, cronSpec); err != nil {
		return fmt.Errorf("cronjob: %w", err)
	}

	// ── Service ───────────────────────────────────────────────────────────────
	svcSpec := orksvc.Resolve(orktypes.ServiceTemplateSource{
		Name:       obj.Name + "-svc",
		Namespace:  ns,
		Port:       "80",
		TargetPort: obj.Spec.Port,
	}, obj.Name)
	if err := orksvc.Update(ctx, kube, obj, svcSpec); err != nil {
		return fmt.Errorf("service: %w", err)
	}

	// ── Ingress ───────────────────────────────────────────────────────────────
	ingSpec := orkingress.Resolve(orktypes.IngressTemplateSource{
		Name:        obj.Name + "-ing",
		Namespace:   ns,
		Host:        obj.Name + ".probe.local",
		ServiceName: obj.Name + "-svc",
		ServicePort: "80",
	}, obj.Name)
	if err := orkingress.Update(ctx, kube, obj, ingSpec); err != nil {
		return fmt.Errorf("ingress: %w", err)
	}

	// ── HPA ───────────────────────────────────────────────────────────────────
	hpaSpec := orkhpa.Resolve(orktypes.HPATemplateSource{
		Name:      obj.Name + "-hpa",
		Namespace: ns,
		ScaleTargetRef: orktypes.ScaleTargetRef{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       obj.Name + "-deploy",
		},
		MinReplicas: "1",
		MaxReplicas: "3",
	}, obj.Name, reg)
	if err := orkhpa.Update(ctx, kube, obj, hpaSpec); err != nil {
		return fmt.Errorf("hpa: %w", err)
	}

	// ── PDB ───────────────────────────────────────────────────────────────────
	pdbSpec := orkpdb.Resolve(orktypes.PDBTemplateSource{
		Name:         obj.Name + "-pdb",
		Namespace:    ns,
		Selector:     map[string]string{"app": obj.Name + "-deploy"},
		MinAvailable: "1",
	}, obj.Name, reg)
	if err := orkpdb.Update(ctx, kube, obj, pdbSpec); err != nil {
		return fmt.Errorf("pdb: %w", err)
	}

	// ── PV ────────────────────────────────────────────────────────────────────
	pvSpec := orkpv.Resolve(orktypes.PVTemplateSource{
		Name:        obj.Name + "-pv",
		Capacity:    obj.Spec.Storage,
		AccessModes: []string{"ReadWriteOnce"},
		HostPath:    "/tmp/orkestra-" + obj.Name,
	}, obj.Name)
	if err := orkpv.Update(ctx, kube, obj, pvSpec); err != nil {
		return fmt.Errorf("pv: %w", err)
	}

	// ── PVC ───────────────────────────────────────────────────────────────────
	pvcSpec := orkpvc.Resolve(orktypes.PVCTemplateSource{
		Name:        obj.Name + "-pvc",
		Namespace:   ns,
		Storage:     obj.Spec.Storage,
		AccessModes: []string{"ReadWriteOnce"},
		VolumeName:  obj.Name + "-pv",
	}, obj.Name)
	if err := orkpvc.Update(ctx, kube, obj, pvcSpec); err != nil {
		return fmt.Errorf("pvc: %w", err)
	}

	return nil
}

// onDelete is a no-op: ResourceProbe is cluster-scoped, so owner references work
// for all resources (namespaced in probe-ns and cluster-scoped). GC handles cleanup.
func onDelete(_ context.Context, _ *rpv1alpha1.ResourceProbe) error {
	return nil
}
