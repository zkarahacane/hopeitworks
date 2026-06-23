package action

import (
	"context"
	stderrors "errors"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// resolveEnvironment fetches the project's single Environment for the run path.
//
// Back-compat is HARD: a project that has no Environment yields NotFound from the
// repository, which is NOT an error here — it returns (nil, nil) so the run
// behaves exactly as it did before Environments existed (no sidecars, no extra
// env, no run network). Any OTHER error is propagated and fails the run.
func (a *AgentRunAction) resolveEnvironment(ctx context.Context, projectID uuid.UUID) (*model.Environment, error) {
	env, err := a.environmentRepo.GetByProjectID(ctx, projectID)
	if err != nil {
		var de *apperrors.DomainError
		if stderrors.As(err, &de) && de.Category == apperrors.CategoryNotFound {
			// No Environment for this project: legacy behaviour.
			return nil, nil
		}
		return nil, err
	}
	return env, nil
}

// buildConnStrings derives one or more "KEY=value" connection-string env entries
// per sidecar service in env, keyed by the service type detected from its image.
// The hostname is the service Name (its DNS alias on the per-run network), and
// credentials are read from the service's own Env using the conventional keys
// each image honours, falling back to sensible defaults when absent.
//
// Returns nil when env is nil or declares no services, so a project without an
// Environment injects no extra env and stays byte-for-byte identical to before.
// Unknown service types produce no auto connection string (documented limit):
// such services are still reachable by DNS on the run network, but the run is
// expected to configure them itself.
func buildConnStrings(env *model.Environment) []string {
	if env == nil || len(env.Services) == 0 {
		return nil
	}

	out := make([]string, 0, len(env.Services))
	for _, svc := range env.Services {
		out = append(out, connStringsForService(svc)...)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// connStringsForService returns the connection-string env entries for a single
// service, or nil when its type is unknown.
func connStringsForService(svc model.EnvironmentService) []string {
	host := svc.Name // DNS alias on the per-run network.
	switch model.DetectServiceType(svc.Image) {
	case model.ServiceTypePostgres:
		user := envOr(svc.Env, "POSTGRES_USER", "postgres")
		pass := envOr(svc.Env, "POSTGRES_PASSWORD", "postgres")
		db := envOr(svc.Env, "POSTGRES_DB", user)
		return []string{
			"DATABASE_URL=postgres://" + user + ":" + pass + "@" + host + ":5432/" + db,
		}
	case model.ServiceTypeMySQL, model.ServiceTypeMariaDB:
		user := envOr(svc.Env, "MYSQL_USER", "root")
		pass := envOr(svc.Env, "MYSQL_PASSWORD", envOr(svc.Env, "MYSQL_ROOT_PASSWORD", ""))
		db := envOr(svc.Env, "MYSQL_DATABASE", "")
		creds := user
		if pass != "" {
			creds = user + ":" + pass
		}
		return []string{
			"DATABASE_URL=mysql://" + creds + "@" + host + ":3306/" + db,
		}
	case model.ServiceTypeRedis:
		pass := envOr(svc.Env, "REDIS_PASSWORD", "")
		auth := ""
		if pass != "" {
			auth = ":" + pass + "@"
		}
		return []string{
			"REDIS_URL=redis://" + auth + host + ":6379/0",
		}
	case model.ServiceTypeMongo:
		user := envOr(svc.Env, "MONGO_INITDB_ROOT_USERNAME", "")
		pass := envOr(svc.Env, "MONGO_INITDB_ROOT_PASSWORD", "")
		creds := ""
		if user != "" {
			creds = user
			if pass != "" {
				creds = user + ":" + pass
			}
			creds += "@"
		}
		return []string{
			"MONGODB_URL=mongodb://" + creds + host + ":27017",
		}
	case model.ServiceTypeElasticsearch:
		return []string{
			"ELASTICSEARCH_URL=http://" + host + ":9200",
		}
	case model.ServiceTypeMailHog:
		return []string{
			"SMTP_HOST=" + host,
			"SMTP_PORT=1025",
		}
	default:
		// Unknown type: no auto connection string (still DNS-reachable).
		return nil
	}
}

// envOr returns env[key] when present and non-empty, otherwise def.
func envOr(env map[string]string, key, def string) string {
	if env != nil {
		if v, ok := env[key]; ok && v != "" {
			return v
		}
	}
	return def
}
