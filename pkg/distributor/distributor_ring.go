// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/cortexproject/cortex/blob/master/pkg/distributor/distributor_ring.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Cortex Authors.

package distributor

import (
	"flag"
	"os"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/dskit/kv"
	"github.com/grafana/dskit/netutil"
	"github.com/grafana/dskit/ring"

	util_log "github.com/grafana/mimir/pkg/util/log"
)

// RingConfig masks the ring lifecycler config which contains
// many options not really required by the distributors ring. This config
// is used to strip down the config to the minimum, and avoid confusion
// to the user.
type RingConfig struct {
	KVStore          kv.Config     `yaml:"kvstore"`
	HeartbeatPeriod  time.Duration `yaml:"heartbeat_period" category:"advanced"`
	HeartbeatTimeout time.Duration `yaml:"heartbeat_timeout" category:"advanced"`

	// Instance details
	InstanceID             string   `yaml:"instance_id" doc:"default=<hostname>" category:"advanced"`
	InstanceInterfaceNames []string `yaml:"instance_interface_names" doc:"default=[<private network interfaces>]"`
	InstancePort           int      `yaml:"instance_port" category:"advanced"`
	InstanceAddr           string   `yaml:"instance_addr" category:"advanced"`

	// Injected internally
	ListenPort int `yaml:"-"`
}

// RegisterFlags adds the flags required to config this to the given FlagSet
func (cfg *RingConfig) RegisterFlags(f *flag.FlagSet, logger log.Logger) {
	hostname, err := os.Hostname()
	if err != nil {
		level.Error(util_log.Logger).Log("msg", "failed to get hostname", "err", err)
		os.Exit(1)
	}

	// Ring flags
	cfg.KVStore.Store = "memberlist"
	cfg.KVStore.RegisterFlagsWithPrefix("distributor.ring.", "collectors/", f)
	f.DurationVar(&cfg.HeartbeatPeriod, "distributor.ring.heartbeat-period", 5*time.Second, "Period at which to heartbeat to the ring. 0 = disabled.")
	f.DurationVar(&cfg.HeartbeatTimeout, "distributor.ring.heartbeat-timeout", time.Minute, "The heartbeat timeout after which distributors are considered unhealthy within the ring. 0 = never (timeout disabled).")

	// Instance flags
	cfg.InstanceInterfaceNames = netutil.PrivateNetworkInterfacesWithFallback([]string{"eth0", "en0"}, logger)
	f.Var((*flagext.StringSlice)(&cfg.InstanceInterfaceNames), "distributor.ring.instance-interface-names", "List of network interface names to look up when finding the instance IP address.")
	f.StringVar(&cfg.InstanceAddr, "distributor.ring.instance-addr", "", "IP address to advertise in the ring. Default is auto-detected.")
	f.IntVar(&cfg.InstancePort, "distributor.ring.instance-port", 0, "Port to advertise in the ring (defaults to -server.grpc-listen-port).")
	f.StringVar(&cfg.InstanceID, "distributor.ring.instance-id", hostname, "Instance ID to register in the ring.")
}

// ToLifecyclerConfig returns a LifecyclerConfig based on the distributor
// ring config.
func (cfg *RingConfig) ToLifecyclerConfig() ring.LifecyclerConfig {
	// We have to make sure that the ring.LifecyclerConfig and ring.Config
	// defaults are preserved
	lc := ring.LifecyclerConfig{}
	rc := ring.Config{}

	flagext.DefaultValues(&lc)
	flagext.DefaultValues(&rc)

	// Configure ring
	rc.KVStore = cfg.KVStore
	rc.HeartbeatTimeout = cfg.HeartbeatTimeout
	rc.ReplicationFactor = 1

	// Configure lifecycler
	lc.RingConfig = rc
	lc.ListenPort = cfg.ListenPort
	lc.Addr = cfg.InstanceAddr
	lc.Port = cfg.InstancePort
	lc.ID = cfg.InstanceID
	lc.InfNames = cfg.InstanceInterfaceNames
	lc.UnregisterOnShutdown = true
	lc.HeartbeatPeriod = cfg.HeartbeatPeriod
	lc.HeartbeatTimeout = cfg.HeartbeatTimeout
	lc.ObservePeriod = 0
	lc.NumTokens = 1
	lc.JoinAfter = 0
	lc.MinReadyDuration = 0
	lc.FinalSleep = 0

	return lc
}

func (cfg *RingConfig) ToRingConfig() ring.Config {
	rc := ring.Config{}
	flagext.DefaultValues(&rc)

	rc.KVStore = cfg.KVStore
	rc.HeartbeatTimeout = cfg.HeartbeatTimeout
	rc.ReplicationFactor = 1

	return rc
}
