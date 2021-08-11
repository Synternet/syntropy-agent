// Package netfilter wraps iptables commands
// and is used to setup Syntropy releated rules
package netfilter

import (
	"log"

	"github.com/coreos/go-iptables/iptables"
	"github.com/vishvananda/netlink"
)

// TODO: review `-nft` and `-legacy` usage

const (
	defaultTable  = "filter"
	natTable      = "nat"
	syntropyChain = "SYNTROPY_CHAIN"
)

func CreateChain() error {
	rule := []string{"-s", "0.0.0.0/0", "-d", "0.0.0.0/0", "-j", syntropyChain}

	ipt, err := iptables.New()
	if err != nil {
		return err
	}

	exists, err := ipt.ChainExists(defaultTable, syntropyChain)
	if !exists && err == nil {
		err = ipt.NewChain(defaultTable, syntropyChain)
	}
	if err != nil {
		return err
	}

	exists, err = ipt.Exists(defaultTable, syntropyChain, rule...)
	if !exists && err == nil {
		err = ipt.Insert(defaultTable, syntropyChain, 1, rule...)
	}
	if err != nil {
		return err
	}

	return nil
}

func processPeerRule(ipt *iptables.IPTables, add bool, ip string) (err error) {
	rule := []string{"-p", "all", "-s", ip, "-j", "ACCEPT"}
	if add {
		err = ipt.AppendUnique(defaultTable, syntropyChain, rule...)
	} else {
		err = ipt.DeleteIfExists(defaultTable, syntropyChain, rule...)
	}
	return err
}

func RulesAdd(ips ...string) error {
	ipt, err := iptables.New()
	if err != nil {
		return err
	}
	for _, ip := range ips {
		err := processPeerRule(ipt, true, ip)
		if err != nil {
			log.Println(err)
			// TODO fallback to last known good state ?
			// TODO or maybe try adding all rules, and return concatenated errors ?
			return err
		}
	}
	return nil
}

func RulesDel(ips ...string) error {
	ipt, err := iptables.New()
	if err != nil {
		return err
	}
	for _, ip := range ips {
		err := processPeerRule(ipt, false, ip)
		if err != nil {
			log.Println(err)
			// TODO fallback to last known good state ?
			// TODO or maybe try adding all rules, and return concatenated errors ?
			return err
		}
	}
	return nil
}

func ForwardEnable(ifname string) error {
	ipt, err := iptables.New()
	if err != nil {
		return err
	}
	forwardRule := []string{"-i", ifname, "-j", "ACCEPT"}
	err = ipt.AppendUnique(defaultTable, "FORWARD", forwardRule...)
	if err != nil {
		return err
	}

	dri := defaultRouteIfname()
	if dri != "" {
		masquaradeRule := []string{"-o", defaultRouteIfname(), "-j", "MASQUERADE"}
		err = ipt.AppendUnique(natTable, "POSTROUTING", masquaradeRule...)
		if err != nil {
			return err
		}
	}
	return nil
}

func defaultRouteIfname() string {
	var ifname string
	var defaultRoute *netlink.Route
	const AF_INET = 2 // from include/linux/socket.h. Is there a better way to use constant ?

	routes, err := netlink.RouteList(nil, AF_INET)
	if err != nil {
		log.Println(err)
		return ifname
	}

	for _, r := range routes {
		if r.Dst == nil {
			if defaultRoute == nil || defaultRoute.Priority > r.Priority {
				defaultRoute = &r
			}

		}
	}

	if defaultRoute != nil {
		l, err := netlink.LinkByIndex(defaultRoute.LinkIndex)
		if err == nil && l != nil {
			ifname = l.Attrs().Name
		}
	}

	return ifname
}
