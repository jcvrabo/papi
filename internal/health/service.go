package health

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Status struct {
	Status       string
	Environments []EnvironmentHealth
}

type EnvironmentHealth struct {
	Name   string
	Status string
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) GetHealth(ctx context.Context) (*Status, error) {
	if err := s.pool.Ping(ctx); err != nil {
		return &Status{Status: "unhealthy"}, nil
	}

	rows, err := s.pool.Query(ctx,
		"SELECT re.name, re.health_status FROM runtime_environments re")
	if err != nil {
		return &Status{Status: "degraded"}, nil
	}
	defer rows.Close()

	var environments []EnvironmentHealth
	overallHealthy := true
	for rows.Next() {
		var env EnvironmentHealth
		if err := rows.Scan(&env.Name, &env.Status); err != nil {
			continue
		}
		if env.Status != "healthy" {
			overallHealthy = false
		}
		environments = append(environments, env)
	}

	status := "healthy"
	if !overallHealthy {
		status = "degraded"
	}

	return &Status{
		Status:       status,
		Environments: environments,
	}, nil
}
