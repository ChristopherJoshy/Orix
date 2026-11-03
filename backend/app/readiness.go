package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

type ReadinessRepository interface{ Ready(context.Context) error }

// Transactions and outbox change streams require a replica set or mongos, not a
// standalone MongoDB. Reject unsupported topology before opening API traffic.
func requireTransactionalMongo(ctx context.Context, client *mongo.Client) error {
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return err
	}
	var topology struct {
		SetName  string `bson:"setName"`
		Message  string `bson:"msg"`
		Writable bool   `bson:"isWritablePrimary"`
		Sessions *int64 `bson:"logicalSessionTimeoutMinutes"`
	}
	if err := client.Database("admin").RunCommand(ctx, bson.D{{Key: "hello", Value: 1}}).Decode(&topology); err != nil {
		return err
	}
	if topology.Sessions == nil || (topology.SetName == "" && topology.Message != "isdbgrid") || !topology.Writable {
		return errors.New("MongoDB requires a writable replica set or sharded cluster with session support")
	}
	return nil
}
func (m mongoUsers) Ready(ctx context.Context) error {
	return requireTransactionalMongo(ctx, m.collection.Database().Client())
}
func (a *AuthDependencies) readiness(w http.ResponseWriter, r *http.Request) {
	if a == nil || a.Verifier == nil || a.Users == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		return
	}
	repository, ok := a.Users.(ReadinessRepository)
	if !ok {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if err := repository.Ready(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
