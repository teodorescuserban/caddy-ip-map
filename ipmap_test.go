package ipmap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func TestHandler(t *testing.T) {
	for i, tc := range []struct {
		handler  Handler
		someCIDR string
		expect   map[string]any
	}{
		{
			someCIDR: "10.0.0.1",
			handler: Handler{
				Source:       "{http.request.header.x-test-input}",
				Destinations: []string{"{output}"},
				Mappings: []Mapping{
					{
						Input:   "10.0.0.1",
						Outputs: []any{"FOO"},
					},
				},
			},
			expect: map[string]any{
				"output": "FOO",
			},
		},
		{
			someCIDR: "10.0.0.2",
			handler: Handler{
				Source:       "{http.request.header.x-test-input}",
				Destinations: []string{"{output}"},
				Defaults:     []string{"default"},
				Mappings: []Mapping{
					{
						Input:   "10.0.0.1",
						Outputs: []any{"FOO"},
					},
				},
			},
			expect: map[string]any{
				"output": "default",
			},
		},
		{
			someCIDR: "10.0.0.1",
			handler: Handler{
				Source:       "{http.request.header.x-test-input}",
				Destinations: []string{"{output}"},
				Mappings: []Mapping{
					{
						Input:   "10.0.0.0/25",
						Outputs: []any{"ABC"},
					},
				},
			},
			expect: map[string]any{
				"output": "ABC",
			},
		}, {
			someCIDR: "10.0.0.1",
			handler: Handler{
				Source:       "{http.request.header.x-test-input}",
				Destinations: []string{"{output}"},
				Defaults:     []string{"default"},
				Mappings: []Mapping{
					{
						Input:   "10.0.1.0/25",
						Outputs: []any{"ABC"},
					},
				},
			},
			expect: map[string]any{
				"output": "default",
			},
		},
		{
			someCIDR: "192.168.225.17",
			handler: Handler{
				Source:       "{http.request.header.x-test-input}",
				Destinations: []string{"{output}"},
				Mappings: []Mapping{
					{
						Input:   "192.168.225.16/28",
						Outputs: []any{"...xyz..."},
					},
				},
			},
			expect: map[string]any{
				"output": "...xyz...",
			},
		},
		{
			someCIDR: "192.168.225.17",
			handler: Handler{
				Source:       "{http.request.header.x-test-input}",
				Destinations: []string{"{output}"},
				Defaults:     []string{"default"},
				Mappings: []Mapping{
					{
						Input:   "192.168.225.0/28",
						Outputs: []any{"...xyz..."},
					},
				},
			},
			expect: map[string]any{
				"output": "default",
			},
		},
		{
			someCIDR: "127.0.0.1",
			handler: Handler{
				Source:       "{http.request.header.x-test-input}",
				Destinations: []string{"{output}"},
				Mappings: []Mapping{
					{
						Input:   "127.0.0.0/8",
						Outputs: []any{"{testvar}"},
					},
				},
			},
			expect: map[string]any{
				"output": "testing",
			},
		},
	} {
		if err := tc.handler.Provision(caddy.Context{}); err != nil {
			t.Fatalf("Test %d: Provisioning handler: %v", i, err)
		}
		req, err := http.NewRequest(http.MethodGet, "/", nil)
		if err != nil {
			t.Fatalf("Test %d: Creating request: %v", i, err)
		}
		req.Header.Set("X-Test-Input", tc.someCIDR)

		repl := caddyhttp.NewTestReplacer(req)
		repl.Set("testvar", "testing")
		ctx := context.WithValue(req.Context(), caddy.ReplacerCtxKey, repl)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		noop := caddyhttp.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) error { return nil })

		if err := tc.handler.ServeHTTP(rr, req, noop); err != nil {
			t.Errorf("Test %d: Handler returned error: %v", i, err)
			continue
		}

		for key, expected := range tc.expect {
			actual, _ := repl.Get(key)
			if !reflect.DeepEqual(actual, expected) {
				t.Errorf("Test %d: Expected %#v but got %#v for {%s}", i, expected, actual, key)
			}
		}
	}
}
