# caddy-ip-map

![Build and test](https://github.com/teodorescuserban/caddy-ip-map/actions/workflows/test.yml/badge.svg)
[![Project Status: WIP – Initial development is in progress, but there has not yet been a stable, usable release suitable for the public.](https://www.repostatus.org/badges/latest/wip.svg)](https://www.repostatus.org/#wip)
[![GoDoc](http://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/teodorescuserban/caddy-ip-map)
[![Go Report Card](https://goreportcard.com/badge/github.com/teodorescuserban/caddy-ip-map)](https://goreportcard.com/report/github.com/teodorescuserban/caddy-ip-map)

This is a caddy plugin. Works with caddy >= 2.8.0.
Sort the request query arguments. Optionally case insensitive.

## Usage

This plugin duplicates the `map` functionality but limits the type of the source variable to IP or subnet (CIDR).

```
ipmap [<matcher>] <source> <destinations...> {
                  <input>  <outputs...>
                  default  <defaults...>
}
```

For `input` you can use:
- `default` to set the default values.
- IP simple notation: `10.11.12.13`
- IP CIDR notation: `10.11.12.13/32`
- subnets in CIDR notation: `10.11.12.128/25`

### Example usage

```caddyfile
:8881 {
    ipmap {client_ip} {var.foo_var} {var.bar_var} {
		default "0" "unknown"
		10.0.0.0/8 "10" "internal network"
		172.16.0.0/12 "172" "vpn network"
		3.3.3.5 "3" "some 3 address"
		3.3.3.0/27 "3" "some small 3 subnet"
		127.0.0.1 "lol" "insider"
		::1/128 "IPv6" "Amagad"
		1.2.3.4/5 "ab" "cdefg"
	}

    respond "Client Type: {var.foo_var}, Description: {var.bar_var}"
}
```

