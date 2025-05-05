package konvey

import (
	"context"
	// "crypto/x509"
	"fmt"
	"io"
	// "net/http"
	"sort"
	// "strings"
	"time"

	// "github.com/kowabunga-cloud/kowabunga/kowabunga/common"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/klog"
	"github.com/kowabunga-cloud/kowabunga/kowabunga/common/metadata"
	// "github.com/go-acme/lego/v4/challenge"
	// gokitmetrics "github.com/go-kit/kit/metrics"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	ptypes "github.com/traefik/paerser/types"
	//"github.com/traefik/traefik/v3/pkg/config/dynamic"
	//"github.com/traefik/traefik/v3/pkg/config/runtime"
	"github.com/traefik/traefik/v3/pkg/config/static"
	"github.com/traefik/traefik/v3/pkg/metrics"
	"github.com/traefik/traefik/v3/pkg/middlewares/accesslog"
	"github.com/traefik/traefik/v3/pkg/ping"
	// "github.com/traefik/traefik/v3/pkg/plugins"
	// "github.com/traefik/traefik/v3/pkg/provider/acme"
	"github.com/traefik/traefik/v3/pkg/provider/aggregator"
	// "github.com/traefik/traefik/v3/pkg/provider/file"
	"github.com/traefik/traefik/v3/pkg/provider/traefik"
	"github.com/traefik/traefik/v3/pkg/proxy/httputil"
	"github.com/traefik/traefik/v3/pkg/safe"
	"github.com/traefik/traefik/v3/pkg/server"
	"github.com/traefik/traefik/v3/pkg/server/middleware"
	"github.com/traefik/traefik/v3/pkg/server/service"
	"github.com/traefik/traefik/v3/pkg/tcp"
	// traefiktls "github.com/traefik/traefik/v3/pkg/tls"
	"github.com/traefik/traefik/v3/pkg/tracing"
	"github.com/traefik/traefik/v3/pkg/types"
	// "github.com/traefik/traefik/v3/pkg/version"
)

type TraefikNetworkLoadBalancer struct {
	sc     static.Configuration
	server *server.Server
}

func NewTraefikNetworkLoadBalancer() (*TraefikNetworkLoadBalancer, error) {
	klog.Infof("Initializing Traefik load-balancer ...")

	// Entrypoints
	entryPoints := make(static.EntryPoints)

	addEntryPoint(entryPoints, "ping", "0.0.0.0", 8081, "")
	addEntryPoint(entryPoints, "metrics", "0.0.0.0", 8082, "")

	lb := &TraefikNetworkLoadBalancer{
		sc: static.Configuration{
			Global: &static.Global{
				CheckNewVersion:    false,
				SendAnonymousUsage: false,
			},
			ServersTransport: &static.ServersTransport{
				MaxIdleConnsPerHost: 200,
			},
			TCPServersTransport: &static.TCPServersTransport{
				DialTimeout:   ptypes.Duration(30 * time.Second),
				DialKeepAlive: ptypes.Duration(15 * time.Second),
			},
			EntryPoints: entryPoints,
			Providers: &static.Providers{
				ProvidersThrottleDuration: ptypes.Duration(2 * time.Second),
				// 	File: &file.Provider{
				// 		Directory: "/etc/traefik/conf.d",
				// 		Watch:     true,
				// 	},
			},
			API: &static.API{
				Insecure:           true,
				Dashboard:          true,
				Debug:              true,
				DisableDashboardAd: false,
			},
			Metrics: &types.Metrics{
				Prometheus: &types.Prometheus{
					EntryPoint:           "metrics",
					AddEntryPointsLabels: true,
					AddRoutersLabels:     true,
					AddServicesLabels:    true,
				},
			},
			Ping: &ping.Handler{
				EntryPoint: "ping",
			},
			Log: &types.TraefikLog{
				Level:    "DEBUG",
				Format:   "json",
				FilePath: "/tmp/traefik.log",
			},
			AccessLog: &types.AccessLog{
				Format:   "json",
				FilePath: "/tmp/traefik.access.log",
			},
		},
	}

	lb.sc.SetEffectiveConfiguration()
	err := lb.sc.ValidateConfiguration()
	if err != nil {
		return lb, err
	}

	err = lb.setup()
	if err != nil {
		return lb, err
	}

	return lb, nil
}

func (lb *TraefikNetworkLoadBalancer) setup() error {
	providerAggregator := aggregator.NewProviderAggregator(*lb.sc.Providers)

	ctx := context.Background()
	routinesPool := safe.NewPool(ctx)

	// adds internal provider
	err := providerAggregator.AddProvider(traefik.New(lb.sc))
	if err != nil {
		return err
	}

	// Observability
	metricRegistries := []metrics.Registry{}
	var semConvMetricRegistry *metrics.SemConvMetricsRegistry
	prometheusRegister := metrics.RegisterPrometheus(ctx, lb.sc.Metrics.Prometheus)
	if prometheusRegister != nil {
		metricRegistries = append(metricRegistries, prometheusRegister)
	}
	metricsRegistry := metrics.NewMultiRegistry(metricRegistries)
	accessLog := setupAccessLog(lb.sc.AccessLog)
	tracer, tracerCloser := setupTracing(lb.sc.Tracing)
	observabilityMgr := middleware.NewObservabilityMgr(lb.sc, metricsRegistry, semConvMetricRegistry, accessLog, tracer, tracerCloser)

	klog.Infof("ObservabilityMgr: %+v", observabilityMgr)

	// Entrypoints
	serverEntryPointsTCP, err := server.NewTCPEntryPoints(lb.sc.EntryPoints, lb.sc.HostResolver, metricsRegistry)
	if err != nil {
		return err
	}

	serverEntryPointsUDP, err := server.NewUDPEntryPoints(lb.sc.EntryPoints)
	if err != nil {
		return err
	}

	klog.Infof("TCP: %+v", serverEntryPointsTCP)
	klog.Infof("UDP: %+v", serverEntryPointsUDP)

	// Service manager factory
	var spiffeX509Source *workloadapi.X509Source
	transportManager := service.NewTransportManager(spiffeX509Source)
	var proxyBuilder service.ProxyBuilder = httputil.NewProxyBuilder(transportManager, semConvMetricRegistry)
	dialerManager := tcp.NewDialerManager(spiffeX509Source)

	managerFactory := service.NewManagerFactory(lb.sc, routinesPool, observabilityMgr, transportManager, proxyBuilder, nil)

	// Router factory

	routerFactory := server.NewRouterFactory(lb.sc, managerFactory, nil, observabilityMgr, nil, dialerManager)
	klog.Infof("Router Factory: %+v", routerFactory)

	// Watcher

	watcher := server.NewConfigurationWatcher(
		routinesPool,
		providerAggregator,
		getDefaultsEntrypoints(&lb.sc),
		"internal",
	)

	lb.server = server.NewServer(routinesPool, serverEntryPointsTCP, serverEntryPointsUDP,
		watcher, observabilityMgr)
	return nil
}

func (lb *TraefikNetworkLoadBalancer) Start() error {
	klog.Infof("Starting up Traefik load-balancer ...")
	if lb.server != nil {
		lb.server.Start(context.Background())
		lb.server.Wait()
	}

	return nil
}

func (lb *TraefikNetworkLoadBalancer) Reload(meta *metadata.InstanceMetadata) error {
	klog.Infof("Reloading configuration ...")
	return nil
}

func (lb *TraefikNetworkLoadBalancer) Stop() error {
	klog.Infof("Stopping down Traefik load-balancer ...")
	if lb.server != nil {
		lb.server.Stop()
	}
	return nil
}

func (lb *TraefikNetworkLoadBalancer) Shutdown() error {
	klog.Infof("Shutting down Traefik load-balancer ...")
	return nil
}

// const outputDir = "./plugins-storage/"

// func createPluginBuilder(staticConfiguration *static.Configuration) (*plugins.Builder, error) {
// 	client, plgs, localPlgs, err := initPlugins(staticConfiguration)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return plugins.NewBuilder(client, plgs, localPlgs)
// }

// func initPlugins(staticCfg *static.Configuration) (*plugins.Client, map[string]plugins.Descriptor, map[string]plugins.LocalDescriptor, error) {
// 	err := checkUniquePluginNames(staticCfg.Experimental)
// 	if err != nil {
// 		return nil, nil, nil, err
// 	}

// 	var client *plugins.Client
// 	plgs := map[string]plugins.Descriptor{}

// 	if hasPlugins(staticCfg) {
// 		opts := plugins.ClientOptions{
// 			Output: outputDir,
// 		}

// 		var err error
// 		client, err = plugins.NewClient(opts)
// 		if err != nil {
// 			return nil, nil, nil, fmt.Errorf("unable to create plugins client: %w", err)
// 		}

// 		err = plugins.SetupRemotePlugins(client, staticCfg.Experimental.Plugins)
// 		if err != nil {
// 			return nil, nil, nil, fmt.Errorf("unable to set up plugins environment: %w", err)
// 		}

// 		plgs = staticCfg.Experimental.Plugins
// 	}

// 	localPlgs := map[string]plugins.LocalDescriptor{}

// 	if hasLocalPlugins(staticCfg) {
// 		err := plugins.SetupLocalPlugins(staticCfg.Experimental.LocalPlugins)
// 		if err != nil {
// 			return nil, nil, nil, err
// 		}

// 		localPlgs = staticCfg.Experimental.LocalPlugins
// 	}

// 	return client, plgs, localPlgs, nil
// }

// func checkUniquePluginNames(e *static.Experimental) error {
// 	if e == nil {
// 		return nil
// 	}

// 	for s := range e.LocalPlugins {
// 		if _, ok := e.Plugins[s]; ok {
// 			return fmt.Errorf("the plugin's name %q must be unique", s)
// 		}
// 	}

// 	return nil
// }

// func hasPlugins(staticCfg *static.Configuration) bool {
// 	return staticCfg.Experimental != nil && len(staticCfg.Experimental.Plugins) > 0
// }

// func hasLocalPlugins(staticCfg *static.Configuration) bool {
// 	return staticCfg.Experimental != nil && len(staticCfg.Experimental.LocalPlugins) > 0
// }

func addEntryPoint(eps static.EntryPoints, name, address string, port int64, protocol string) {
	ep := &static.EntryPoint{
		Address: fmt.Sprintf("%s:%d", address, port),
	}
	if protocol != "" {
		ep.Address += fmt.Sprintf("/%s", protocol)
	}
	ep.SetDefaults()
	eps[name] = ep
}

// // initACMEProvider creates and registers acme.Provider instances corresponding to the configured ACME certificate resolvers.
// func initACMEProvider(c *static.Configuration, providerAggregator *aggregator.ProviderAggregator, tlsManager *traefiktls.Manager, httpChallengeProvider, tlsChallengeProvider challenge.Provider) []*acme.Provider {
// 	localStores := map[string]*acme.LocalStore{}

// 	var resolvers []*acme.Provider
// 	for name, resolver := range c.CertificatesResolvers {
// 		if resolver.ACME == nil {
// 			continue
// 		}

// 		if localStores[resolver.ACME.Storage] == nil {
// 			localStores[resolver.ACME.Storage] = acme.NewLocalStore(resolver.ACME.Storage)
// 		}

// 		p := &acme.Provider{
// 			Configuration:         resolver.ACME,
// 			Store:                 localStores[resolver.ACME.Storage],
// 			ResolverName:          name,
// 			HTTPChallengeProvider: httpChallengeProvider,
// 			TLSChallengeProvider:  tlsChallengeProvider,
// 		}

// 		if err := providerAggregator.AddProvider(p); err != nil {
// 			klog.Errorf("The ACME resolve is skipped from the resolvers list: %v", err)
// 			continue
// 		}

// 		p.SetTLSManager(tlsManager)

// 		p.SetConfigListenerChan(make(chan dynamic.Configuration))

// 		resolvers = append(resolvers, p)
// 	}

// 	return resolvers
// }

// func appendCertMetric(gauge gokitmetrics.Gauge, certificate *x509.Certificate) {
// 	sort.Strings(certificate.DNSNames)

// 	labels := []string{
// 		"cn", certificate.Subject.CommonName,
// 		"serial", certificate.SerialNumber.String(),
// 		"sans", strings.Join(certificate.DNSNames, ","),
// 	}

// 	notAfter := float64(certificate.NotAfter.Unix())

// 	gauge.With(labels...).Set(notAfter)
// }

func setupAccessLog(conf *types.AccessLog) *accesslog.Handler {
	if conf == nil {
		return nil
	}

	accessLoggerMiddleware, err := accesslog.NewHandler(conf)
	if err != nil {
		klog.Warningf("Unable to create access logger: %v", err)
		return nil
	}

	return accessLoggerMiddleware
}

func setupTracing(conf *static.Tracing) (*tracing.Tracer, io.Closer) {
	if conf == nil {
		return nil, nil
	}

	tracer, closer, err := tracing.NewTracing(conf)
	if err != nil {
		klog.Warningf("Unable to create tracer: %v", err)
		return nil, nil
	}

	return tracer, closer
}

// func getHTTPChallengeHandler(acmeProviders []*acme.Provider, httpChallengeProvider http.Handler) http.Handler {
// 	var acmeHTTPHandler http.Handler
// 	for _, p := range acmeProviders {
// 		if p != nil && p.HTTPChallenge != nil {
// 			acmeHTTPHandler = httpChallengeProvider
// 			break
// 		}
// 	}
// 	return acmeHTTPHandler
// }

func getDefaultsEntrypoints(sc *static.Configuration) []string {
	var defaultEntryPoints []string

	// Determines if at least one EntryPoint is configured to be used by default.
	var hasDefinedDefaults bool
	for _, ep := range sc.EntryPoints {
		if ep.AsDefault {
			hasDefinedDefaults = true
			break
		}
	}

	for name, cfg := range sc.EntryPoints {
		// By default all entrypoints are considered.
		// If at least one is flagged, then only flagged entrypoints are included.
		if hasDefinedDefaults && !cfg.AsDefault {
			continue
		}

		protocol, err := cfg.GetProtocol()
		if err != nil {
			// Should never happen because Traefik should not start if protocol is invalid.
			klog.Errorf("Invalid protocol: %v", err)
		}

		if protocol != "udp" && name != static.DefaultInternalEntryPointName {
			defaultEntryPoints = append(defaultEntryPoints, name)
		}
	}

	sort.Strings(defaultEntryPoints)
	return defaultEntryPoints
}

// func switchRouter(routerFactory *server.RouterFactory, serverEntryPointsTCP server.TCPEntryPoints, serverEntryPointsUDP server.UDPEntryPoints) func(conf dynamic.Configuration) {
// 	return func(conf dynamic.Configuration) {
// 		rtConf := runtime.NewConfig(conf)

// 		routers, udpRouters := routerFactory.CreateRouters(rtConf)

// 		serverEntryPointsTCP.Switch(routers)
// 		serverEntryPointsUDP.Switch(udpRouters)
// 	}
// }

// func setupServer(sc *static.Configuration) (*server.Server, error) {

// 	providerAggregator := aggregator.NewProviderAggregator(*sc.Providers)

// 	ctx := context.Background()
// 	routinesPool := safe.NewPool(ctx)

// 	// adds internal provider
// 	err := providerAggregator.AddProvider(traefik.New(*sc))
// 	if err != nil {
// 		return nil, err
// 	}

// 	// ACME
// 	tlsManager := traefiktls.NewManager()
// 	httpChallengeProvider := acme.NewChallengeHTTP()

// 	tlsChallengeProvider := acme.NewChallengeTLSALPN()
// 	err = providerAggregator.AddProvider(tlsChallengeProvider)
// 	if err != nil {
// 		return nil, err
// 	}

// 	acmeProviders := initACMEProvider(sc, &providerAggregator, tlsManager, httpChallengeProvider, tlsChallengeProvider)

// 	// Observability
// 	metricRegistries := []metrics.Registry{}
// 	var semConvMetricRegistry *metrics.SemConvMetricsRegistry
// 	prometheusRegister := metrics.RegisterPrometheus(ctx, sc.Metrics.Prometheus)
// 	if prometheusRegister != nil {
// 		metricRegistries = append(metricRegistries, prometheusRegister)
// 	}
// 	metricsRegistry := metrics.NewMultiRegistry(metricRegistries)
// 	accessLog := setupAccessLog(sc.AccessLog)
// 	tracer, tracerCloser := setupTracing(sc.Tracing)
// 	observabilityMgr := middleware.NewObservabilityMgr(*sc, metricsRegistry, semConvMetricRegistry, accessLog, tracer, tracerCloser)

// 	klog.Infof("ObservabilityMgr: %+v", observabilityMgr)

// 	// Entrypoints
// 	serverEntryPointsTCP, err := server.NewTCPEntryPoints(sc.EntryPoints, sc.HostResolver, metricsRegistry)
// 	if err != nil {
// 		return nil, err
// 	}

// 	serverEntryPointsUDP, err := server.NewUDPEntryPoints(sc.EntryPoints)
// 	if err != nil {
// 		return nil, err
// 	}

// 	klog.Infof("TCP: %+v", serverEntryPointsTCP)
// 	klog.Infof("UDP: %+v", serverEntryPointsUDP)

// 	if sc.API != nil {
// 		version.DisableDashboardAd = sc.API.DisableDashboardAd
// 	}

// 	// Plugins
// 	pluginBuilder, err := createPluginBuilder(sc)
// 	if err != nil {
// 		return nil, fmt.Errorf("plugin: failed to create plugin builder: %w", err)
// 	}

// 	// Providers plugins
// 	for name, conf := range sc.Providers.Plugin {
// 		if pluginBuilder == nil {
// 			break
// 		}

// 		p, err := pluginBuilder.BuildProvider(name, conf)
// 		if err != nil {
// 			return nil, fmt.Errorf("plugin: failed to build provider: %w", err)
// 		}

// 		err = providerAggregator.AddProvider(p)
// 		if err != nil {
// 			return nil, fmt.Errorf("plugin: failed to add provider: %w", err)
// 		}
// 	}

// 	// Service manager factory
// 	var spiffeX509Source *workloadapi.X509Source
// 	transportManager := service.NewTransportManager(spiffeX509Source)
// 	var proxyBuilder service.ProxyBuilder = httputil.NewProxyBuilder(transportManager, semConvMetricRegistry)
// 	dialerManager := tcp.NewDialerManager(spiffeX509Source)
// 	acmeHTTPHandler := getHTTPChallengeHandler(acmeProviders, httpChallengeProvider)
// 	managerFactory := service.NewManagerFactory(*sc, routinesPool, observabilityMgr, transportManager, proxyBuilder, acmeHTTPHandler)

// 	// Router factory

// 	routerFactory := server.NewRouterFactory(*sc, managerFactory, tlsManager, observabilityMgr, nil, dialerManager)
// 	klog.Infof("Router Factory: %+v", routerFactory)

// 	// Watcher

// 	watcher := server.NewConfigurationWatcher(
// 		routinesPool,
// 		providerAggregator,
// 		getDefaultsEntrypoints(sc),
// 		"internal",
// 	)

// 	// TLS
// 	watcher.AddListener(func(conf dynamic.Configuration) {
// 		ctx := context.Background()
// 		tlsManager.UpdateConfigs(ctx, conf.TLS.Stores, conf.TLS.Options, conf.TLS.Certificates)

// 		gauge := metricsRegistry.TLSCertsNotAfterTimestampGauge()
// 		for _, certificate := range tlsManager.GetServerCertificates() {
// 			appendCertMetric(gauge, certificate)
// 		}
// 	})

// 	// Metrics
// 	watcher.AddListener(func(_ dynamic.Configuration) {
// 		metricsRegistry.ConfigReloadsCounter().Add(1)
// 		metricsRegistry.LastConfigReloadSuccessGauge().Set(float64(time.Now().Unix()))
// 	})

// 	// Server Transports
// 	watcher.AddListener(func(conf dynamic.Configuration) {
// 		transportManager.Update(conf.HTTP.ServersTransports)
// 		proxyBuilder.Update(conf.HTTP.ServersTransports)
// 		dialerManager.Update(conf.TCP.ServersTransports)
// 	})

// 	// Switch router
// 	watcher.AddListener(switchRouter(routerFactory, serverEntryPointsTCP, serverEntryPointsUDP))

// 	// Metrics
// 	if metricsRegistry.IsEpEnabled() || metricsRegistry.IsRouterEnabled() || metricsRegistry.IsSvcEnabled() {
// 		var eps []string
// 		for key := range serverEntryPointsTCP {
// 			eps = append(eps, key)
// 		}
// 		watcher.AddListener(func(conf dynamic.Configuration) {
// 			metrics.OnConfigurationUpdate(conf, eps)
// 		})
// 	}

// 	// TLS challenge
// 	watcher.AddListener(tlsChallengeProvider.ListenConfiguration)

// 	// Certificate Resolvers

// 	resolverNames := map[string]struct{}{}

// 	// ACME
// 	for _, p := range acmeProviders {
// 		resolverNames[p.ResolverName] = struct{}{}
// 		watcher.AddListener(p.ListenConfiguration)
// 	}

// 	// Certificate resolver logs
// 	watcher.AddListener(func(config dynamic.Configuration) {
// 		for _, rt := range config.HTTP.Routers {
// 			if rt.TLS == nil || rt.TLS.CertResolver == "" {
// 				continue
// 			}

// 			if _, ok := resolverNames[rt.TLS.CertResolver]; !ok {
// 				klog.Errorf("Router uses a nonexistent certificate resolver")
// 			}
// 		}
// 	})

// 	return server.NewServer(routinesPool, serverEntryPointsTCP, serverEntryPointsUDP, watcher, observabilityMgr), nil
// }

// func NewTraefikLayer4LoadBalancer() error {

// 	// Entrypoints
// 	entryPoints := make(static.EntryPoints)

// 	addEntryPoint(entryPoints, "ping", settings.LocalIP, 8081, "")
// 	addEntryPoint(entryPoints, "metrics", settings.LocalIP, 8082, "")

// 	for _, e := range meta.Konvey.Endpoints {
// 		addEntryPoint(entryPoints, e.Name, "", e.Port, e.Protocol)
// 	}

// 	sc := static.Configuration{
// 		Global: &static.Global{
// 			CheckNewVersion:    false,
// 			SendAnonymousUsage: false,
// 		},
// 		ServersTransport: &static.ServersTransport{
// 			MaxIdleConnsPerHost: 200,
// 		},
// 		TCPServersTransport: &static.TCPServersTransport{
// 			DialTimeout:   ptypes.Duration(30 * time.Second),
// 			DialKeepAlive: ptypes.Duration(15 * time.Second),
// 		},
// 		EntryPoints: entryPoints,
// 		Providers: &static.Providers{
// 			ProvidersThrottleDuration: ptypes.Duration(2 * time.Second),
// 			File: &file.Provider{
// 				Directory: "/etc/traefik/conf.d",
// 				Watch:     true,
// 			},
// 		},
// 		API: &static.API{
// 			Insecure:           true,
// 			Dashboard:          true,
// 			Debug:              true,
// 			DisableDashboardAd: false,
// 		},
// 		Metrics: &types.Metrics{
// 			Prometheus: &types.Prometheus{
// 				EntryPoint:           "metrics",
// 				AddEntryPointsLabels: true,
// 				AddRoutersLabels:     true,
// 				AddServicesLabels:    true,
// 			},
// 		},
// 		Ping: &ping.Handler{
// 			EntryPoint: "ping",
// 		},
// 		Log: &types.TraefikLog{
// 			Level:    "ERROR",
// 			Format:   "json",
// 			FilePath: "/var/log/traefik/traefik.log",
// 		},
// 	}

// 	sc.SetEffectiveConfiguration()
// 	if err := sc.ValidateConfiguration(); err != nil {
// 		return err
// 	}

// 	for k, v := range sc.EntryPoints {
// 		klog.Infof("EntryPoint: %v : %#v\n", k, v)
// 	}

// 	srv, err := setupServer(&sc)
// 	if err != nil {
// 		return err
// 	}

// 	klog.Infof("Srv: %+v", srv)

// 	klog.Infof("Starting up Traefik")
// 	srv.Start(context.Background())
// 	defer srv.Close()

// 	srv.Wait()
// 	klog.Infof("Shutting down Traefik")

// 	return nil
// }
