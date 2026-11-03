package server

import (
	"context"
	"errors"
	"net/http"
	"orix/backend/internal/matching"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

type GroupMemberRecord struct {
	UserID   string    `bson:"user_id"`
	Role     string    `bson:"role"`
	Ready    bool      `bson:"readiness_confirmed"`
	Here     bool      `bson:"is_here"`
	Finished bool      `bson:"finished"`
	Joined   time.Time `bson:"joined_at"`
}
type GroupRecord struct {
	Reminders     []MemberReminder    `bson:"reminder_schedule" json:"-"`
	SearchRegion  *SearchRegion       `bson:"route_search" json:"-"`
	SearchExpires time.Time           `bson:"search_expires_at"`
	Mode          string              `bson:"mode"`
	RoadPlan      *RoadPlan           `bson:"road_plan"`
	ID            string              `bson:"group_id"`
	Route         string              `bson:"route_summary"`
	Pickup        string              `bson:"pickup_summary"`
	Drop          string              `bson:"drop_summary"`
	Date          string              `bson:"date"`
	Window        RideWindow          `bson:"time_window"`
	FemaleOnly    bool                `bson:"female_only"`
	Status        string              `bson:"status"`
	Deadline      time.Time           `bson:"confirmation_deadline"`
	Created       time.Time           `bson:"created_at"`
	Finalized     *time.Time          `bson:"finalized_at"`
	Departure     time.Time           `bson:"departure_start"`
	End           time.Time           `bson:"departure_end"`
	Members       []GroupMemberRecord `bson:"members"`
}
type MemberView struct {
	Occupation string  `json:"occupation,omitempty"`
	Email      string  `json:"email,omitempty"`
	ID         string  `json:"user_id"`
	Name       string  `json:"display_name"`
	Photo      *string `json:"photo_url"`
	Phone      *string `json:"phone"`
	Gender     *string `json:"gender"`
	Role       string  `json:"role"`
	Ready      bool    `json:"readiness_confirmed"`
	IsMe       bool    `json:"is_me"`
	Here       bool    `json:"is_here"`
	Finished   bool    `json:"finished"`
}
type GroupView struct {
	DepartureEnd    time.Time    `json:"departure_end"`
	Mode            string       `json:"mode"`
	MyPickupTime    *time.Time   `json:"my_pickup_time,omitempty"`
	MyPickupSummary *string      `json:"my_pickup_summary,omitempty"`
	RoadPlan        *RoadPlan    `json:"road_plan,omitempty"`
	ID              string       `json:"group_id"`
	Route           string       `json:"route_summary"`
	Pickup          string       `json:"pickup_summary"`
	Drop            string       `json:"drop_summary"`
	Date            string       `json:"date"`
	Window          RideWindow   `json:"time_window"`
	FemaleOnly      bool         `json:"female_only"`
	Status          string       `json:"status"`
	Deadline        time.Time    `json:"confirmation_deadline"`
	Created         time.Time    `json:"created_at"`
	Finalized       *time.Time   `json:"finalized_at"`
	Departure       time.Time    `json:"departure_start"`
	MyReady         bool         `json:"my_readiness_confirmed"`
	AllReady        bool         `json:"all_confirmed"`
	Remaining       *int64       `json:"time_to_confirm_seconds"`
	Members         []MemberView `json:"members"`
}
type GroupRepository interface {
	MyGroups(context.Context, string, bool, string) ([]GroupView, error)
	ChangeGroup(context.Context, string, string, string) (GroupRecord, error)
}

func (m mongoUsers) MyGroups(ctx context.Context, user string, history bool, id string) ([]GroupView, error) {
	query := bson.M{"members.user_id": user}
	if id != "" {
		query["group_id"] = id
	} else if !history {
		query["members"] = bson.M{"$elemMatch": bson.M{"user_id": user, "finished": bson.M{"$ne": true}}}
		query["status"] = bson.M{"$in": []string{"pending", "confirming", "active", "in_progress"}}
	}
	if id == "" {
		pageQuery(ctx, query, "created_at", "group_id")
	}
	cursor, err := m.collection.Database().Collection("ride_groups").Find(ctx, query, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "group_id", Value: -1}}).SetLimit(pageLimit(ctx, 100)))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	records := []GroupRecord{}
	if err = cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	if id == "" {
		records = trimPage(ctx, records, func(group GroupRecord) (time.Time, string) { return group.Created, group.ID })
	}
	ids := []string{}
	for _, g := range records {
		for _, member := range g.Members {
			ids = append(ids, member.UserID)
		}
	}
	profiles := map[string]User{}
	if len(ids) > 0 {
		users, err := m.collection.Find(ctx, bson.M{"user_id": bson.M{"$in": ids}})
		if err != nil {
			return nil, err
		}
		defer users.Close(ctx)
		var data []User
		if err = users.All(ctx, &data); err != nil {
			return nil, err
		}
		for _, profile := range data {
			profiles[profile.ID] = profile
		}
	}
	blocked, err := m.BlockedPeers(ctx, user)
	if err != nil {
		return nil, err
	}
	views := make([]GroupView, 0, len(records))
	for _, g := range records {
		mode := g.Mode
		if mode == "" {
			mode = "auto"
		}
		view := GroupView{DepartureEnd: g.End, Mode: mode, ID: g.ID, Route: g.Route, Pickup: g.Pickup, Drop: g.Drop, Date: g.Date, Window: g.Window, FemaleOnly: g.FemaleOnly, Status: g.Status, Deadline: g.Deadline, Created: g.Created, Finalized: g.Finalized, Departure: g.Departure, AllReady: len(g.Members) > 1, Members: []MemberView{}}
		safePlan := g.Status != "cancelled" && g.Status != "expired"
		for _, member := range g.Members {
			if blocked[member.UserID] {
				safePlan = false
			}
		}
		if safePlan {
			view.RoadPlan = g.RoadPlan
			if planned, ok := memberPlannedPickup(g, user); ok && (g.Status == "active" || g.Status == "in_progress" || g.Status == "completed") {
				view.MyPickupTime = &planned.RideTime
				view.MyPickupSummary = &planned.Pickup
			}
		}
		if g.Status == "confirming" {
			remaining := max(0, int64(time.Until(g.Deadline).Seconds()))
			view.Remaining = &remaining
		}
		for _, member := range g.Members {
			profile, exists := profiles[member.UserID]
			name := profile.Name
			if !exists {
				name = "Former traveler"
			}
			var phone *string
			if member.UserID == user || (g.Status == "active" || g.Status == "in_progress") && member.Ready {
				if !blocked[member.UserID] {
					phone = profile.Phone
				}
			}
			email, photo := profile.Email, profile.Photo
			if blocked[member.UserID] {
				email = ""
				photo = nil
			}
			view.Members = append(view.Members, MemberView{Occupation: profile.Occupation, Email: email, ID: member.UserID, Name: name, Photo: photo, Phone: phone, Gender: profile.Gender, Role: member.Role, Ready: member.Ready, IsMe: member.UserID == user, Here: member.Here, Finished: member.Finished})
			view.AllReady = view.AllReady && member.Ready
			if member.UserID == user {
				view.MyReady = member.Ready
			}
		}
		views = append(views, view)
	}
	return views, nil
}

func (m mongoUsers) ChangeGroup(ctx context.Context, user, id, action string) (GroupRecord, error) {
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return GroupRecord{}, err
	}
	defer session.EndSession(ctx)
	result, err := session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		db := m.collection.Database()
		groups := db.Collection("ride_groups")
		now := time.Now().UTC()
		var profile User
		if err := m.collection.FindOne(tx, bson.M{"user_id": user}).Decode(&profile); err != nil {
			return nil, err
		}
		if accountRestricted(profile) || !profile.Onboarded {
			return nil, ErrGroupUnavailable
		}
		var group GroupRecord
		if err := groups.FindOne(tx, bson.M{"group_id": id, "members.user_id": user}).Decode(&group); err != nil {
			return nil, err
		}
		index := -1
		for i, member := range group.Members {
			if member.UserID == user {
				index = i
				break
			}
		}
		if index < 0 {
			return nil, ErrGroupUnavailable
		}
		changed := true
		extra := bson.M{}
		switch action {
		case "confirm":
			// A waiting singleton has no travel partner to confirm with. Marking it
			// ready would also remove it from automatic singleton reconciliation.
			if len(group.Members) < 2 || !group.End.After(now) {
				return nil, ErrGroupUnavailable
			}
			if group.Members[index].Ready && (group.Status == "confirming" || group.Status == "active") {
				changed = false
				break
			}
			if (group.Status != "pending" && group.Status != "confirming") || !group.Deadline.After(now) {
				return nil, ErrGroupUnavailable
			}
			group.Members[index].Ready = true
			all := len(group.Members) > 1
			for _, member := range group.Members {
				all = all && member.Ready
			}
			if all {
				if group.Mode == "lift" && (!roadPlanHasOffsets(group.RoadPlan) || group.RoadPlan.Created.Before(now.Add(-5*time.Minute))) {
					requests := []matching.Request{}
					for _, member := range group.Members {
						var memberRide Ride
						var memberProfile User
						if err := m.rides().FindOne(tx, bson.M{"group_id": id, "user_id": member.UserID, "status": "matched"}).Decode(&memberRide); err != nil {
							return nil, err
						}
						if err := m.collection.FindOne(tx, bson.M{"user_id": member.UserID}).Decode(&memberProfile); err != nil {
							return nil, err
						}
						route, err := routeRequest(memberRide, memberProfile)
						if err != nil {
							return nil, err
						}
						requests = append(requests, route)
					}
					plan, err := m.router.LiftPlan(tx, requests)
					if err != nil {
						return nil, err
					}
					group.RoadPlan = plan
					extra["road_plan"] = plan
				}
				if group.Mode == "lift" {
					if group.RoadPlan == nil || !group.RoadPlan.Latest.After(now) {
						return nil, ErrNoRoadPlan
					}
					group.Departure = group.RoadPlan.Earliest
					if group.Departure.Before(now) {
						group.Departure = now
					}
					group.End = group.RoadPlan.Latest
					extra["departure_start"] = group.Departure
					extra["departure_end"] = group.End
					extra["expires_at"] = group.End
				}
				planned, err := m.prepareMemberReminders(tx, group)
				if err != nil {
					return nil, err
				}
				group.Reminders = planned
				extra["reminder_schedule"] = planned
				group.Status = "active"
				group.Finalized = &now
			}
		case "mark-here":
			if group.Status != "active" && group.Status != "in_progress" {
				return nil, ErrGroupUnavailable
			}
			if group.Members[index].Here {
				changed = false
				break
			}
			group.Members[index].Here = true
		case "start":
			if group.Status == "in_progress" {
				changed = false
				break
			}
			if group.Status != "active" || group.Departure.IsZero() || now.Before(group.Departure) || !group.End.After(now) {
				return nil, ErrGroupUnavailable
			}
			if group.Mode == "lift" && group.Members[index].Role != "rider" {
				return nil, ErrGroupUnavailable
			}
			for _, member := range group.Members {
				// Lift passengers can be waiting at later stops. The ready driver starts
				// after arriving at their origin; other modes coordinate a common start.
				requireArrival := group.Mode != "lift" || member.Role == "rider"
				if !member.Ready || requireArrival && !member.Here {
					return nil, ErrGroupUnavailable
				}
			}
			// A materially late driver start shifts planned passenger pickups by the
			// same amount. Sub-minute start jitter keeps the confirmed schedule stable.
			if group.Mode == "lift" && now.Sub(group.Departure) >= time.Minute {
				group.Departure = now.UTC().Truncate(time.Millisecond)
				planned, err := m.prepareMemberReminders(tx, group)
				if err != nil {
					return nil, err
				}
				group.Reminders = planned
				extra["departure_start"] = group.Departure
				extra["reminder_schedule"] = planned
			}
			group.Status = "in_progress"
			extra["started_at"] = now
		case "finish":
			if group.Status == "completed" || group.Members[index].Finished {
				changed = false
				break
			}
			if group.Status != "in_progress" {
				return nil, ErrGroupUnavailable
			}
			group.Members[index].Finished = true
			all := true
			for _, member := range group.Members {
				all = all && member.Finished
			}
			if all {
				group.Status = "completed"
				extra["completed_at"] = now
			}
			if _, err := m.rides().UpdateMany(tx, bson.M{"user_id": user, "group_id": id, "status": "matched"}, bson.M{"$set": bson.M{"status": "completed", "completed_at": now}}); err != nil {
				return nil, err
			}
		case "leave":
			if group.Status != "pending" && group.Status != "confirming" && group.Status != "active" {
				return nil, ErrGroupUnavailable
			}
			driverLeft := group.Members[index].Role == "rider"
			group.Members = append(group.Members[:index], group.Members[index+1:]...)
			group.Finalized = nil
			group.RoadPlan = nil
			extra["road_plan"] = nil
			extra["route_search"] = nil
			group.Reminders = nil
			extra["reminder_schedule"] = nil
			group.Status = "confirming"
			if len(group.Members) == 1 {
				group.Status = "pending"
			}
			if len(group.Members) == 0 || driverLeft {
				group.Status = "cancelled"
			}
			for i := range group.Members {
				group.Members[i].Ready = false
				group.Members[i].Here = false
			}
			if len(group.Members) == 1 && group.Status != "cancelled" {
				group.SearchExpires = minTime(now.Add(15*time.Hour), group.End)
				extra["search_expires_at"] = group.SearchExpires
				if _, err := m.rides().UpdateMany(tx, bson.M{"group_id": id, "user_id": group.Members[0].UserID, "status": "matched"}, bson.M{"$set": bson.M{"expires_at": group.SearchExpires}}); err != nil {
					return nil, err
				}
			}
			group.Deadline = minTime(now.Add(time.Hour), group.End)
			filter := bson.M{"group_id": id, "status": "matched"}
			if group.Status != "cancelled" {
				filter["user_id"] = user
			}
			if _, err := m.rides().UpdateMany(tx, filter, bson.M{"$set": bson.M{"status": "cancelled", "cancelled_at": now}}); err != nil {
				return nil, err
			}
		default:
			return nil, errors.New("unknown group action")
		}
		if !changed {
			return group, nil
		}
		extra["members"] = group.Members
		extra["status"] = group.Status
		extra["finalized_at"] = group.Finalized
		extra["confirmation_deadline"] = group.Deadline
		extra["updated_at"] = now
		if _, err := groups.UpdateOne(tx, bson.M{"group_id": id, "members.user_id": user}, bson.M{"$set": extra}); err != nil {
			return nil, err
		}
		_, err := db.Collection("notification_outbox").InsertOne(tx, bson.M{"type": "group_" + action, "group_id": id, "user_id": user, "status": "pending", "attempts": 0, "created_at": now})
		if err != nil {
			return nil, err
		}
		return group, nil
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
	if err != nil {
		return GroupRecord{}, err
	}
	group, ok := result.(GroupRecord)
	if !ok {
		return GroupRecord{}, errors.New("missing group result")
	}
	return group, nil
}
func (a *AuthDependencies) groupStorage(w http.ResponseWriter, user User) (GroupRepository, bool) {
	if _, ok := a.rideStorage(w, user); !ok {
		return nil, false
	}
	repository, ok := a.Users.(GroupRepository)
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Group storage is not configured"})
	}
	return repository, ok
}
func (a *AuthDependencies) getGroups(w http.ResponseWriter, r *http.Request, user User) {
	repository, ok := a.groupStorage(w, user)
	if !ok {
		return
	}
	if r.PathValue("group_id") == "" {
		paged, valid := preparePage(w, r, "groups:"+user.ID, 100)
		if !valid {
			return
		}
		r = paged
	}
	groups, err := repository.MyGroups(r.Context(), user.ID, r.URL.Query().Get("include_completed") == "true", r.PathValue("group_id"))
	if err != nil {
		writeJSON(w, 503, map[string]string{"detail": "Unable to load groups"})
		return
	}
	if r.PathValue("group_id") != "" {
		if len(groups) == 0 {
			writeJSON(w, 404, map[string]string{"detail": "Group not found"})
			return
		}
		writeJSON(w, 200, groups[0])
		return
	}
	writePageHeader(w, r)
	writeJSON(w, 200, groups)
}
func (a *AuthDependencies) groupAction(action string) func(http.ResponseWriter, *http.Request, User) {
	return func(w http.ResponseWriter, r *http.Request, user User) {
		repository, ok := a.groupStorage(w, user)
		if !ok {
			return
		}
		if action == "confirm" {
			var input struct {
				Confirmed *bool `json:"confirmed"`
			}
			if decodeBody(r, &input) != nil || input.Confirmed != nil && !*input.Confirmed {
				writeJSON(w, 422, map[string]string{"detail": "Readiness must be confirmed"})
				return
			}
		}
		group, err := repository.ChangeGroup(r.Context(), user.ID, r.PathValue("group_id"), action)
		if errors.Is(err, mongo.ErrNoDocuments) {
			writeJSON(w, 404, map[string]string{"detail": "Group not found"})
			return
		}
		if errors.Is(err, ErrGroupUnavailable) {
			writeJSON(w, 409, map[string]string{"detail": "This group action is no longer available"})
			return
		}
		if err != nil {
			writeJSON(w, 503, map[string]string{"detail": "Unable to update group"})
			return
		}
		all := len(group.Members) > 1
		for _, member := range group.Members {
			all = all && member.Ready
		}
		writeJSON(w, 200, map[string]any{"message": "Group updated", "group_id": group.ID, "status": group.Status, "group_status": group.Status, "all_confirmed": all})
	}
}

// Moderator cancellation is separate from member departure: it ends the group
// for everyone and must release every member's active request atomically.
type GroupModerationRepository interface {
	CancelGroupAsAdmin(context.Context, string, string, string) error
}

func (m mongoUsers) CancelGroupAsAdmin(ctx context.Context, actor, id, note string) error {
	session, err := m.collection.Database().Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		if _, err := m.authorizeAdminMutation(tx, actor); err != nil {
			return nil, err
		}
		db := m.collection.Database()
		groups := db.Collection("ride_groups")
		var group GroupRecord
		if err := groups.FindOne(tx, bson.M{"group_id": id}).Decode(&group); err != nil {
			return nil, err
		}
		if group.Status == "cancelled" {
			return nil, nil
		}
		switch group.Status {
		case "pending", "confirming", "active", "in_progress":
		default:
			return nil, ErrGroupUnavailable
		}
		now := time.Now().UTC()
		if _, err := groups.UpdateOne(tx, bson.M{"group_id": id}, bson.M{
			"$set":   bson.M{"status": "cancelled", "updated_at": now},
			"$unset": bson.M{"match_lease": "", "match_lease_until": ""},
		}); err != nil {
			return nil, err
		}
		if _, err := m.rides().UpdateMany(tx, bson.M{"group_id": id, "status": "matched"}, bson.M{"$set": bson.M{"status": "cancelled", "updated_at": now}}); err != nil {
			return nil, err
		}
		if _, err := db.Collection("notification_outbox").InsertOne(tx, bson.M{"type": "group_cancelled", "group_id": id, "status": "pending", "attempts": 0, "created_at": now}); err != nil {
			return nil, err
		}
		_, err := WriteAudit(tx, db, bson.M{"actor_id": actor, "target_id": id, "action": "cancel_group", "note": note, "created_at": now})
		return nil, err
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
	return err
}

func (a *AuthDependencies) cancelGroupAsAdmin(w http.ResponseWriter, r *http.Request, user User) {
	var input struct {
		Note string `json:"note"`
	}
	if decodeBody(r, &input) != nil {
		writeJSON(w, 422, map[string]string{"detail": "Provide a cancellation note"})
		return
	}
	input.Note = strings.TrimSpace(input.Note)
	id := r.PathValue("group_id")
	if id == "" || len(id) > 100 || utf8.RuneCountInString(input.Note) < 3 || utf8.RuneCountInString(input.Note) > 1000 {
		writeJSON(w, 422, map[string]string{"detail": "Provide a valid group and a 3–1000 character note"})
		return
	}
	repository, ok := a.Users.(GroupModerationRepository)
	if !ok {
		writeJSON(w, 503, map[string]string{"detail": "Group moderation storage unavailable"})
		return
	}
	err := repository.CancelGroupAsAdmin(r.Context(), user.ID, id, input.Note)
	if adminAccessDenied(w, err) {
		return
	}
	switch {
	case errors.Is(err, mongo.ErrNoDocuments):
		writeJSON(w, 404, map[string]string{"detail": "Group or administrator not found"})
	case errors.Is(err, ErrModerationConflict):
		writeJSON(w, 403, map[string]string{"detail": "Administrator access required"})
	case errors.Is(err, ErrGroupUnavailable):
		writeJSON(w, 409, map[string]string{"detail": "This group is already closed"})
	case err != nil:
		writeJSON(w, 503, map[string]string{"detail": "Unable to cancel group"})
	default:
		writeJSON(w, 200, map[string]string{"group_id": id, "status": "cancelled"})
	}
}
