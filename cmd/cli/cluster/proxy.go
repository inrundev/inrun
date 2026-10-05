//go:build !runtime && !gateway

package cluster

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/orkspace/orkestra/cmd/cli/cmdutil"

	"github.com/orkspace/orkestra/pkg/tools/proxy"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Forward Orkestra service ports to localhost",
	Long: `Forward ports for deployed Orkestra components to localhost.

By default all deployed components are forwarded. Use --for to select specific components.

Examples:
  ork proxy                        # Forward Runtime, Control Center, and Gateway
  ork proxy --for cc               # Forward Control Center only
  ork proxy --for runtime,cc       # Forward Runtime and Control Center
  ork proxy -n my-platform-ns      # Forward from a custom namespace
  ork proxy --runtime-port 9090    # Use port 9090 for Runtime instead of 8080`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ns, _ := cmd.Flags().GetString("namespace")
		runtimePort, _ := cmd.Flags().GetInt("runtime-port")
		ccPort, _ := cmd.Flags().GetInt("cc-port")
		gatewayPort, _ := cmd.Flags().GetInt("gateway-port")
		devServerPort, _ := cmd.Flags().GetInt("dev-server-port")
		kubeContext, _ := cmd.Flags().GetString("context")

		include, err := proxyComponentsFromFor(cmd)
		if err != nil {
			return err
		}

		cfg, cs, err := buildProxyClient(kubeContext)
		if err != nil {
			return fmt.Errorf("connect to cluster: %w", err)
		}

		var targets []proxy.ForwardTarget
		if include.runtime {
			targets = append(targets, proxy.ForwardTarget{
				Label:     "Runtime",
				Komponent: proxy.KomponentRuntime,
				Namespace: ns,
				LocalPort: runtimePort,
				Scheme:    "http",
				ViaLease:  true,
			})
		}
		if include.cc {
			targets = append(targets, proxy.ForwardTarget{
				Label:     "Control Center",
				Komponent: proxy.KomponentCC,
				Namespace: ns,
				LocalPort: ccPort,
				Scheme:    "http",
			})
		}
		if include.gateway {
			targets = append(targets, proxy.ForwardTarget{
				Label:     "Gateway",
				Komponent: proxy.KomponentGateway,
				Namespace: ns,
				LocalPort: gatewayPort,
				Scheme:    "http",
			})
		}
		if include.devServer {
			targets = append(targets, proxy.ForwardTarget{
				Label:       "Dev Server",
				ServiceName: proxy.DevServerServiceName,
				Komponent:   proxy.DevServer,
				Namespace:   ns,
				LocalPort:   devServerPort,
				Scheme:      "http",
			})
		}

		// Port conflict pre-check
		portOK := true
		for _, t := range targets {
			if err := proxy.CheckPort(t.LocalPort); err != nil {
				fmt.Fprintf(os.Stderr, "  %s %-14s port %d in use — use --%s-port to set an alternative\n",
					cmdutil.Red("✗"), t.Label, t.LocalPort, strings.ToLower(t.Komponent))
				portOK = false
			}
		}
		if !portOK {
			return fmt.Errorf("one or more local ports are already in use")
		}

		fmt.Printf("\n  Connecting to %s...  %s\n\n", ns, cmdutil.Gray("Press Ctrl+C to stop."))

		ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		proxy.RunAll(ctx, cfg, cs, targets, os.Stdout)

		fmt.Println("\n  Disconnected.")
		return nil
	},
}

// proxyIncludes tracks which components the user selected via --for.
type proxyIncludes struct {
	runtime   bool
	cc        bool
	gateway   bool
	devServer bool
}

// proxyComponentsFromFor parses the --for flag using the same vocabulary as
// ork generate bundle --for: runtime (run), gateway (gw), cc (controlcenter, control-center).
// An absent flag means all three components.
func proxyComponentsFromFor(cmd *cobra.Command) (proxyIncludes, error) {
	forVal, _ := cmd.Flags().GetString("for")
	if forVal == "" {
		return proxyIncludes{runtime: true, cc: true, gateway: true, devServer: true}, nil
	}
	var inc proxyIncludes
	var unknown []string
	for _, part := range strings.Split(forVal, ",") {
		name := strings.TrimSpace(strings.ToLower(part))
		if name == "" {
			continue
		}
		switch name {
		case "run", "runtime":
			inc.runtime = true
		case "gw", "gateway":
			inc.gateway = true
		case "cc", "controlcenter", "control-center":
			inc.cc = true
		case "dev", "devserver", "dev-server":
			inc.devServer = true
		default:
			unknown = append(unknown, part)
		}
	}
	if len(unknown) > 0 {
		return proxyIncludes{}, fmt.Errorf(
			"orkestra: unknown --for value(s): %s\n\nValid values are:\n"+
				"  runtime   	(alias: run)          	— reconcilers, leader election\n"+
				"  gateway   	(alias: gw)            	— TLS, admission webhooks\n"+
				"  cc        	(alias: controlcenter) 	— control center\n\n"+
				"  devServer 	(alias: devserver)		— dev server\n\n"+
				"Example: --for gateway\n"+
				"         --for runtime,cc",
			strings.Join(unknown, ", "),
		)
	}
	if !inc.runtime && !inc.cc && !inc.gateway && !inc.devServer {
		return proxyIncludes{}, fmt.Errorf(
			"orkestra: --for produced an empty component list\n\n" +
				"Valid values are: runtime (run), gateway (gw), cc (controlcenter, control-center) devserver (dev, devserver, dev-server)",
		)
	}
	return inc, nil
}

// buildProxyClient constructs a REST config and typed clientset from the active
// kubeconfig. Respects the --kubeconfig flag (via kfg) and an explicit context override.
func buildProxyClient(kubeContext string) (*rest.Config, kubernetes.Interface, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if cmdutil.Kfg != nil && cmdutil.Kfg.Cluster().KubekonfigPath() != "" {
		loadingRules.ExplicitPath = cmdutil.Kfg.Cluster().KubekonfigPath()
	}
	overrides := &clientcmd.ConfigOverrides{}
	if kubeContext != "" {
		overrides.CurrentContext = kubeContext
	}
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules, overrides,
	).ClientConfig()
	if err != nil {
		return nil, nil, err
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}
	return cfg, cs, nil
}

func init() {
	cmdutil.RootCmd.AddCommand(proxyCmd)

	proxyCmd.Flags().StringP("namespace", "n", cmdutil.DefaultNamespace(), "Namespace where Orkestra is deployed")
	proxyCmd.Flags().String("for", "", "Comma-separated components to forward: runtime (run), gateway (gw), cc (controlcenter, control-center)")
	proxyCmd.Flags().Int("runtime-port", 8080, "Local port for Runtime")
	proxyCmd.Flags().Int("cc-port", 8081, "Local port for Control Center")
	proxyCmd.Flags().Int("gateway-port", 8443, "Local port for Gateway")
	proxyCmd.Flags().Int("dev-server-port", 9999, "Local port for Dev Server")
	proxyCmd.Flags().String("context", "", "Kubernetes context to use")

	// Shadow global flags so they don't appear under `ork proxy`
	cmdutil.ShadowGlobalCommandFlags(proxyCmd, "file")
}
