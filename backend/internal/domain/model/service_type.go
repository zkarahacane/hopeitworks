package model

import "strings"

// Service type identifiers for sidecar services declared in an Environment. The
// type is detected from the service image name (registry/tag stripped) and is the
// single key both the Docker readiness probe (sidecar_manager) and the run-path
// connection-string injection (agent_run) hang their per-type knowledge off of.
// Keeping detection + default ports here (domain, infra-free) avoids duplicating
// the mapping across adapters.
const (
	ServiceTypePostgres      = "postgres"
	ServiceTypeRedis         = "redis"
	ServiceTypeMySQL         = "mysql"
	ServiceTypeMariaDB       = "mariadb"
	ServiceTypeMongo         = "mongo"
	ServiceTypeElasticsearch = "elasticsearch"
	ServiceTypeMailHog       = "mailhog"
)

// servicePorts is the default listen port for each known service type. It is the
// single source of truth for ports, reused by the Docker readiness probe and the
// connection-string builder.
var servicePorts = map[string]int{
	ServiceTypePostgres:      5432,
	ServiceTypeRedis:         6379,
	ServiceTypeMySQL:         3306,
	ServiceTypeMariaDB:       3306,
	ServiceTypeMongo:         27017,
	ServiceTypeElasticsearch: 9200,
	ServiceTypeMailHog:       1025,
}

// DetectServiceType maps an image reference to a known service type, or "" when
// unknown. It matches on the repository segment of the image (ignoring registry
// host and tag/digest), e.g. "docker.io/library/postgres:16" -> "postgres".
func DetectServiceType(image string) string {
	ref := image
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		ref = ref[i+1:]
	}
	if i := strings.IndexAny(ref, ":@"); i >= 0 {
		ref = ref[:i]
	}
	ref = strings.ToLower(ref)
	if _, ok := servicePorts[ref]; ok {
		return ref
	}
	return ""
}

// ServicePort returns the default listen port for a detected service type, or 0
// when the type is unknown.
func ServicePort(svcType string) int {
	return servicePorts[svcType]
}
