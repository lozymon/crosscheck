package auth

import (
	"fmt"

	"github.com/lozymon/crosscheck/config"
	"github.com/lozymon/crosscheck/httpclient"
	"github.com/lozymon/crosscheck/interpolate"
)

// Result holds what auth produced — vars to merge into the test namespace
// and a header to inject into every subsequent request.
type Result struct {
	Vars   map[string]string // captured vars (e.g. token → "abc123")
	Header string            // header name  (e.g. "Authorization")
	Value  string            // resolved header value (e.g. "Bearer abc123")
}

// Resolve executes the auth block and returns a Result.
// Returns nil, nil when auth is nil.
func Resolve(auth *config.Auth, client *httpclient.Client, vars map[string]string) (*Result, error) {
	if auth == nil {
		return nil, nil
	}
	switch auth.Type {
	case "static":
		return resolveStatic(auth, vars)
	case "login":
		return resolveLogin(auth, client, vars)
	default:
		return nil, fmt.Errorf("unknown auth type %q: must be \"static\" or \"login\"", auth.Type)
	}
}

func resolveStatic(auth *config.Auth, vars map[string]string) (*Result, error) {
	return &Result{
		Vars:   nil,
		Header: auth.Inject.Header,
		Value:  interpolate.Apply(auth.Inject.Format, vars),
	}, nil
}

func resolveLogin(auth *config.Auth, client *httpclient.Client, vars map[string]string) (*Result, error) {
	if auth.Request == nil {
		return nil, fmt.Errorf("auth type \"login\" requires a request block")
	}

	resp, err := client.Do(auth.Request, vars)
	if err != nil {
		return nil, fmt.Errorf("auth login request: %w", err)
	}

	captured := make(map[string]string, len(auth.Capture))
	for varName, path := range auth.Capture {
		result := resp.Get(path)
		if !result.Exists() {
			return nil, fmt.Errorf("auth capture %q: path %q not found in response (status %d body: %s)",
				varName, path, resp.Status, resp.BodyString())
		}
		captured[varName] = result.String()
	}

	// Merge captured into a local copy of vars so the inject format can reference them.
	merged := make(map[string]string, len(vars)+len(captured))
	for k, v := range vars {
		merged[k] = v
	}
	for k, v := range captured {
		merged[k] = v
	}

	return &Result{
		Vars:   captured,
		Header: auth.Inject.Header,
		Value:  interpolate.Apply(auth.Inject.Format, merged),
	}, nil
}
