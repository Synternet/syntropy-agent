[![made-with-Go](https://img.shields.io/badge/Made%20with-Go-1f425f.svg)](http://golang.org)
[![Build](https://github.com/SyntropyNet/syntropy-agent/actions/workflows/build.yml/badge.svg)](https://github.com/SyntropyNet/syntropy-agent/actions/workflows/build.yml)
[![Tests](https://github.com/SyntropyNet/syntropy-agent/actions/workflows/test.yml/badge.svg)](https://github.com/SyntropyNet/syntropy-agent/actions/workflows/test.yml)
[![GitHub release](https://img.shields.io/badge/release-releases-green)](https://GitHub.com/SyntropyNet/syntropy-agent/releases/)
[![GitHub license](http://img.shields.io/:license-mit-blue.svg?style=flat-square)](http://badges.mit-license.org)

# Syntropy Platform Agent ![logo](syntropy-logo.png)

## FAQ

### What is Syntropy Platform Agent
Syntropy Agent is an easy-to-use dependency to automatically encrypt and connect endpoints within a network. Full documentation [here](https://docs.syntropystack.com/docs/what-is-syntropy-agent).

### Why Syntropy Agent
Syntropy Agent allows you to easy setup encrypted network using a nice [WebUI](https://platform.syntropystack.com/) without getting your hands dirty with Wireguard and network/routes configuration. Also it constantly monitors configured network and chooses best SDN path automatically, taking into account packet loss and latency.

### How Syntropy Agent finds best path
It uses [DARP](https://darp.syntropystack.com).

### Where can I can find full documentation
Full, constantly maintained documentation can be found [here](https://docs.syntropystack.com/docs/start-syntropy-agent).

### How do I know which Agent version I am running
* Running plain binary on bare-metal:
 ```syntropy-agent -version```
* Running Docker container:
 ```docker logs `docker ps | grep syntropynet\/agent | cut -b1-10` | grep started```

### Why GO
Every programming language has pros and cons, but motivation for GO is:
* allows quickly and easily refactor code and make big changes fast. That's a huge benefit for projects that are in active development stage;
* is very effective and uses less resources if compared with scriptable languages;
* compiles to single binary without dependencies;
* is quite simple language and in this project we like [KISS principle](https://en.wikipedia.org/wiki/KISS_principle).

### Why don't you rewrite it in Rust
It may or may not happen in future. But right now see [Why GO](#why-go) and [why not Rewrite It In Rust](https://github.com/ansuz/RIIR).

### I think this project would benefit from "feature X"
Thanks. Propose your idea in [issues](https://github.com/SyntropyNet/syntropy-agent/issues).

### I've found a bug and have a fix for it
Thanks. Create a fork of this project, fix a bug and submit a *Merge Request* for the review.

### I've found a bug and don't have a fix for it
Thanks. Submit a bug report in [issues](https://github.com/SyntropyNet/syntropy-agent/issues).