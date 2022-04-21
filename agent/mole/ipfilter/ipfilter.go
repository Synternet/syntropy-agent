// Package ipfilter wraps iptables commands
// and is used to setup Syntropy releated rules
package ipfilter

import (
	"errors"
	"fmt"

	"github.com/SyntropyNet/syntropy-agent/internal/logger"
	"github.com/SyntropyNet/syntropy-agent/pkg/iptables"
	"github.com/SyntropyNet/syntropy-agent/pkg/netcfg"
)

// TODO: review `-nft` and `-legacy` usage

type PacketFilter struct {
	ipt          *iptables.IPTables
	chainCreated bool
}

func New() (*PacketFilter, error) {
	pf := new(PacketFilter)
	var err error

	pf.ipt, err = iptables.New(iptables.IPFamily(iptables.ProtocolIPv4), iptables.IptVariant(iptables.Legacy))
	if err == nil {
		return pf, nil
	}

	logger.Error().Println(pkgName, "iptables-legacy failed. Trying iptables-nft")
	pf.ipt, err = iptables.New(iptables.IPFamily(iptables.ProtocolIPv4), iptables.IptVariant(iptables.Nftables))
	if err == nil {
		return pf, nil
	}

	logger.Error().Println(pkgName, "iptables-nft failed. Fallback to OS default iptables")

	pf.ipt, err = iptables.New(iptables.IPFamily(iptables.ProtocolIPv4), iptables.IptVariant(iptables.Default))
	if err == nil {
		return pf, nil
	}

	logger.Error().Println(pkgName, "Default iptables failed", err)
	return nil, fmt.Errorf("iptables failed")
}

const (
	pkgName       = "IpTables. "
	defaultTable  = "filter"
	natTable      = "nat"
	forwardChain  = "FORWARD"
	syntropyChain = "SYNTROPY_CHAIN"
)

func (pf *PacketFilter) CreateChain() error {
	rule := []string{"-s", "0.0.0.0/0", "-d", "0.0.0.0/0", "-j", syntropyChain}

	exists, err := pf.ipt.ChainExists(defaultTable, syntropyChain)
	if !exists && err == nil {
		err = pf.ipt.NewChain(defaultTable, syntropyChain)
	}
	if err != nil {
		return err
	}

	exists, err = pf.ipt.Exists(defaultTable, forwardChain, rule...)
	if !exists && err == nil {
		err = pf.ipt.Insert(defaultTable, forwardChain, 1, rule...)
	}
	if err != nil {
		return err
	}

	pf.chainCreated = true
	return nil
}

func (pf *PacketFilter) processPeerRule(add bool, ip string) error {
	// No need adding rules to non existing chain
	if !pf.chainCreated {
		return nil
	}

	var err error
	rule := []string{"-p", "all", "-s", ip, "-j", "ACCEPT"}
	if add {
		err = pf.ipt.AppendUnique(defaultTable, syntropyChain, rule...)
	} else {
		err = pf.ipt.DeleteIfExists(defaultTable, syntropyChain, rule...)
	}
	return err
}

func (pf *PacketFilter) RulesAdd(ips ...string) error {
	// No need adding rules to non existing chain
	if !pf.chainCreated {
		return nil
	}

	for _, ip := range ips {
		err := pf.processPeerRule(true, ip)
		if err != nil {
			return err
		}
	}
	return nil
}

func (pf *PacketFilter) RulesDel(ips ...string) error {
	// No need adding rules to non existing chain
	if !pf.chainCreated {
		return nil
	}

	for _, ip := range ips {
		err := pf.processPeerRule(false, ip)
		if err != nil {
			return err
		}
	}
	return nil
}

func (pf *PacketFilter) ForwardEnable(ifname string) error {
	forwardRule := []string{"-i", ifname, "-j", "ACCEPT"}
	err := pf.ipt.AppendUnique(defaultTable, "FORWARD", forwardRule...)
	if err != nil {
		return err
	}

	_, dri, _ := netcfg.DefaultRoute()
	if dri == "" {
		return errors.New("could not parse default route interface")
	}

	// TODO correctly handle masquarate in nf_tables case
	masquaradeRule := []string{"-o", dri, "-j", "MASQUERADE"}
	return pf.ipt.AppendUnique(natTable, "POSTROUTING", masquaradeRule...)
}

func (pf *PacketFilter) Close() error {
	// TODO: cleanup configured iptables rules on exit
	return nil
}

func (pf *PacketFilter) Flush() {
	// TODO: Flush is called when new configuration is received.
	// Think about marking roles for deletion
	// NB: need to introduce some kind of cache
}
