# Change Log
All notable changes to this project will be documented in this file.
 
The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).


## 0.4.0 - Prometheus exporter + routes deletion
* Prometheus exporter
* Route deletion when peer is unreachable
* Some more configuration variables
* Configuration example in config.env
* Bugfix: reporting kubernetes services to controller.

## 0.3.1 - Default route hotfix
* Bugfix default route parsing

## 0.3.0 - SDN router
* Service Router refactor. Cache settings and sync cache to OS settings.
* Fix removed services and/or peers route deletion and cleanup.
* Reroute thresholds.
* Bugfix peer monitor and best path selection.
* Make ping loss chart more smooth.
* Send full version to controller.
* Detect NAT and send back port=0 to controller. This turns on SND Agent port autodetect.
* CI/CD and Dockerfile improovements. 

## 0.2.1 - Hotfix multipple IP addresses issue
* When reusing already created interface remove resisual IP addresses.
* Attempt to guess and solve residual routes when reusing previously created interface.
* Gracefully handle agent deletion in UI. Cleanup created tunnels and exit.

## 0.2.0 - STUN + own pinger package
* STUN is primary method for public IP resolving. WEB is fallback, if STUN ports are blocked.
* Ping (ICMP) library replacement. Improve ping results robustness
* Improvement: use moving average for better SDN path calculation (reduce jitter)
* Bugfix: Remove minimal tag length validation
* Bugfix: router do not add local interface IP address as remote hop IP when adding routes
* Bugfix: docker names fix (strip prepending '/')
* Some other small fixes and fine touches.
 
## 0.1.0 - First stable release
* Controller revert to 0.0.13 version
* Wireguard cache fix
* prev_connection_id in IFACES_PEERS_ACTIVE_DATA
* Bugfix docker json fields