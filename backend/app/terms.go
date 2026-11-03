package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

// Change this only alongside the actual published and displayed terms. Safety
// consent during onboarding is separate and must not imply acceptance here.
const TermsVersion = "1.0"

type TermsStatus struct {
	Current         string     `json:"current_version"`
	Accepted        *string    `json:"accepted_version"`
	AcceptedAt      *time.Time `json:"accepted_at"`
	NeedsAcceptance bool       `json:"needs_acceptance"`
}
type TermsRepository interface {
	Terms(context.Context, string) (TermsStatus, error)
	AcceptTerms(context.Context, string, string) error
}

func (m mongoUsers) Terms(ctx context.Context, user string) (TermsStatus, error) {
	var record struct {
		Version    *string    `bson:"terms_version"`
		AcceptedAt *time.Time `bson:"terms_accepted_at"`
	}
	err := m.collection.FindOne(ctx, bson.M{"user_id": user}).Decode(&record)
	return TermsStatus{TermsVersion, record.Version, record.AcceptedAt, record.Version == nil || *record.Version != TermsVersion || record.AcceptedAt == nil}, err
}
func (m mongoUsers) AcceptTerms(ctx context.Context, user, version string) error {
	if version != TermsVersion {
		return ErrModerationConflict
	}
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		var profile User
		if err := m.collection.FindOne(tx, bson.M{"user_id": user}).Decode(&profile); err != nil {
			return nil, err
		}
		if accountRestricted(profile) {
			return nil, ErrModerationConflict
		}
		state, err := m.Terms(tx, user)
		if err != nil {
			return nil, err
		}
		// Preserve the original acceptance time and avoid duplicate audits on retries.
		if !state.NeedsAcceptance {
			return nil, nil
		}
		now := time.Now().UTC()
		if _, err = m.collection.UpdateOne(tx, bson.M{"user_id": user}, bson.M{"$set": bson.M{"terms_version": version, "terms_accepted_at": now, "updated_at": now}}); err != nil {
			return nil, err
		}
		_, err = WriteAudit(tx, m.collection.Database(), bson.M{"actor_id": user, "target_id": user, "action": "accepted_terms", "terms_version": version, "created_at": now})
		return nil, err
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
	return err
}
func (a *AuthDependencies) terms(w http.ResponseWriter, r *http.Request, user User) {
	repository, ok := a.Users.(TermsRepository)
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Terms storage unavailable"})
		return
	}
	if r.Method == http.MethodPost {
		var input struct {
			Version string `json:"terms_version"`
		}
		if decodeBody(r, &input) != nil || input.Version != TermsVersion {
			writeJSON(w, 422, map[string]string{"detail": "Accept the current terms version", "current_version": TermsVersion})
			return
		}
		if err := repository.AcceptTerms(r.Context(), user.ID, input.Version); err != nil {
			if errors.Is(err, ErrModerationConflict) {
				writeJSON(w, 409, map[string]string{"detail": "Terms acceptance is unavailable for this account"})
				return
			}
			writeJSON(w, 503, map[string]string{"detail": "Unable to record terms acceptance"})
			return
		}
		writeJSON(w, 200, map[string]string{"message": "Terms accepted", "terms_version": TermsVersion})
		return
	}
	state, err := repository.Terms(r.Context(), user.ID)
	if errors.Is(err, mongo.ErrNoDocuments) {
		writeJSON(w, 404, map[string]string{"detail": "Account not found"})
		return
	}
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Unable to load terms status"})
		return
	}
	writeJSON(w, 200, state)
}
