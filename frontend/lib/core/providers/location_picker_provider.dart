/// Transient picker state. Results never survive a changed query or map choice.
library;

import 'package:flutter/foundation.dart';
import 'package:latlong2/latlong.dart';
import '../services/place_service.dart';

class LocationPickerProvider extends ChangeNotifier {
  final PlaceService _service;
  PlaceSelection? _selection;
  PlaceSearchResults? _results;
  PlaceSearchResults? _selectedSource;
  String _query = '';
  String? _error;
  bool _busy = false;
  bool _disposed = false;
  int _operation = 0;
  LocationPickerProvider(this._service, {PlaceSelection? initial})
    : _selection = initial;

  PlaceSelection? get selection => _selection;
  PlaceSearchResults? get results => _results;
  PlaceSearchResults? get selectedSource => _selectedSource;
  String? get error => _error;
  bool get busy => _busy;
  void _notify() {
    if (!_disposed) notifyListeners();
  }

  bool _current(int operation) => !_disposed && operation == _operation;

  void updateQuery(String query) {
    _query = query;
    // Editing the query invalidates the previous canonical place selection.
    _selection = null;
    _selectedSource = null;
    _operation++;
    _busy = false;
    _results = null;
    _error = null;
    _notify();
  }

  void select(PlaceSelection place) {
    _selectedSource = _results?.places.contains(place) == true
        ? _results
        : null;
    _operation++;
    _busy = false;
    _selection = place;
    _results = null;
    _error = null;
    _notify();
  }

  void selectPoint(LatLng point) {
    if (!PlaceSelection.valid(point.latitude, point.longitude)) return;
    select(PlaceSelection(point, 'Selected map point'));
  }

  void rename(String label) {
    final place = _selection;
    if (place != null)
      _selection = PlaceSelection(place.point, label.trim(), id: place.id);
    _notify();
  }

  Future<void> search() async {
    final query = _query.trim();
    if (query.runes.length < 3 || query.runes.length > 200) {
      _error = 'Enter a place name using 3–200 characters.';
      _notify();
      return;
    }
    final operation = ++_operation;
    _busy = true;
    _error = null;
    _results = null;
    _notify();
    try {
      final results = await _service.search(query);
      if (!_current(operation)) return;
      _results = results;
      if (results.places.isEmpty)
        _error = 'No places found. Try another name or select a map point.';
    } on PlaceLookupException catch (error) {
      if (_current(operation)) _error = error.message;
    } catch (_) {
      if (_current(operation))
        _error = 'Place search failed. Select a map point or try again.';
    } finally {
      if (_current(operation)) {
        _busy = false;
        _notify();
      }
    }
  }

  Future<PlaceSelection?> useCurrentLocation() async {
    final operation = ++_operation;
    _busy = true;
    _error = null;
    _results = null;
    _notify();
    try {
      final place = await _service.currentLocation();
      if (!_current(operation)) return null;
      _selection = place;
      _selectedSource = null;
      return place;
    } on PlaceLookupException catch (error) {
      if (_current(operation)) _error = error.message;
    } catch (_) {
      if (_current(operation))
        _error = 'Could not determine your location. Select a map point.';
    } finally {
      if (_current(operation)) {
        _busy = false;
        _notify();
      }
    }
    return null;
  }

  @override
  void dispose() {
    _disposed = true;
    _operation++;
    super.dispose();
  }
}
