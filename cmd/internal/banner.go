package internal

import (
	"fmt"
	"strings"

	"github.com/orkspace/orkestra/pkg/utils"
	"github.com/orkspace/orkestra/pkg/version"
)

func printBanner(kfg *runtimeKfg, konductor string) {
	// Logo
	fmt.Println(utils.Cyan(utils.Center(utils.OrkestraLogo)))

	fmt.Println("====================================================")
	fmt.Printf("%s Orkestra Runtime%s (%s)\n",
		utils.Magenta(""), utils.Reset(""), version.Version)

	fmt.Printf("        Namespace: %s\n", utils.Cyan(kfg.konfig.Cluster().Namespace()))
	fmt.Printf("        Environment: %s\n", utils.Blue(kfg.konfig.Ork().Environment()))
	fmt.Printf("        Listening on: %s:%s\n",
		utils.Green(kfg.konfig.Health().Port()), utils.Reset(""))

	if konductor != "" {
		fmt.Printf("        Konductor: %s\n", utils.Yellow(konductor))
	} else {
		fmt.Printf("        Konductor: %s\n", utils.Red("PENDING"))
	}

	fmt.Println("====================================================")

	// Endpoints
	fmt.Println("Orkestra Endpoints:")
	fmt.Printf("- Startup:   %s\n", utils.Green("/startup"))
	fmt.Printf("- Health:    %s\n", utils.Green("/health"))
	fmt.Printf("- Ready:     %s\n", utils.Green("/ready"))
	fmt.Printf("- Metrics:   %s\n", utils.Green("/metrics"))
	fmt.Println()

	// Webhooks
	if kfg.katalog.HasMutationRules() ||
		kfg.katalog.HasValidationRules() ||
		kfg.katalog.IsDeletionProtectionEnabled() {

		fmt.Println("Webhook Endpoints:")

		if kfg.katalog.HasMutationRules() {
			fmt.Printf("- Mutation:  %s\n", utils.Green("/mutate"))
		}
		if kfg.katalog.HasValidationRules() {
			fmt.Printf("- Validation: %s\n", utils.Green("/validate"))
		}
		if kfg.katalog.IsDeletionProtectionEnabled() {
			fmt.Printf("- Deletion Protection: %s\n", utils.Green("/deletion-protection"))
			fmt.Printf("- Failure Policy: %s\n",
				utils.Cyan(kfg.katalog.DeletionProtectionFailurePolicy()))
		}
		if kfg.katalog.IsNamespaceProtectionEnabled() {
			fmt.Printf("- Namespace Protection: %s\n", utils.Green("/namespace-protection"))
			fmt.Printf("- Failure Policy: %s\n",
				utils.Cyan(kfg.katalog.NamespaceProtectionFailurePolicy()))
		}
		if kfg.katalog.HasConversionPaths() {
			fmt.Printf("- Conversion: %s\n", utils.Green("/convert"))
		}

		fmt.Println("Webhook Configuration:")
		fmt.Printf("- Service Name: %s\n", utils.Cyan(kfg.katalog.WebhooksServiceName()))
		fmt.Printf("- Service Namespace: %s\n", utils.Cyan(kfg.konfig.Cluster().Namespace()))
		fmt.Printf("- General Failure Policy: %s\n",
			utils.Cyan(kfg.katalog.WebhooksFailurePolicy()))
		fmt.Println()
	}

	fmt.Println("Katalog Endpoints:")
	fmt.Printf("- Katalog:  %s\n", utils.Green("/katalog"))

	for _, crd := range kfg.katalog.Enabled() {
		if !crd.IsEnabledAllEndpoints() {
			continue
		}
		kind := utils.Cyan(crd.APITypes.Kind)
		name := strings.ToLower(crd.Name)

		if crd.IsInfoEnabled() {
			fmt.Printf("  - %s (%s): %s\n", kind, crd.Name, utils.Green("/katalog/"+name))
		}
		if crd.IsHealthEnabled() {
			fmt.Printf("  - %s (%s): %s\n", kind, crd.Name, utils.Green("/katalog/"+name+"/health"))
		}
	}
	fmt.Println("====================================================")

	// Komponents
	fmt.Println("Komponents:")
	for _, c := range *kfg.komp {
		name := fmt.Sprintf("- %-20s", c.Name())
		switch {
		case c.Started():
			fmt.Printf("%s %s\n", name, utils.Green("AVAILABLE"))
		case c.Name() == "orkestra dependency kordinator":
			fmt.Printf("%s %s\n", name, utils.Blue("STARTING"))
		default:
			fmt.Printf("%s %s\n", name, utils.Red("UNAVAILABLE"))
		}
	}
	fmt.Println("====================================================")

	// CRDs
	fmt.Println("CRDs:")
	for _, crd := range kfg.katalog.Enabled() {
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
			fmt.Printf("  Workers:       %d (default)\n", kfg.konfig.Katalog().DefaultWorkers())
		}

		if d := crd.SetQueueDepth(0); d > 0 {
			fmt.Printf("  MaxDepth: %d\n", d)
		} else {
			fmt.Printf("  MaxDepth: %d (default)\n",
				kfg.konfig.Katalog().DefaultQueueDepth())
		}

		if r := crd.SetResync(0); r != 0 {
			fmt.Printf("  Resync:        %s\n", r.String())
		} else {
			fmt.Printf("  Resync:        %s (default)\n",
				kfg.konfig.Katalog().DefaultResync().String())
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

	fmt.Println(utils.Magenta("Orkestra is konducting your CRDs..."))
}
