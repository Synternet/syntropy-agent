// Package netfilter wraps iptables commands
// and is used to setup Syntropy releated rules
package netfilter

import "log"

// TODO: review `-nft` and `-legacy` usage

func CreateChain() error {
	log.Println("iptables creating SYNTROPY_CHAIN")
	return nil
}

func RulesAdd(ips ...string) error {
	log.Println("iptables add ", ips)
	return nil
}

func RulesDel(ips ...string) error {
	log.Println("iptables del ", ips)
	return nil
}

func ForwardEnable(ifname string) error {
	log.Println("iptables MASQ ", ifname, defaultRouteIfname())
	return nil
}

func defaultRouteIfname() string {
	var ifname string

	return ifname
}
