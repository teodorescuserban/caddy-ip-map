// Copyright 2015 Matthew Holt and The Caddy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ipmap

import (
	"fmt"
	"net/http"
	"net/netip"
	"slices"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
	caddy.RegisterModule(Handler{})
}

// Handler implements a middleware that maps IP / Subnets as inputs to outputs.
// Specifically, it compares a source value against the map inputs, and for one
// that matches, it applies the output values to each destination. Destinations
// become placeholder names.
//
// Mapped placeholders are not evaluated until they are used, so even for very
// large mappings, this handler is quite efficient.
type Handler struct {
	// Source is the placeholder from which to get the input value.
	Source string `json:"source,omitempty"`

	// Destinations are the names of placeholders in which to store the outputs.
	// Destination values should be wrapped in braces, for example, {my_placeholder}.
	Destinations []string `json:"destinations,omitempty"`

	// Mappings from IP / Subnet source values (inputs) to destination values (outputs).
	// The first matching, non-nil mapping will be applied.
	Mappings []Mapping `json:"mappings,omitempty"`

	// If no mappings match or if the mapped output is null/nil, the associated
	// default output will be applied (optional).
	Defaults []string `json:"defaults,omitempty"`
}

// CaddyModule returns the Caddy module information.
func (Handler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.ipmap",
		New: func() caddy.Module { return new(Handler) },
	}
}

// Provision sets up h.
func (h *Handler) Provision(_ caddy.Context) error {
	for j, dest := range h.Destinations {
		if strings.Count(dest, "{") != 1 || !strings.HasPrefix(dest, "{") {
			return fmt.Errorf("destination must be a placeholder and only a placeholder")
		}
		h.Destinations[j] = strings.Trim(dest, "{}")
	}

	// TODO: improve efficiency even further by using an actual map type
	// OR sort them and do a binary search

	return nil
}

// Validate ensures that h is configured properly.
func (h *Handler) Validate() error {
	nDest, nDef := len(h.Destinations), len(h.Defaults)
	if nDef > 0 && nDef != nDest {
		return fmt.Errorf("%d destinations != %d defaults", nDest, nDef)
	}

	seen := make(map[string]int)
	for i, m := range h.Mappings {
		// caddy.Log().Named("ipmap").Info("Input", zap.String("source", m.Input))
		input := m.Input
		// prevent invalid IP / Subnet
		_, err := netip.ParseAddr(input)
		if err != nil {
			_, err := netip.ParsePrefix(input)
			if err != nil {
				return fmt.Errorf("mapping %d has an invalid input '%s'; it doesn't look like an IP or subnet", i, input)
			}
		}
		if prev, ok := seen[input]; ok {
			return fmt.Errorf("mapping %d has a duplicate input '%s' previously used with mapping %d", i, input, prev)
		}
		seen[input] = i

		// ensure mappings have 1:1 output-to-destination correspondence
		nOut := len(m.Outputs)
		if nOut != nDest {
			return fmt.Errorf("mapping %d has %d outputs but there are %d destinations defined", i, nOut, nDest)
		}
	}

	return nil
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	repl := r.Context().Value(caddy.ReplacerCtxKey).(*caddy.Replacer)

	// defer work until a variable is actually evaluated by using replacer's Map callback
	repl.Map(func(key string) (any, bool) {
		// return early if the variable is not even a configured destination
		destIdx := slices.Index(h.Destinations, key)
		if destIdx < 0 {
			return nil, false
		}

		input := repl.ReplaceAll(h.Source, "")

		// find the first mapping matching the input and return
		// the requested destination/output value
		for _, m := range h.Mappings {
			output := m.Outputs[destIdx]
			if output == nil {
				continue
			}
			outputStr := caddy.ToString(output)

			// this is where we see which is which
			ipMatched, _ := checkIpInSubnet(input, m.Input)

			if ipMatched {
				return repl.ReplaceAll(outputStr, ""), true
			}
		}

		// fall back to default if no match or if matched nil value
		if len(h.Defaults) > destIdx {
			return repl.ReplaceAll(h.Defaults[destIdx], ""), true
		}

		return nil, true
	})

	return next.ServeHTTP(w, r)
}

func checkIpInSubnet(i string, s string) (bool, error) {

	ip, err := netip.ParseAddr(i)

	if err != nil {
		return false, fmt.Errorf("can't parse the input as IP Address (%s)", i)
	}

	// try to parse the source as IP address
	ipMaybe, err := netip.ParseAddr(s)
	if err == nil {
		if ip == ipMaybe {
			return true, nil
		}
	} else {
		// then it should be subnet, right?
		subnet, err := netip.ParsePrefix(s)
		if err != nil {
			return false, fmt.Errorf("can't parse the source as IP Address or subnet (%s)", s)
		}
		if subnet.Contains(ip) {
			return true, nil
		}
	}
	return false, nil
}

// Mapping describes a mapping from input to outputs.
type Mapping struct {
	// The input value to match. Must be distinct from other mappings.
	// Needs to be an IP / Subnet
	Input string `json:"input,omitempty"`

	// Upon a match with the input, each output is positionally correlated
	// with each destination of the parent handler. An output that is null
	// (nil) will be treated as if it was not mapped at all.
	Outputs []any `json:"outputs,omitempty"`
}

// Interface guards
var (
	_ caddy.Provisioner           = (*Handler)(nil)
	_ caddy.Validator             = (*Handler)(nil)
	_ caddyhttp.MiddlewareHandler = (*Handler)(nil)
)
