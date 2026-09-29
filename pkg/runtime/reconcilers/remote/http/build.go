package http

import (
	"fmt"
	"strings"

	"github.com/orkspace/orkestra/domain"
	"github.com/orkspace/orkestra/pkg/resources/configmaps"
	"github.com/orkspace/orkestra/pkg/resources/cronjobs"
	"github.com/orkspace/orkestra/pkg/resources/customresources"
	"github.com/orkspace/orkestra/pkg/resources/deployments"
	"github.com/orkspace/orkestra/pkg/resources/ingresses"
	"github.com/orkspace/orkestra/pkg/resources/jobs"
	"github.com/orkspace/orkestra/pkg/resources/secrets"
	"github.com/orkspace/orkestra/pkg/resources/serviceaccounts"
	"github.com/orkspace/orkestra/pkg/resources/services"
	"github.com/orkspace/orkestra/pkg/resources/statefulsets"
)

// supportedIntentTypes lists all named resource types accepted in the intent form.
// Extend this slice whenever a new BuildFromIntent is added.
var supportedIntentTypes = []string{
	"configmap",
	"cronjob",
	"custom",
	"deployment",
	"ingress",
	"job",
	"secret",
	"serviceaccount",
	"service",
	"statefulset",
}

// buildFromIntent dispatches a resource intent to the appropriate builder.
func buildFromIntent(typ string, fields map[string]interface{}, owner domain.Object) (map[string]interface{}, error) {
	if owner == nil {
		return nil, fmt.Errorf("intent form requires a prepared request with an owner object")
	}
	switch typ {
	case "configmap":
		return configmaps.BuildFromIntent(fields, owner)
	case "cronjob":
		return cronjobs.BuildFromIntent(fields, owner)
	case "custom":
		return customresources.BuildFromIntent(fields, owner)
	case "deployment":
		return deployments.BuildFromIntent(fields, owner)
	case "ingress":
		return ingresses.BuildFromIntent(fields, owner)
	case "job":
		return jobs.BuildFromIntent(fields, owner)
	case "secret":
		return secrets.BuildFromIntent(fields, owner)
	case "serviceaccount":
		return serviceaccounts.BuildFromIntent(fields, owner)
	case "service":
		return services.BuildFromIntent(fields, owner)
	case "statefulset":
		return statefulsets.BuildFromIntent(fields, owner)
	default:
		return nil, fmt.Errorf(
			"unknown resource type %q — use full apiVersion/kind form or one of: %s",
			typ, strings.Join(supportedIntentTypes, ", "),
		)
	}
}
