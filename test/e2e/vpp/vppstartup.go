package vpp

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"text/template"
)

var vppStartupTemplateStr = `
unix {
  nodaemon
  coredump-size unlimited
  full-coredump
  cli-listen {{.CLISock}}
  log {{.VPPLog}}
}

socksvr {
  socket-name {{.APISock}}
}

api-segment {
  prefix {{.APIPrefix}}
}

api-trace {
  on
}

cpu {
  main-core {{.MainCore}}
{{ if .Multicore }}
  corelist-workers {{.WorkerCore}}
{{ else }}
  workers 0
{{ end }}
}

heapsize 2G

statseg {
  socket-name {{.StatsSock}}
  size 512M
}

plugins {
  path {{.PluginPath}}
  plugin dpdk_plugin.so { disable }
  plugin gtpu_plugin.so { disable }
}

`

// vlib {
// 	elog-events 10000000
// 	elog-post-mortem-dump
// }

var startupTemplate *template.Template
var vppIndex int32

// Cores list the logical CPU cores that can be used for VPP.
// It is set in SynchronizedBeforeSuite()
var Cores []int = []int{0, 1}

func init() {
	var err error
	startupTemplate, err = template.New("test").Parse(vppStartupTemplateStr)
	if err != nil {
		panic(err)
	}
}

type VPPStartupConfig struct {
	BinaryPath    string
	PluginPath    string
	CLISock       string
	APISock       string
	StatsSock     string
	VPPLog        string
	APIPrefix     string
	MainCore      int
	WorkerCore    int
	UseGDB        bool
	Trace         bool
	DispatchTrace bool
	Multicore     bool
	XDP           bool
}

func (cfg *VPPStartupConfig) SetFromEnv() {
	binPath := os.Getenv("VPP_PATH")
	if binPath != "" {
		cfg.BinaryPath = binPath
	}
	pluginPath := os.Getenv("VPP_PLUGIN_PATH")
	if pluginPath != "" {
		cfg.PluginPath = pluginPath
	}
	cfg.UseGDB = os.Getenv("VPP_NO_GDB") == ""
	cfg.Trace = os.Getenv("VPP_TRACE") != ""
	cfg.DispatchTrace = os.Getenv("VPP_DISPATCH_TRACE") != ""
	cfg.Multicore = os.Getenv("VPP_MULTICORE") != ""
	cfg.XDP = os.Getenv("VPP_XDP") != ""
	cfg.SetDefaults()
}

func (cfg *VPPStartupConfig) SetDefaults() {
	if cfg.BinaryPath == "" {
		cfg.BinaryPath = "/usr/bin/vpp"
	}
	if cfg.PluginPath == "" {
		cfg.PluginPath = "/usr/lib/x86_64-linux-gnu/vpp_plugins"
	}
	if cfg.CLISock == "" {
		cfg.CLISock = "/run/vpp/cli.sock"
	}
	if cfg.APISock == "" {
		cfg.APISock = "/run/vpp/api.sock"
	}
	if cfg.StatsSock == "" {
		cfg.StatsSock = "/run/vpp/stats.sock"
	}
	if cfg.VPPLog == "" {
		cfg.VPPLog = "/dev/null"
	}
	if cfg.APIPrefix == "" {
		cfg.APIPrefix = fmt.Sprintf("vpp%d", atomic.AddInt32(&vppIndex, 1))
	}
}

func (cfg VPPStartupConfig) Get() string {
	var b strings.Builder
	if err := startupTemplate.Execute(&b, cfg); err != nil {
		panic(err)
	}
	return b.String()
}

func (cfg VPPStartupConfig) DefaultMTU() int {
	if cfg.XDP {
		// TODO: this is the max MTU value which works for veths.
		// Find out why & whether it's always the case
		// (it may perhaps depend on the kernel version, etc.)
		return 2034
	} else {
		return 9000
	}
}
