/// Place lookup and user-initiated device location. UI searches are debounced.
library;

import 'package:geolocator/geolocator.dart';
import 'package:latlong2/latlong.dart';

import 'api_service.dart';

class PlaceSelection {
  final LatLng point;
  final String label;
  final String? id;
  const PlaceSelection(this.point, this.label, {this.id});
  static bool valid(double latitude, double longitude) =>
      latitude.isFinite &&
      longitude.isFinite &&
      latitude.abs() <= 90 &&
      longitude.abs() <= 180;
  factory PlaceSelection.fromJson(Map<String, dynamic> json) {
    final latitude = (json['latitude'] as num).toDouble();
    final longitude = (json['longitude'] as num).toDouble();
    if (!valid(latitude, longitude))
      throw const FormatException('Invalid place');
    return PlaceSelection(
      LatLng(latitude, longitude),
      json['label'] as String,
      id: json['place_id'] as String?,
    );
  }
  String get coordinateLabel =>
      '${point.latitude.toStringAsFixed(5)}, ${point.longitude.toStringAsFixed(5)}';
}

class PlaceSearchResults {
  final List<PlaceSelection> places;
  final String attribution;
  final Uri? attributionUrl;
  const PlaceSearchResults(this.places, this.attribution, this.attributionUrl);
}

class PlaceService {
  final ApiService _api;
  PlaceService(this._api);

  Future<PlaceSearchResults> search(String query) async {
    final response = await _api.get<Map<String, dynamic>>(
      '/places/search',
      queryParams: {'query': query},
      forceRefresh: true,
    );
    if (!response.success || response.data == null) {
      throw PlaceLookupException(
        response.error ??
            'Place search is unavailable. Select a point on the map.',
      );
    }
    final json = response.data!;
    final url = Uri.tryParse(json['attribution_url'] as String? ?? '');
    return PlaceSearchResults(
      List.unmodifiable(
        (json['places'] as List<dynamic>).map(
          (row) => PlaceSelection.fromJson(row as Map<String, dynamic>),
        ),
      ),
      json['attribution'] as String? ?? '',
      url?.scheme == 'https' && url!.host.isNotEmpty && url.userInfo.isEmpty
          ? url
          : null,
    );
  }

  Future<PlaceSelection> currentLocation() async {
    if (!await Geolocator.isLocationServiceEnabled()) {
      throw const PlaceLookupException(
        'Location services are off. Enable them or select a map point.',
      );
    }
    var permission = await Geolocator.checkPermission();
    if (permission == LocationPermission.denied)
      permission = await Geolocator.requestPermission();
    if (permission == LocationPermission.denied ||
        permission == LocationPermission.deniedForever) {
      throw const PlaceLookupException(
        'Location permission is unavailable. You can still select a point on the map.',
      );
    }
    final location = await Geolocator.getCurrentPosition(
      locationSettings: const LocationSettings(
        accuracy: LocationAccuracy.high,
        timeLimit: Duration(seconds: 15),
      ),
    );
    if (!PlaceSelection.valid(location.latitude, location.longitude)) {
      throw const PlaceLookupException(
        'Could not determine your location. Select a map point.',
      );
    }
    return PlaceSelection(
      LatLng(location.latitude, location.longitude),
      'Current location',
    );
  }
}

class PlaceLookupException implements Exception {
  final String message;
  const PlaceLookupException(this.message);
}
