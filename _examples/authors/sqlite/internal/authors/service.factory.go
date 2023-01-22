// Code generated by sqlc-grpc (https://github.com/walterwanderley/sqlc-grpc).

package authors

import (
	"database/sql"

	"go.uber.org/zap"

	pb "authors/api/authors/v1"
)

// NewService is a constructor of a pb.AuthorsServiceServer implementation.
// Use this function to customize the server by adding middlewares to it.
func NewService(logger *zap.Logger, querier *Queries, db *sql.DB) pb.AuthorsServiceServer {
	return &Service{logger: logger, querier: querier}
}
