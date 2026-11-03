/// Own endpoints or a member-authorized road itinerary, with visible attribution.
library;

import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:latlong2/latlong.dart';
import '../config/map_config.dart';

import '../providers/group_provider.dart';
import '../theme/app_theme.dart';

class TripRouteMap extends StatelessWidget {
  final double? pickupLat;
  final double? pickupLng;
  final double? dropLat;
  final double? dropLng;
  final RoadItinerary? roadPlan;
  final double height;
  const TripRouteMap({
    super.key,
    this.pickupLat,
    this.pickupLng,
    this.dropLat,
    this.dropLng,
    this.roadPlan,
    this.height = 260,
  });

  bool _valid(double? lat, double? lng) =>
      lat != null &&
      lng != null &&
      lat.isFinite &&
      lng.isFinite &&
      lat.abs() <= 90 &&
      lng.abs() <= 180;

  @override
  Widget build(BuildContext context) {
    final route =
        roadPlan?.geometry
            .map((point) => LatLng(point.latitude, point.longitude))
            .toList() ??
        <LatLng>[];
    final endpoints = <LatLng>[
      if (_valid(pickupLat, pickupLng)) LatLng(pickupLat!, pickupLng!),
      if (_valid(dropLat, dropLng)) LatLng(dropLat!, dropLng!),
    ];
    final stops = roadPlan?.stops ?? <PlannedRoadStop>[];
    final points = [
      ...route,
      ...endpoints,
      for (final stop in stops)
        LatLng(stop.point.latitude, stop.point.longitude),
    ];
    if (points.isEmpty) return const SizedBox.shrink();
    final usableRoute = route.length > 1;
    final bounds = LatLngBounds.fromPoints(points);
    final distinct = points.any((point) => point != points.first);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        ClipRRect(
          borderRadius: BorderRadius.zero,
          child: SizedBox(
            height: height,
            child: FlutterMap(
              key: ValueKey(
                '${points.first.latitude}:${points.first.longitude}:${points.last.latitude}:${points.last.longitude}:${route.length}',
              ),
              options: MapOptions(
                initialCenter: points.first,
                initialZoom: 14,
                initialCameraFit: distinct
                    ? CameraFit.bounds(
                        bounds: bounds,
                        padding: const EdgeInsets.all(36),
                        maxZoom: 15,
                      )
                    : null,
                interactionOptions: const InteractionOptions(
                  flags: InteractiveFlag.all & ~InteractiveFlag.rotate,
                ),
              ),
              children: [
                MapConfig.tiles(),
                if (_valid(pickupLat, pickupLng))
                  CircleLayer(
                    circles: [
                      CircleMarker(
                        point: LatLng(pickupLat!, pickupLng!),
                        radius: 300,
                        useRadiusInMeter: true,
                        color: AppColors.primary.withValues(alpha: .06),
                        borderColor: AppColors.primary.withValues(alpha: .5),
                        borderStrokeWidth: 1,
                      ),
                    ],
                  ),
                if (usableRoute)
                  PolylineLayer(
                    polylines: [
                      Polyline(
                        points: route,
                        strokeWidth: 5,
                        color: AppColors.primary,
                      ),
                    ],
                  ),
                MarkerLayer(
                  markers: [
                    if (stops.isNotEmpty)
                      for (var index = 0; index < stops.length; index++)
                        Marker(
                          point: LatLng(
                            stops[index].point.latitude,
                            stops[index].point.longitude,
                          ),
                          width: 34,
                          height: 34,
                          child: Tooltip(
                            message:
                                '${stops[index].kind == 'pickup' ? 'Pickup' : 'Drop-off'} ${index + 1}',
                            child: CircleAvatar(
                              backgroundColor: AppColors.primary,
                              child: Text(
                                '${index + 1}',
                                style: const TextStyle(color: Colors.white),
                              ),
                            ),
                          ),
                        ),
                    if (stops.isEmpty)
                      for (var index = 0; index < endpoints.length; index++)
                        Marker(
                          point: endpoints[index],
                          width: 42,
                          height: 42,
                          child: Tooltip(
                            message: index == 0
                                ? 'Your pickup'
                                : 'Your destination',
                            child: Icon(
                              index == 0
                                  ? Icons.trip_origin
                                  : Icons.location_on,
                              color: index == 0
                                  ? AppColors.primary
                                  : AppColors.success,
                              size: 34,
                            ),
                          ),
                        ),
                  ],
                ),
                MapConfig.credits(),
              ],
            ),
          ),
        ),
        const SizedBox(height: 6),
        Text(
          usableRoute
              ? 'Planned road route · numbered pickups and drop-offs'
              : 'Your selected locations · a road route appears for a confirmed lift',
          style: Theme.of(context).textTheme.bodySmall,
        ),
      ],
    );
  }
}
