package matching

import (
	"math"
	"testing"
)

func withinBounds(bounds Bounds, point Point) bool {
	return point.Lat >= bounds.MinLat && point.Lat <= bounds.MaxLat && point.Lng >= bounds.MinLng && point.Lng <= bounds.MaxLng
}

// Interpolate on the sphere rather than a latitude/longitude line. This catches
// overly narrow boxes for the great-circle bulge at higher latitudes.
func arcPoint(a, b Point, fraction float64) Point {
	vector := func(p Point) [3]float64 {
		latitude, longitude := rad(p.Lat), rad(p.Lng)
		return [3]float64{math.Cos(latitude) * math.Cos(longitude), math.Cos(latitude) * math.Sin(longitude), math.Sin(latitude)}
	}
	av, bv := vector(a), vector(b)
	angle := distance(a, b) / 6371
	left, right := math.Sin((1-fraction)*angle)/math.Sin(angle), math.Sin(fraction*angle)/math.Sin(angle)
	x, y, z := left*av[0]+right*bv[0], left*av[1]+right*bv[1], left*av[2]+right*bv[2]
	return Point{Lat: math.Atan2(z, math.Hypot(x, y)) * 180 / math.Pi, Lng: math.Atan2(y, x) * 180 / math.Pi}
}
func TestBoundsContainSphericalRouteAndNearbyPoints(t *testing.T) {
	pairs := [][2]Point{
		{{10, 76}, {10.2, 76.1}},
		{{70, -10}, {70, 10}},
		{{-70, -10}, {-70, 10}},
		{{89.5, 10}, {89.5, 90}},
		{{5, 179.9}, {5, -179.9}},
	}
	for _, pair := range pairs {
		bounds, ok := CorridorBounds(pair[0], pair[1], 1)
		if !ok {
			t.Fatal("valid route rejected")
		}
		for step := 0; step <= 100; step++ {
			point := arcPoint(pair[0], pair[1], float64(step)/100)
			if !withinBounds(bounds, point) {
				t.Fatalf("route escaped bounds: %+v %+v", point, bounds)
			}
			// A short north/south displacement stays within the one-kilometer margin.
			for _, offset := range []float64{-.005, .005} {
				nearby := point
				nearby.Lat += offset
				if math.Abs(nearby.Lat) <= 90 && !withinBounds(bounds, nearby) {
					t.Fatalf("padding excluded nearby point: %+v", nearby)
				}
			}
		}
	}
}
func TestBoundsRemainUsefulForLocalTrips(t *testing.T) {
	bounds, ok := CorridorBounds(Point{10, 76}, Point{10.1, 76}, 1)
	if !ok || withinBounds(bounds, Point{20, 76}) || withinBounds(bounds, Point{10, 86}) {
		t.Fatal("local bounds did not prune distant regions")
	}
	long, ok := CorridorBounds(Point{0, 0}, Point{0, 170}, 1)
	if !ok || long != (Bounds{-90, 90, -180, 180}) {
		t.Fatal("long route should skip geographic pruning")
	}
	dateline, _ := CorridorBounds(Point{5, 179.9}, Point{5, -179.9}, 1)
	if dateline.MinLng != -180 || dateline.MaxLng != 180 {
		t.Fatal("dateline bounds must keep both sides eligible")
	}
}
func TestBoundsRejectInvalidInput(t *testing.T) {
	for _, point := range []Point{{91, 0}, {0, 181}, {math.NaN(), 0}, {0, math.Inf(1)}} {
		if _, ok := CorridorBounds(point, Point{10, 76}, 1); ok {
			t.Fatal("invalid coordinates accepted")
		}
	}
	for _, padding := range []float64{-1, math.NaN(), math.Inf(1)} {
		if _, ok := CorridorBounds(Point{10, 76}, Point{10.1, 76}, padding); ok {
			t.Fatal("invalid padding accepted")
		}
	}
}
