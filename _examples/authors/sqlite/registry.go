// Code generated by sqlc-grpc (https://github.com/walterwanderley/sqlc-grpc). DO NOT EDIT.

package main

import (
	"database/sql"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	pb_authors "authors/api/authors/v1"
	app_authors "authors/internal/authors"
	"authors/internal/server"
)

func registerServer(logger *zap.Logger, db *sql.DB) server.RegisterServer {
	return func(grpcServer *grpc.Server) {
		pb_authors.RegisterAuthorsServiceServer(grpcServer, app_authors.NewService(logger, app_authors.New(db), db))

	}
}

func registerHandlers() []server.RegisterHandlerFromEndpoint {
	var handlers []server.RegisterHandlerFromEndpoint

	handlers = append(handlers, pb_authors.RegisterAuthorsServiceHandlerFromEndpoint)

	return handlers
}
