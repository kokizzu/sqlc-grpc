// Code generated by sqlc-grpc (https://github.com/walterwanderley/sqlc-grpc). DO NOT EDIT.

package main

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	pb_authors "booktest/api/authors/v1"
	app_authors "booktest/internal/authors"
	"booktest/internal/server"
)

func registerServer(db *pgxpool.Pool) server.RegisterServer {
	return func(grpcServer *grpc.Server) {
		pb_authors.RegisterAuthorsServiceServer(grpcServer, app_authors.NewService(app_authors.New(db), db))

	}
}

func registerHandlers() []server.RegisterHandlerFromEndpoint {
	var handlers []server.RegisterHandlerFromEndpoint

	handlers = append(handlers, pb_authors.RegisterAuthorsServiceHandlerFromEndpoint)

	return handlers
}
