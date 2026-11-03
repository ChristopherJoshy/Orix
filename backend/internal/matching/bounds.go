package matching

import "math"

// Bounds is a conservative region for a minor great-circle route plus padding.
// It is a database prefilter, never evidence of actual route compatibility.
type Bounds struct {
	MinLat, MaxLat float64
	MinLng, MaxLng float64
}

func CorridorBounds(pickup, drop Point, paddingKM float64) (Bounds, bool) {
	if !validPoint(pickup) || !validPoint(drop) || math.IsNaN(paddingKM) || math.IsInf(paddingKM, 0) || paddingKM < 0 {
		return Bounds{}, false
	}
	world := Bounds{-90, 90, -180, 180}
	length := distance(pickup, drop)
	// Very long/ambiguous routes intentionally skip geographic pruning. This
	// keeps polar and nearly antipodal geometry out of a local-trip optimization.
	if length > 1000 {
		return world, true
	}
	// Every point on the arc is at most half its length from one endpoint. A
	// rectangle enclosing both endpoint balls therefore encloses the whole arc.
	radius := length/2 + paddingKM
	latMargin := radius/6371*180/math.Pi + 1e-9
	bounds := Bounds{
		MinLat: math.Max(-90, math.Min(pickup.Lat, drop.Lat)-latMargin),
		MaxLat: math.Min(90, math.Max(pickup.Lat, drop.Lat)+latMargin),
		MinLng: -180, MaxLng: 180,
	}
	maxLatitude := math.Max(math.Abs(bounds.MinLat), math.Abs(bounds.MaxLat))
	if maxLatitude >= 89.9 || math.Abs(pickup.Lng-drop.Lng) > 180 {
		return bounds, true
	}
	// Longitude changes faster near a pole. Use the largest possible latitude
	// within the region; dateline-crossing results widen to all longitudes.
	lngMargin := latMargin/math.Cos(rad(maxLatitude)) + 1e-9
	minLng, maxLng := math.Min(pickup.Lng, drop.Lng)-lngMargin, math.Max(pickup.Lng, drop.Lng)+lngMargin
	if minLng > -180 && maxLng < 180 {
		bounds.MinLng, bounds.MaxLng = minLng, maxLng
	}
	return bounds, true
}
