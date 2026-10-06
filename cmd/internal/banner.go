package internal

import (
	"fmt"
	"strings"

	"github.com/inrundev/inrun/pkg/utils"
	"github.com/inrundev/inrun/pkg/version"
)

func printBanner(kfg *runtimeKfg, leader string) {
	fmt.Println("====================================================")
	fmt.Printf("%s Inrun Runtime%s (%s)\n",
		utils.Magenta(""), utils.Reset(""), version.Version)

	fmt.Printf("        Namespace: %s\n", utils.Cyan(kfg.config.Cluster().Namespace()))
	fmt.Printf("        Environment: %s\n", utils.Blue(kfg.config.Inrun().Environment()))
	fmt.Printf("        Listening on: %s:%s\n",
		utils.Green(kfg.config.Health().Port()), utils.Reset(""))

	if leader != "" {
		fmt.Printf("        Leader: %s\n", utils.Yellow(leader))
	} else {
		fmt.Printf("        Leader: %s\n", utils.Red("PENDING"))
	}

	fmt.Println("====================================================")

	// Endpoints
	fmt.Println("Inrun Endpoints:")
	fmt.Printf("- Startup:   %s\n", utils.Green("/startup"))
	fmt.Printf("- Health:    %s\n", utils.Green("/health"))
	fmt.Printf("- Ready:     %s\n", utils.Green("/ready"))
	fmt.Printf("- Metrics:   %s\n", utils.Green("/metrics"))
	fmt.Println()

	// Webhooks
	if kfg.catalog.HasMutationRules() ||
		kfg.catalog.HasValidationRules() ||
		kfg.catalog.IsDeletionProtectionEnabled() {

		fmt.Println("Webhook Endpoints:")

		if kfg.catalog.HasMutationRules() {
			fmt.Printf("- Mutation:  %s\n", utils.Green("/mutate"))
		}
		if kfg.catalog.HasValidationRules() {
			fmt.Printf("- Validation: %s\n", utils.Green("/validate"))
		}
		if kfg.catalog.IsDeletionProtectionEnabled() {
			fmt.Printf("- Deletion Protection: %s\n", utils.Green("/deletion-protection"))
			fmt.Printf("- Failure Policy: %s\n",
				utils.Cyan(kfg.catalog.DeletionProtectionFailurePolicy()))
		}
		if kfg.catalog.IsNamespaceProtectionEnabled() {
			fmt.Printf("- Namespace Protection: %s\n", utils.Green("/namespace-protection"))
			fmt.Printf("- Failure Policy: %s\n",
				utils.Cyan(kfg.catalog.NamespaceProtectionFailurePolicy()))
		}
		if kfg.catalog.HasConversionPaths() {
			fmt.Printf("- Conversion: %s\n", utils.Green("/convert"))
		}

		fmt.Println("Webhook Configuration:")
		fmt.Printf("- Service Name: %s\n", utils.Cyan(kfg.catalog.WebhooksServiceName()))
		fmt.Printf("- Service Namespace: %s\n", utils.Cyan(kfg.config.Cluster().Namespace()))
		fmt.Printf("- General Failure Policy: %s\n",
			utils.Cyan(kfg.catalog.WebhooksFailurePolicy()))
		fmt.Println()
	}

	fmt.Println("Catalog Endpoints:")
	fmt.Printf("- Catalog:  %s\n", utils.Green("/catalog"))

	for _, crd := range kfg.catalog.Enabled() {
		if !crd.IsEnabledAllEndpoints() {
			continue
		}
		kind := utils.Cyan(crd.APITypes.Kind)
		name := strings.ToLower(crd.Name)

		if crd.IsInfoEnabled() {
			fmt.Printf("  - %s (%s): %s\n", kind, crd.Name, utils.Green("/catalog/"+name))
		}
		if crd.IsHealthEnabled() {
			fmt.Printf("  - %s (%s): %s\n", kind, crd.Name, utils.Green("/catalog/"+name+"/health"))
		}
	}
	fmt.Println("====================================================")

	// Components
	fmt.Println("Components:")
	for _, c := range *kfg.komp {
		name := fmt.Sprintf("- %-20s", c.Name())
		switch {
		case c.Started():
			fmt.Printf("%s %s\n", name, utils.Green("AVAILABLE"))
		case c.Name() == "inrun dependency coordinator":
			fmt.Printf("%s %s\n", name, utils.Blue("STARTING"))
		default:
			fmt.Printf("%s %s\n", name, utils.Red("UNAVAILABLE"))
		}
	}
	fmt.Println("====================================================")

	// CRDs
	fmt.Println("CRDs:")
	for _, crd := range kfg.catalog.Enabled() {
		fmt.Printf("- %s\n", utils.Cyan(crd.APITypes.Kind))

		fmt.Printf("  Name:          %s\n", utils.Yellow(crd.Name))
		fmt.Printf("  Group:         %s\n", utils.Yellow(crd.APITypes.Group))
		fmt.Printf("  Version:       %s\n", utils.Yellow(crd.APITypes.Version))
		fmt.Printf("  Enabled:       %s\n",
			utils.Green(map[bool]string{true: "Yes", false: "No"}[crd.IsEnabled()]))

		if crd.Namespace != "" {
			fmt.Printf("  Namespace:     %s\n", utils.Yellow(crd.Namespace))
		}

		fmt.Printf("  Namespaced:    %s\n",
			utils.Green(map[bool]string{true: "Yes", false: "No"}[crd.IsNamespaced()]))

		if w := crd.SetWorkers(0); w > 0 {
			fmt.Printf("  Workers:       %d\n", w)
		} else {
			fmt.Printf("  Workers:       %d (default)\n", kfg.config.Catalog().DefaultWorkers())
		}

		if d := crd.SetQueueDepth(0); d > 0 {
			fmt.Printf("  MaxDepth: %d\n", d)
		} else {
			fmt.Printf("  MaxDepth: %d (default)\n",
				kfg.config.Catalog().DefaultQueueDepth())
		}

		if r := crd.SetResync(0); r != 0 {
			fmt.Printf("  Resync:        %s\n", r.String())
		} else {
			fmt.Printf("  Resync:        %s (default)\n",
				kfg.config.Catalog().DefaultResync().String())
		}

		if len(crd.DependsOn) > 0 {
			fmt.Printf("  DependsOn:     %s\n",
				utils.Yellow(strings.Join(crd.DependsOn.Names(), ", ")))
		} else {
			fmt.Printf("  DependsOn:     No dependencies\n")
		}

		fmt.Println()
	}
	fmt.Println("====================================================")

	fmt.Println(utils.Magenta("Inrun is reconciling your CRDs..."))
}
