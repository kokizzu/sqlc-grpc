// Code generated by sqlc-grpc (https://github.com/walterwanderley/sqlc-grpc).

package authors

import (
	"github.com/jackc/pgx/v5/pgxpool"

	pb "booktest/api/authors/v1"
)

// NewService is a constructor of a pb.AuthorsServiceServer implementation.
// Use this function to customize the server by adding middlewares to it.
func NewService(querier *Queries, db *pgxpool.Pool) pb.AuthorsServiceServer {
	return &Service{querier: querier}
}
