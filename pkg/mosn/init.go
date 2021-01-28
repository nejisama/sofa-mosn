package mosn

import (
	goplugin "plugin"

	"mosn.io/api"
	"mosn.io/mosn/pkg/admin/store"
	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/metrics"
	"mosn.io/mosn/pkg/metrics/shm"
	"mosn.io/mosn/pkg/metrics/sink"
	"mosn.io/mosn/pkg/plugin"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/server/keeper"
	"mosn.io/mosn/pkg/trace"
	"mosn.io/mosn/pkg/types"
)

// Default Initialize wrappers. if more initialize needs to extend.
// modify it in main function
func DefaultInitialize(c *v2.MOSNConfig) {
	InitializePidFile(c)
	InitializeTracing(c)
	InitializePlugin(c)
	InitializeMetrics(c)
	InitializeThirdPartCodec(c)
}

// Init Stages Function
func InitializeTracing(c *v2.MOSNConfig) {
	initializeTracing(c.Tracing)
}

func initializeTracing(config v2.TracingConfig) {
	if config.Enable && config.Driver != "" {
		err := trace.Init(config.Driver, config.Config)
		if err != nil {
			log.StartLogger.Errorf("[mosn] [init tracing] init driver '%s' failed: %s, tracing functionality is turned off.", config.Driver, err)
			trace.Disable()
			return
		}
		log.StartLogger.Infof("[mosn] [init tracing] enable tracing")
		trace.Enable()
	} else {
		log.StartLogger.Infof("[mosn] [init tracing] disable tracing")
		trace.Disable()
	}
}

func InitializeMetrics(c *v2.MOSNConfig) {
	initializeMetrics(c.Metrics)
}

func initializeMetrics(config v2.MetricsConfig) {
	// init shm zone
	if config.ShmZone != "" && config.ShmSize > 0 {
		shm.InitDefaultMetricsZone(config.ShmZone, int(config.ShmSize), store.GetMosnState() != store.Active_Reconfiguring)
	}

	// set metrics package
	statsMatcher := config.StatsMatcher
	metrics.SetStatsMatcher(statsMatcher.RejectAll, statsMatcher.ExclusionLabels, statsMatcher.ExclusionKeys)
	metrics.SetMetricsFeature(config.FlushMosn, config.LazyFlush)
	// create sinks
	for _, cfg := range config.SinkConfigs {
		_, err := sink.CreateMetricsSink(cfg.Type, cfg.Config)
		// abort
		if err != nil {
			log.StartLogger.Errorf("[mosn] [init metrics] %s. %v metrics sink is turned off", err, cfg.Type)
			return
		}
		log.StartLogger.Infof("[mosn] [init metrics] create metrics sink: %v", cfg.Type)
	}
}

func InitializePidFile(c *v2.MOSNConfig) {
	initializePidFile(c.Pid)
}

func initializePidFile(pid string) {
	keeper.SetPid(pid)
}

func InitializePlugin(c *v2.MOSNConfig) {
	initializePlugin(c.Plugin.LogBase)
}

func initializePlugin(log string) {
	if log == "" {
		log = types.MosnLogBasePath
	}
	plugin.InitPlugin(log)
}

func InitializeThirdPartCodec(c *v2.MOSNConfig) {
	initializeThirdPartCodec(c.ThirdPartCodec)
}

func initializeThirdPartCodec(config v2.ThirdPartCodecConfig) {
	for _, codec := range config.Codecs {
		if !codec.Enable {
			log.StartLogger.Infof("[mosn] [init codec] third part codec disabled for %+v, skip...", codec.Path)
			continue
		}

		switch codec.Type {
		case v2.GoPlugin:
			if err := readProtocolPlugin(codec.Path, codec.LoaderFuncName); err != nil {
				log.StartLogger.Errorf("[mosn] [init codec] init go-plugin codec failed: %+v", err)
				continue
			}
			log.StartLogger.Infof("[mosn] [init codec] load go plugin codec succeed: %+v", codec.Path)

		case v2.Wasm:
			// todo
			log.StartLogger.Errorf("[mosn] [init codec] wasm codec not supported now.")

		default:
			log.StartLogger.Errorf("[mosn] [init codec] unknown third part codec type: %+v", codec.Type)
		}
	}
}

const (
	DefaultLoaderFunctionName string = "LoadCodec"
)

func readProtocolPlugin(path, loadFuncName string) error {
	p, err := goplugin.Open(path)
	if err != nil {
		return err
	}

	if loadFuncName == "" {
		loadFuncName = DefaultLoaderFunctionName
	}

	sym, err := p.Lookup(loadFuncName)
	if err != nil {
		return err
	}

	loadFunc := sym.(func() api.XProtocolCodec)
	codec := loadFunc()

	protocolName := codec.ProtocolName()
	log.StartLogger.Infof("[mosn] [init codec] loading protocol [%v] from third part codec", protocolName)

	if err := xprotocol.RegisterProtocol(protocolName, codec.XProtocol()); err != nil {
		return err
	}
	if err := xprotocol.RegisterMapping(protocolName, codec.HTTPMapping()); err != nil {
		return err
	}
	if err := xprotocol.RegisterMatcher(protocolName, codec.ProtocolMatch()); err != nil {
		return err
	}

	return nil
}
