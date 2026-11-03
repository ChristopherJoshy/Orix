package server

import (
	"go.mongodb.org/mongo-driver/v2/bson"
	"orix/backend/internal/matching"
)

// Increment when changing the bound construction or matching corridor policy.
// Old versions remain eligible for full evaluation until explicit backfill.
const searchRegionVersion = 1

type SearchRegion struct {
	Version    int     `bson:"version" json:"-"`
	AnchorUser string  `bson:"anchor_user_id" json:"-"`
	MinLat     float64 `bson:"min_lat" json:"-"`
	MaxLat     float64 `bson:"max_lat" json:"-"`
	MinLng     float64 `bson:"min_lng" json:"-"`
	MaxLng     float64 `bson:"max_lng" json:"-"`
}

// SearchRegionForRide is shared by normal writes and the explicit migration
// planner. A region must belong to a current member; leaving clears it.
func SearchRegionForRide(ride Ride) *SearchRegion {
	if len(ride.Pickup.Coordinates) != 2 || len(ride.Drop.Coordinates) != 2 || ride.UserID == "" {
		return nil
	}
	bounds, ok := matching.CorridorBounds(
		matching.Point{Lat: ride.Pickup.Coordinates[1], Lng: ride.Pickup.Coordinates[0]},
		matching.Point{Lat: ride.Drop.Coordinates[1], Lng: ride.Drop.Coordinates[0]},
		matching.DefaultPolicy().CorridorKM,
	)
	if !ok {
		return nil
	}
	return &SearchRegion{searchRegionVersion, ride.UserID, bounds.MinLat, bounds.MaxLat, bounds.MinLng, bounds.MaxLng}
}
func candidateRegionFilter(ride Ride) bson.M {
	region := SearchRegionForRide(ride)
	if region == nil {
		return nil
	}
	return bson.M{"$or": bson.A{
		bson.M{"route_search.version": bson.M{"$ne": searchRegionVersion}},
		bson.M{
			"route_search.min_lat": bson.M{"$lte": region.MaxLat},
			"route_search.max_lat": bson.M{"$gte": region.MinLat},
			"route_search.min_lng": bson.M{"$lte": region.MaxLng},
			"route_search.max_lng": bson.M{"$gte": region.MinLng},
		},
	}}
}
