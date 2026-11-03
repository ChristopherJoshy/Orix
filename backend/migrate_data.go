package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	server "orix/backend/app"
)

type migrationIssue struct {
	Collection string `json:"collection"`
	ID         string `json:"id"`
	Reason     string `json:"reason"`
}
type migrationChange struct {
	collection string
	id         any
	fields     bson.M
	original   bson.M
}
type migrationPlan struct {
	Version    int              `json:"version"`
	Records    int              `json:"records"`
	Updates    int              `json:"updates"`
	IssueCount int              `json:"issue_count"`
	Issues     []migrationIssue `json:"issues"`
	Applied    bool             `json:"applied"`
	changes    []migrationChange
}

func (p *migrationPlan) issue(collection, id, reason string) {
	p.IssueCount++
	if len(p.Issues) < 100 {
		p.Issues = append(p.Issues, migrationIssue{collection, id, reason})
	}
}
func localBounds(date string, window server.RideWindow, offset *int) (time.Time, time.Time, error) {
	if offset == nil || *offset < -720 || *offset > 840 {
		return time.Time{}, time.Time{}, errors.New("timezone offset is missing or invalid")
	}
	zone := time.FixedZone("ride", *offset*60)
	start, err := time.ParseInLocation("2006-01-02 15:04", date+" "+window.Start, zone)
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("invalid local start")
	}
	end, err := time.ParseInLocation("2006-01-02 15:04", date+" "+window.End, zone)
	if err != nil || !end.After(start) {
		return time.Time{}, time.Time{}, errors.New("invalid local end")
	}
	return start.UTC(), end.UTC(), nil
}
func migrationRows(ctx context.Context, db *mongo.Database, name string) ([]bson.M, error) {
	cursor, err := db.Collection(name).Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	rows := []bson.M{}
	for cursor.Next(ctx) {
		if len(rows) >= 50000 {
			return nil, fmt.Errorf("%s exceeds the single-run migration bound; use a reviewed batched migration", name)
		}
		var row bson.M
		if err = cursor.Decode(&row); err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, cursor.Err()
}
func decodeMigration(row bson.M, value any) error {
	raw, err := bson.Marshal(row)
	if err != nil {
		return err
	}
	return bson.Unmarshal(raw, value)
}
func planLegacyData(ctx context.Context, db *mongo.Database) (migrationPlan, error) {
	plan := migrationPlan{Version: 4, Issues: []migrationIssue{}}
	now := time.Now().UTC()
	groupRows, err := migrationRows(ctx, db, "ride_groups")
	if err != nil {
		return plan, err
	}
	groups := map[string]server.GroupRecord{}
	for _, row := range groupRows {
		var group server.GroupRecord
		if decodeMigration(row, &group) != nil || group.ID == "" {
			plan.issue("ride_groups", "unknown", "invalid group schema")
			continue
		}
		if _, exists := groups[group.ID]; exists {
			plan.issue("ride_groups", group.ID, "duplicate group ID")
		}
		groups[group.ID] = group
	}
	rideRows, err := migrationRows(ctx, db, "ride_requests")
	if err != nil {
		return plan, err
	}
	byGroup := map[string]map[string][]server.Ride{}
	active := map[string]string{}
	requestIDs := map[string]bool{}
	for _, row := range rideRows {
		plan.Records++
		var ride server.Ride
		var metadata struct {
			Offset *int `bson:"timezone_offset_minutes"`
		}
		if decodeMigration(row, &ride) != nil || decodeMigration(row, &metadata) != nil || ride.ID == "" || ride.UserID == "" {
			plan.issue("ride_requests", "unknown", "invalid request schema")
			continue
		}
		if requestIDs[ride.ID] {
			plan.issue("ride_requests", ride.ID, "duplicate request ID")
		}
		requestIDs[ride.ID] = true
		start, end, boundsErr := localBounds(ride.Date, ride.Window, metadata.Offset)
		if boundsErr != nil {
			plan.issue("ride_requests", ride.ID, boundsErr.Error())
			continue
		}
		fields := bson.M{}
		if ride.StartUTC.IsZero() {
			fields["departure_start"] = start
			ride.StartUTC = start
		} else if !ride.StartUTC.Equal(start) {
			plan.issue("ride_requests", ride.ID, "stored UTC start disagrees with explicit local window")
		}
		if ride.EndUTC.IsZero() {
			fields["departure_end"] = end
			ride.EndUTC = end
		} else if !ride.EndUTC.Equal(end) {
			plan.issue("ride_requests", ride.ID, "stored UTC end disagrees with explicit local window")
		}
		if ride.Mode == "" {
			if ride.Driver {
				plan.issue("ride_requests", ride.ID, "driver request needs explicit mode/capacity review")
				continue
			}
			ride.Mode = "auto"
			fields["mode"] = ride.Mode
		}
		if ride.Mode != "auto" && ride.Mode != "partner" && ride.Mode != "lift" {
			plan.issue("ride_requests", ride.ID, "unknown travel mode")
		}
		if ride.Expires.IsZero() {
			if ride.Created.IsZero() {
				plan.issue("ride_requests", ride.ID, "missing creation/expiry time")
				continue
			}
			ride.Expires = ride.Created.Add(15 * time.Hour)
			if end.Before(ride.Expires) {
				ride.Expires = end
			}
			fields["expires_at"] = ride.Expires
		}
		if (ride.Status == "pending" || ride.Status == "matching") && !ride.Expires.After(now) {
			ride.Status = "expired"
			fields["status"] = "expired"
		}
		if ride.Status == "matched" {
			if ride.GroupID == nil {
				plan.issue("ride_requests", ride.ID, "matched request has no group")
			} else if group, exists := groups[*ride.GroupID]; !exists {
				plan.issue("ride_requests", ride.ID, "matched group is missing")
			} else if group.Status == "completed" || group.Status == "cancelled" || group.Status == "expired" {
				ride.Status = group.Status
				fields["status"] = group.Status
			}
		}
		if ride.Status == "pending" || ride.Status == "matching" || ride.Status == "matched" {
			if previous, exists := active[ride.UserID]; exists {
				plan.issue("ride_requests", ride.ID, "multiple active requests; resolve explicitly with request "+previous)
			} else {
				active[ride.UserID] = ride.ID
			}
		}
		if ride.GroupID != nil {
			if byGroup[*ride.GroupID] == nil {
				byGroup[*ride.GroupID] = map[string][]server.Ride{}
			}
			byGroup[*ride.GroupID][ride.UserID] = append(byGroup[*ride.GroupID][ride.UserID], ride)
		}
		if len(fields) > 0 {
			plan.changes = append(plan.changes, migrationChange{"ride_requests", row["_id"], fields, row})
		}
	}
	for _, row := range groupRows {
		plan.Records++
		var group server.GroupRecord
		var metadata struct {
			Offset *int `bson:"timezone_offset_minutes"`
		}
		if decodeMigration(row, &group) != nil || decodeMigration(row, &metadata) != nil || group.ID == "" {
			continue
		}
		live := group.Status == "pending" || group.Status == "confirming" || group.Status == "active" || group.Status == "in_progress"
		fields := bson.M{}
		if group.Status == "pending" && len(group.Members) == 1 && group.Members[0].Ready {
			group.Members[0].Ready = false
			group.Members[0].Here = false
			group.Members[0].Finished = false
			fields["members"] = group.Members
		}
		start, end := group.Departure, group.End
		mode := group.Mode
		searchEnd := time.Time{}
		seen := map[string]bool{}
		for _, member := range group.Members {
			if seen[member.UserID] {
				plan.issue("ride_groups", group.ID, "duplicate member")
			}
			seen[member.UserID] = true
			requests := byGroup[group.ID][member.UserID]
			if len(requests) != 1 {
				if live {
					plan.issue("ride_groups", group.ID, "live member has missing or ambiguous route")
				}
				continue
			}
			ride := requests[0]
			if mode == "" {
				mode = string(ride.Mode)
			} else if mode != string(ride.Mode) {
				plan.issue("ride_groups", group.ID, "mixed travel modes")
			}
			if group.RoadPlan == nil {
				if start.IsZero() || ride.StartUTC.After(start) {
					start = ride.StartUTC
				}
				if end.IsZero() || ride.EndUTC.Before(end) {
					end = ride.EndUTC
				}
			}
			if searchEnd.IsZero() || ride.Expires.Before(searchEnd) {
				searchEnd = ride.Expires
			}
		}
		if group.RoadPlan != nil {
			start, end = group.RoadPlan.Earliest, group.RoadPlan.Latest
		}
		if start.IsZero() || end.IsZero() {
			if localStart, localEnd, err := localBounds(group.Date, group.Window, metadata.Offset); err == nil {
				start, end = localStart, localEnd
			}
		}
		if live && (start.IsZero() || !end.After(start) || len(group.Members) == 0 || len(group.Members) > 4) {
			plan.issue("ride_groups", group.ID, "live group has invalid timing or member count")
			continue
		}
		if live && mode == "lift" && group.RoadPlan == nil && group.Status != "pending" {
			plan.issue("ride_groups", group.ID, "live lift needs a verified road plan")
			continue
		}
		if group.Departure.IsZero() && !start.IsZero() {
			fields["departure_start"] = start
		}
		if group.End.IsZero() && !end.IsZero() {
			fields["departure_end"] = end
		}
		if group.Mode == "" && mode != "" {
			fields["mode"] = mode
		}
		if group.SearchExpires.IsZero() && !searchEnd.IsZero() {
			fields["search_expires_at"] = searchEnd
		}
		if group.Deadline.IsZero() && !group.Created.IsZero() && !end.IsZero() {
			deadline := group.Created.Add(time.Hour)
			if end.Before(deadline) {
				deadline = end
			}
			fields["confirmation_deadline"] = deadline
		}
		if _, exists := row["origin_user_id"]; !exists && len(group.Members) > 0 {
			fields["origin_user_id"] = group.Members[0].UserID
		}
		if group.Status == "active" || group.Status == "in_progress" {
			normalized := group
			normalized.Departure = start
			normalized.Mode = mode
			labels := map[string]string{}
			for _, member := range group.Members {
				requests := byGroup[group.ID][member.UserID]
				if len(requests) == 1 {
					labels[member.UserID] = requests[0].PickupLabel
				}
			}
			planned, err := server.BuildMemberReminders(normalized, labels)
			if err != nil {
				plan.issue("ride_groups", group.ID, "live group needs verified per-member pickup timing before cutover")
			} else if !samePlannedReminders(group.Reminders, planned) {
				fields["reminder_schedule"] = planned
			}
		}
		if (group.Status == "pending" || group.Status == "confirming") && len(group.Members) > 0 {
			memberRoutes := byGroup[group.ID][group.Members[0].UserID]
			if len(memberRoutes) == 1 {
				region := server.SearchRegionForRide(memberRoutes[0])
				if region == nil {
					plan.issue("ride_groups", group.ID, "waiting group route coordinates need review")
				} else if group.SearchRegion == nil || *group.SearchRegion != *region {
					fields["route_search"] = region
				}
			}
		}
		if len(fields) > 0 {
			plan.changes = append(plan.changes, migrationChange{"ride_groups", row["_id"], fields, row})
		}
	}
	// Python audits used timestamp; Go uses created_at for indexed paging.
	// Preserve the original field and backfill only a valid BSON date, never now.
	auditRows, err := migrationRows(ctx, db, "admin_logs")
	if err != nil {
		return plan, err
	}
	seenAuditIDs := map[string]bool{}
	for _, row := range auditRows {
		plan.Records++
		id, ok := row["_id"].(bson.ObjectID)
		if !ok {
			plan.issue("admin_logs", "unknown", "audit ID requires explicit ObjectID migration")
			continue
		}
		var record struct {
			Created *time.Time `bson:"created_at"`
			Legacy  *time.Time `bson:"timestamp"`
		}
		if err := decodeMigration(row, &record); err != nil {
			plan.issue("admin_logs", id.Hex(), "invalid audit timestamp schema")
			continue
		}
		fields := bson.M{}
		logID, validID := row["log_id"].(string)
		if !validID || logID == "" {
			if raw, exists := row["log_id"]; exists && raw != nil && !validID {
				plan.issue("admin_logs", id.Hex(), "invalid audit log ID type")
				continue
			}
			logID = id.Hex()
			fields["log_id"] = logID
		}
		if seenAuditIDs[logID] {
			plan.issue("admin_logs", id.Hex(), "duplicate planned audit log ID")
		}
		seenAuditIDs[logID] = true
		if record.Created == nil || record.Created.IsZero() {
			if record.Legacy == nil || record.Legacy.IsZero() {
				plan.issue("admin_logs", id.Hex(), "missing audit timestamp; do not infer an acceptance or action time")
				continue
			}
			fields["created_at"] = record.Legacy.UTC()
		}
		if len(fields) > 0 {
			plan.changes = append(plan.changes, migrationChange{"admin_logs", id, fields, row})
		}
	}

	if err := migrationUniqueConflicts(ctx, db, &plan); err != nil {
		return plan, err
	}
	plan.Updates = len(plan.changes)
	return plan, nil
}
func migrateData() error {
	database := os.Getenv("MONGODB_DATABASE")
	if database == "" || os.Getenv("MONGODB_URI") == "" {
		return errors.New("explicit MongoDB URI/database required")
	}
	apply := os.Getenv("MIGRATION_APPLY") == "true"
	if apply && (os.Getenv("MIGRATION_DATABASE") != database || os.Getenv("MIGRATION_WRITERS_STOPPED") != "true") {
		return errors.New("apply requires matching MIGRATION_DATABASE and explicit MIGRATION_WRITERS_STOPPED=true")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	client, err := mongo.Connect(options.Client().ApplyURI(os.Getenv("MONGODB_URI")))
	if err != nil {
		return err
	}
	defer client.Disconnect(context.Background())
	plan, err := planLegacyData(ctx, client.Database(database))
	if err != nil {
		return err
	}
	if apply && plan.IssueCount == 0 {
		for _, change := range plan.changes {
			filter := bson.M{"_id": change.id}
			if updated, exists := change.original["updated_at"]; exists {
				filter["updated_at"] = updated
			} else {
				filter["updated_at"] = bson.M{"$exists": false}
			}
			for _, key := range []string{"timestamp", "date", "time_window", "timezone_offset_minutes", "mode", "status", "members", "group_id", "created_at", "expires_at", "road_plan"} {
				if original, exists := change.original[key]; exists {
					filter[key] = original
				} else {
					filter[key] = bson.M{"$exists": false}
				}
			}
			// Compare every changed field too, so newer data is never overwritten blindly.
			for key := range change.fields {
				if original, exists := change.original[key]; exists {
					filter[key] = original
				} else {
					filter[key] = bson.M{"$exists": false}
				}
			}
			result, writeErr := client.Database(database).Collection(change.collection).UpdateOne(ctx, filter, bson.M{"$set": change.fields})
			if writeErr != nil {
				return writeErr
			}
			if result.MatchedCount != 1 {
				return errors.New("migration encountered concurrent changes; stop writers and re-plan")
			}
		}
		plan.Applied = true
	}
	if err = json.NewEncoder(os.Stdout).Encode(plan); err != nil {
		return err
	}
	if plan.IssueCount > 0 {
		return errors.New("migration has conflicts; no changes applied")
	}
	return nil
}

func migrationUniqueConflicts(ctx context.Context, db *mongo.Database, plan *migrationPlan) error {
	checks := []struct {
		collection string
		fields     []string
	}{
		{"users", []string{"user_id"}}, {"users", []string{"firebase_uid"}},
		{"user_avatars", []string{"user_id"}}, {"ratings", []string{"group_id", "rater_user_id", "rated_user_id"}},
		{"chat_messages", []string{"message_id"}}, {"reports", []string{"report_id"}},
		{"support_tickets", []string{"ticket_id"}},
	}
	for _, check := range checks {
		key := bson.M{}
		for _, field := range check.fields {
			key[field] = "$" + field
		}
		cursor, err := db.Collection(check.collection).Aggregate(ctx, bson.A{bson.M{"$group": bson.M{"_id": key, "count": bson.M{"$sum": 1}}}, bson.M{"$match": bson.M{"count": bson.M{"$gt": 1}}}})
		if err != nil {
			return err
		}
		for cursor.Next(ctx) {
			plan.issue(check.collection, "duplicate-key", "duplicate records conflict with required uniqueness; resolve explicitly")
		}
		err = cursor.Err()
		cursor.Close(ctx)
		if err != nil {
			return err
		}
	}
	for _, collection := range []string{"ride_requests", "ride_groups"} {
		cursor, err := db.Collection(collection).Indexes().List(ctx)
		if err != nil {
			return err
		}
		for cursor.Next(ctx) {
			var index bson.M
			if err = cursor.Decode(&index); err != nil {
				cursor.Close(ctx)
				return err
			}
			if _, exists := index["expireAfterSeconds"]; exists {
				plan.issue(collection, fmt.Sprint(index["name"]), "legacy TTL deletion conflicts with explicit lifecycle/history; review and remove deliberately")
			}
		}
		err = cursor.Err()
		cursor.Close(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func samePlannedReminders(a, b []server.MemberReminder) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].UserID != b[i].UserID || a[i].Pickup != b[i].Pickup || !a[i].RideTime.Equal(b[i].RideTime) {
			return false
		}
	}
	return true
}
