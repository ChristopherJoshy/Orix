package server

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type KnownPartner struct {
	ID    string  `json:"user_id" bson:"user_id"`
	Name  string  `json:"display_name" bson:"display_name"`
	Photo *string `json:"photo_url" bson:"photo_url"`
	Email string  `json:"email" bson:"email"`
}
type PartnerDirectory interface {
	SearchPartners(context.Context, string, string) ([]KnownPartner, error)
}

func (m mongoUsers) SearchPartners(ctx context.Context, user, query string) ([]KnownPartner, error) {
	// Only people already encountered in a ride are report/search candidates.
	pipeline := bson.A{
		bson.M{"$match": bson.M{"members.user_id": user}},
		bson.M{"$unwind": "$members"},
		bson.M{"$match": bson.M{"members.user_id": bson.M{"$ne": user}}},
		bson.M{"$group": bson.M{"_id": "$members.user_id"}},
		bson.M{"$lookup": bson.M{"from": "users", "localField": "_id", "foreignField": "user_id", "as": "profile"}},
		bson.M{"$unwind": "$profile"},
		bson.M{"$match": bson.M{"profile.status": bson.M{"$nin": []string{"deleting", "deleted"}}}},
		bson.M{"$sort": bson.D{{Key: "profile.email", Value: 1}, {Key: "_id", Value: 1}}},
		bson.M{"$limit": 200},
		bson.M{"$project": bson.M{"_id": 0, "user_id": "$profile.user_id", "display_name": "$profile.display_name", "photo_url": "$profile.photo_url", "email": "$profile.email"}},
	}
	cursor, err := m.collection.Database().Collection("ride_groups").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	partners := []KnownPartner{}
	if err = cursor.All(ctx, &partners); err != nil {
		return nil, err
	}
	blocked, err := m.BlockedPeers(ctx, user)
	if err != nil {
		return nil, err
	}
	for i := range partners {
		if blocked[partners[i].ID] {
			partners[i].Photo = nil
		}
	}
	return rankPartnerEmails(partners, query), nil
}
func (a *AuthDependencies) searchPartners(w http.ResponseWriter, r *http.Request, user User) {
	if !user.Onboarded {
		writeJSON(w, 403, map[string]string{"detail": "Complete onboarding first"})
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if utf8.RuneCountInString(query) < 2 || utf8.RuneCountInString(query) > 100 {
		writeJSON(w, 422, map[string]string{"detail": "Search by 2–100 characters of a travel partner’s email"})
		return
	}
	repository, ok := a.Users.(PartnerDirectory)
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Partner search unavailable"})
		return
	}
	partners, err := repository.SearchPartners(r.Context(), user.ID, query)
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Unable to search ride partners"})
		return
	}
	writeJSON(w, 200, partners)
}

// Bounded fuzzy lookup over already encountered partners. It does not expose a
// public account directory. Exact, substring and small typo matches rank in order.
func rankPartnerEmails(partners []KnownPartner, query string) []KnownPartner {
	query = strings.ToLower(strings.TrimSpace(query))
	type ranked struct {
		person KnownPartner
		score  int
	}
	matches := []ranked{}
	for _, person := range partners {
		email := strings.ToLower(person.Email)
		score := 1000
		if email == query {
			score = 0
		} else if strings.Contains(email, query) {
			score = 1
		} else {
			local := strings.SplitN(email, "@", 2)[0]
			target := email
			if !strings.Contains(query, "@") {
				target = local
			}
			distance := emailDistance(query, target)
			if len(query) >= 4 && distance <= 2 {
				score = distance + 2
			}
		}
		if score < 1000 {
			matches = append(matches, ranked{person, score})
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score < matches[j].score
		}
		return matches[i].person.Email < matches[j].person.Email
	})
	result := []KnownPartner{}
	for i, item := range matches {
		if i == 20 {
			break
		}
		result = append(result, item.person)
	}
	return result
}
func emailDistance(a, b string) int {
	x, y := []rune(a), []rune(b)
	previous := make([]int, len(y)+1)
	for j := range previous {
		previous[j] = j
	}
	for i, u := range x {
		current := make([]int, len(y)+1)
		current[0] = i + 1
		for j, v := range y {
			cost := 0
			if u != v {
				cost = 1
			}
			current[j+1] = min(current[j]+1, previous[j+1]+1, previous[j]+cost)
		}
		previous = current
	}
	return previous[len(y)]
}
