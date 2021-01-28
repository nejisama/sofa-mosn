package mosn

import (
	admin "mosn.io/mosn/pkg/admin/server"
	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/mosn"
)

type MosnWrapper struct {
	m *mosn.Mosn
}

// This is a wrapper for main
func NewMosn(c *v2.MOSNConfig) *MosnWrapper {
	mosn.DefaultInitialize(c)
	m := mosn.NewMosn(c)
	return &MosnWrapper{
		m: m,
	}
}

// see details in cmd/mosn/main/control.go
func (m *MosnWrapper) Start() {
	m.m.StartXdsClient()
	m.m.HandleExtendConfig()
	srv := admin.Server{}
	srv.Start(m.m.Config)
	m.m.TransferConnection()
	m.m.CleanUpgrade()
	m.m.Start()
}

func (m *MosnWrapper) Close() {
	m.m.Close()
}
