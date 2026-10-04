package konfig

import (
	"time"

	"github.com/orkspace/orkestra/pkg/labels"
)

type Konfig struct {
	ork          orkKonfig
	cluster      clusterKonfig
	konductor    konductorElection
	healthServer healthServer
	katalog      katalogKonfig
	security     SecurityConfig
	registry     registryConfig
}

type orkKonfig struct {
	name        string   `validate:"required"`
	instance    Instance // runtime or gateway
	shortName   string
	environment string
	logLevel    string
}

type healthServer struct {
	port         string
	readTimeout  time.Duration
	writeTimeout time.Duration
}

type clusterKonfig struct {
	kubekonfigPath string
	masterURL      string
	name           string
	namespace      string `validate:"required"`
}

type registryConfig struct {
	RegistryURL string
}

// SecurityConfig is the unified security configuration populated from ENV vars
// at Init() time. Katalog YAML values are merged on top via the Katalog
// loader, so this represents the ENV-level defaults.
//
// Precedence: Katalog YAML > SecurityConfig (ENV) > hard default.
type SecurityConfig struct {
	ServiceName struct {
		Runtime string
		Gateway string
	}

	DeletionProtection struct {
		Enabled           bool
		CleanupOnShutdown bool
		ServiceName       string
		FailurePolicy     string
	}
	Webhooks struct {
		Admission struct {
			Enabled bool
		}
		CleanupOnShutdown bool
		FailurePolicy     string
		ServiceName       string
		// TLS paths — shared with deletion protection, admission, and conversion.
		// Set by ensureSecurity() after cert generation/loading.
		TLSCert string
		TLSKey  string

		Housekeeper struct {
			Enabled      bool
			SyncInterval time.Duration
		}
	}
	// Conversion is separate from admission webhooks — conversion has its own
	// /convert endpoint, window stats, and CRD patch logic.
	Conversion struct {
		Enabled bool
		// ConversionWindow is the rolling window size for latency/throughput stats.
		ConversionWindow  int
		CleanupOnShutdown bool // TODO
	}

	// NamespaceProtection controls the optional validating webhook that prevents
	// Orkestra-managed CRs from being created or updated in forbidden namespaces.
	//
	// This is an admission-time safeguard only. If disabled, namespace rules are
	// not enforced at apply time. If enabled, the webhook blocks CRs whose target
	// namespace violates the CRD’s declared allowedNamespaces or restrictedNamespaces.
	//
	// The webhook is managed by the housekeeper and will be recreated if
	// deleted, ensuring continuous enforcement when enabled.
	//
	// Precedence: Katalog YAML > SecurityConfig (ENV) > hard default.
	NamespaceProtection struct {
		Enabled           bool
		FailurePolicy     string
		ServiceName       string
		CleanupOnShutdown bool
	}

	// CertManager controls the lifecycle of Orkestra’s auto-generated TLS certificate.
	// Only applies when certificates are auto-generated (TLS_CERT/TLS_KEY not set).
	//
	// Precedence: Katalog YAML > SecurityConfig (ENV) > hard default.
	CertManager struct {
		// AutoRotate enables pre-emptive certificate rotation before expiry.
		// Default: true. Set TLS_AUTO_ROTATE=false to opt out.
		AutoRotate bool
		// RotationThreshold is how far before expiry Orkestra rotates.
		// Parsed from TLS_ROTATION_THRESHOLD env (e.g. "30d"). Default: "30d".
		RotationThreshold string
		// ValidFor is the default certificate validity duration.
		// Parsed from TLS_ROTATE_AFTER env (e.g. "30d"). Default: "1y".
		ValidFor string
	}
}

type katalogKonfig struct {
	paths                   []string // Comma separated Paths to CRD katalog YAML file
	defaultQueueDepth       int
	defaultFailureThreshold int `validate:"required"`
	defaultResync           time.Duration
	defaultWorkers          int
	shutdownTimeout         time.Duration
	shutdownGracePeriod     time.Duration
	// gatewayEndpoint is advertised in the runtime /katalog response so the
	// control center can locate the companion gateway and merge stats.
	// Populated from ORK_GATEWAY_ENDPOINT; empty when no gateway is configured.
	gatewayEndpoint string
}

type konductorElection struct {
	namespace     string
	leaseDuration time.Duration
	renewDeadline time.Duration
	retryPeriod   time.Duration
}

// Methods

// NewDefaultKonfig returns a Konfig populated from environment variables with
// sensible defaults. Used by CLI commands that do not go through the full
// Init() path (e.g. ork validate, ork template, ork simulate). All fields
// follow the same GetStrEnv/GetIntEnv/GetDurEnvSeconds pattern as Init().
func NewDefaultKonfig() *Konfig {
	ns := resolveNamespace()
	return &Konfig{
		ork: orkKonfig{
			name:        Orkestra,
			shortName:   Ork,
			environment: GetStrEnv("ORK_ENV", "development"),
			logLevel:    GetStrEnv("LOG_LEVEL", "info"),
		},
		cluster: clusterKonfig{
			kubekonfigPath: GetStrEnv("KUBEKONFIG", ""),
			masterURL:      GetStrEnv("MASTER_URL", ""),
			name:           GetStrEnv("CLUSTER_NAME", ""),
			namespace:      ns,
		},
		konductor: konductorElection{
			namespace:     ns,
			leaseDuration: GetDurEnvSeconds("LEASE_DURATION", 60),
			renewDeadline: GetDurEnvSeconds("RENEW_DEADLINE", 40),
			retryPeriod:   GetDurEnvSeconds("RETRY_PERIOD", 10),
		},
		healthServer: healthServer{
			port:         GetStrEnv("ORK_PORT", "8080"),
			readTimeout:  GetDurEnvSeconds("SRV_READ_TIMEOUT", 5),
			writeTimeout: GetDurEnvSeconds("SRV_WRITE_TIMEOUT", 20),
		},
		katalog: katalogKonfig{
			paths:                   GetStrSliceEnv("KATALOG_PATH", []string{"katalog.yaml"}),
			defaultWorkers:          GetIntEnv("DEFAULT_WORKERS", 3),
			defaultResync:           GetDurEnvSeconds("DEFAULT_RESYNC", 15),
			defaultQueueDepth:       GetIntEnv("QUEUE_DEPTH", 100),
			defaultFailureThreshold: GetIntEnv("FAILURE_THRESHOLD", 5),
		},
	}
}

// IsDev returns true for development environment
func (k *Konfig) IsDev() bool {
	return k.Ork().environment == "devlopment"
}

// IsDev returns true for staging environment
func (k *Konfig) IsStaging() bool {
	return k.Ork().environment == "staging"
}

// IsDev returns true for production environment
func (c *Konfig) IsProduction() bool {
	return c.Ork().environment == "production"
}

// Health returns health konfigurations
func (k *Konfig) Health() *healthServer {
	return &k.healthServer
}

// Ork returns Ork Konfigurations
func (k *Konfig) Ork() *orkKonfig {
	return &k.ork
}

// Runtime service name
func (k *Konfig) RuntimeServiceName() string {
	return k.security.ServiceName.Runtime
}

// Gateway service name
func (k *Konfig) GatewayServiceName() string {
	return k.security.ServiceName.Gateway
}

// Running instance returns the current running orkestra instance
func (k *Konfig) RunningInstance() string {
	return k.ork.instance.String()
}

// Cluster returns cluster Konfigurations
func (k *Konfig) Cluster() *clusterKonfig {
	return &k.cluster
}

// Konductor returns konductor Konfigurations
func (c *Konfig) Konductor() *konductorElection {
	return &c.konductor
}

// Katalog returns katalog Konfigurations
func (k *Konfig) Katalog() *katalogKonfig {
	return &k.katalog
}

// Finalizers return a list of default finalizers
func (k *Konfig) Finalizers() []string {
	return []string{labels.FinalizerOrkestra}
}

// Security returns the unified security configuration.
// This is the primary accessor for all security-related settings.
func (k *Konfig) Security() *SecurityConfig {
	return &k.security
}

// RegistryConfig returns registry configuration.
func (k *Konfig) RegistryConfig() *registryConfig {
	return &k.registry
}

// ConversionEnabled reports whether the conversion webhook is enabled.
// Reads from SecurityConfig (populated from ENV at Init).
func (k *Konfig) ConversionEnabled() bool {
	return k.security.Conversion.Enabled
}

// AdmissionEnabled reports whether admission webhooks are enabled.
// Reads from SecurityConfig (populated from ENV at Init).
func (k *Konfig) AdmissionEnabled() bool {
	return k.security.Webhooks.Admission.Enabled
}

// GatewayEndpoint returns the companion gateway URL advertised to the control center.
// Empty string when no gateway is configured (e.g. runtime-only deployment).
func (k *Konfig) GatewayEndpoint() string {
	return k.katalog.gatewayEndpoint
}

// HTTPSPort returns the HTTPS port string (e.g. ":8443") used by the webhook server.
func (k *Konfig) HTTPSPort() string {
	return httpsPort
}

// HTTPSPortInt32 returns the HTTPS port as int32 (8443) used in webhook client configs.
func (k *Konfig) HTTPSPortInt32() int32 {
	return httpsPortInt32
}

// DefaultInternalTLSName returns the default name for Orkestra's internal TLS secret.
func DefaultInternalTLSName() string {
	return defaultInternalTLSSecretName
}

// DefaultWorkloadSecretName returns the base name used for generated workload secrets.
// The caller appends the CR's name to form the final secret name.
func DefaultWorkloadSecretName() string {
	return defaultWorkloadSecretName
}
