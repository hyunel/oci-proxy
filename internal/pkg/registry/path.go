package registry

import (
	"slices"
	"strings"
)

type Path struct {
	Host string
	Name string
	Kind string
	Rest string
}

var kinds = map[string]bool{"manifests": true, "blobs": true, "tags": true, "referrers": true}

func Parse(path string) Path {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] != "v2" {
		return Path{}
	}
	parts = parts[1:]

	// scanned from the right so a repository component named "blobs" or "tags" stays in the name
	for i := len(parts) - 1; i > 0; i-- {
		if kinds[parts[i]] {
			return Path{
				Name: strings.Join(parts[:i], "/"),
				Kind: parts[i],
				Rest: strings.Join(parts[i+1:], "/"),
			}
		}
	}
	return Path{Name: strings.Join(parts, "/")}
}

func (p Path) SplitHost() Path {
	first, rest, _ := strings.Cut(p.Name, "/")

	// same rule as the docker CLI: only a domain may hold ".", ":" or uppercase
	if strings.ContainsAny(first, ".:") || first == "localhost" || strings.ToLower(first) != first {
		p.Host, p.Name = first, rest
	}
	return p
}

func (p Path) UpstreamPath(host string) string {
	name := p.Name
	if name == "" {
		return "/v2/"
	}
	if p.Kind != "" && !strings.Contains(name, "/") && isDockerHub(host) {
		name = "library/" + name
	}
	segments := slices.DeleteFunc([]string{"v2", name, p.Kind, p.Rest}, func(s string) bool { return s == "" })
	return "/" + strings.Join(segments, "/")
}

func isDockerHub(host string) bool {
	switch host {
	case "docker.io", "index.docker.io", "registry-1.docker.io", "registry.hub.docker.com":
		return true
	}
	return false
}
