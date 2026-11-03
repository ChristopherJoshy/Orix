/// Ride Provider
///
/// Manages ride requests and matching status.
/// Optimized to minimize unnecessary rebuilds.
library;

import 'package:flutter/foundation.dart';
import 'package:flutter/scheduler.dart' show SchedulerPhase;
import 'package:flutter/widgets.dart';

import '../services/api_service.dart';
import '../utils/time_utils.dart';

/// Time window for rides
class TimeWindow {
  final String start;
  final String end;

  TimeWindow({required this.start, required this.end});

  factory TimeWindow.fromJson(Map<String, dynamic> json) {
    return TimeWindow(
      start: json['start'] as String,
      end: json['end'] as String,
    );
  }

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is TimeWindow && start == other.start && end == other.end;

  @override
  int get hashCode => Object.hash(start, end);
}

/// Ride request model
class RideRequest {
  final double? pickupLat;
  final double? pickupLng;
  final double? dropLat;
  final double? dropLng;
  final DateTime? departureStart;
  final DateTime? departureEnd;
  final String? pickupPlaceId, dropPlaceId;
  final String requestId;
  final String pickupLabel;
  final String dropLabel;
  final String date;
  final TimeWindow timeWindow;
  final String mode;
  final bool driver;
  final int seats;
  final bool allowRiders;
  final bool femaleOnly;
  final String status;
  final String? groupId;
  final DateTime createdAt;
  final DateTime expiresAt;
  final int timeRemainingSeconds;

  RideRequest({
    this.pickupPlaceId,
    this.dropPlaceId,
    this.pickupLat,
    this.pickupLng,
    this.dropLat,
    this.dropLng,
    this.departureStart,
    this.departureEnd,
    required this.requestId,
    required this.pickupLabel,
    required this.dropLabel,
    required this.date,
    required this.timeWindow,
    this.mode = 'auto',
    this.driver = false,
    this.seats = 0,
    required this.allowRiders,
    required this.femaleOnly,
    required this.status,
    this.groupId,
    required this.createdAt,
    required this.expiresAt,
    required this.timeRemainingSeconds,
  });

  factory RideRequest.fromJson(Map<String, dynamic> json) {
    double? coordinate(String field, int index) {
      final point = json[field];
      if (point is! Map<String, dynamic>) return null;
      final coordinates = point['coordinates'];
      if (coordinates is! List ||
          coordinates.length != 2 ||
          coordinates[index] is! num)
        return null;
      return (coordinates[index] as num).toDouble();
    }

    return RideRequest(
      pickupPlaceId: json["pickup_place_id"] as String?,
      dropPlaceId: json["drop_place_id"] as String?,
      pickupLng: coordinate('pickup', 0),
      pickupLat: coordinate('pickup', 1),
      dropLng: coordinate('drop', 0),
      dropLat: coordinate('drop', 1),
      departureStart: TimeUtils.tryParseUtc(json['departure_start']),
      departureEnd: TimeUtils.tryParseUtc(json['departure_end']),
      requestId: json['request_id'] as String,
      pickupLabel: json['pickup_label'] as String,
      dropLabel: json['drop_label'] as String,
      date: json['date'] as String,
      timeWindow: TimeWindow.fromJson(
        json['time_window'] as Map<String, dynamic>,
      ),
      mode: json['mode'] as String? ?? 'auto',
      driver: json['driver'] as bool? ?? false,
      seats: json['seats'] as int? ?? 0,
      allowRiders: json['allow_riders'] as bool,
      femaleOnly: json['female_only'] as bool,
      status: json['status'] as String,
      groupId: json['group_id'] as String?,
      createdAt: TimeUtils.parseUtc(json['created_at'] as String),
      expiresAt: TimeUtils.parseUtc(json['expires_at'] as String),
      timeRemainingSeconds: json['time_remaining_seconds'] as int,
    );
  }
}

/// Matching status
class MatchingStatus {
  final RideRequest? request;
  final String? groupId;
  final String? groupStatus;
  final int groupSize;

  bool get canSearch =>
      hasActiveRequest &&
      (status == 'pending' ||
          status == 'matching' ||
          (status == 'matched' &&
              groupStatus == 'pending' &&
              groupSize == 1 &&
              !myReadinessConfirmed));
  bool get canEdit => canSearch && request != null;
  final bool hasActiveRequest;
  final String? requestId;
  final String? status;
  final String? pickupLabel;
  final String? dropLabel;
  final String? date;
  final TimeWindow? timeWindow;
  final int? timeRemainingSeconds;
  final String? pendingConfirmationGroupId;
  final bool myReadinessConfirmed;
  final double? matchScore;
  final double? routeOverlapScore;
  final double? timeCompatibilityScore;

  MatchingStatus({
    this.request,
    this.groupId,
    this.groupStatus,
    this.groupSize = 0,
    required this.hasActiveRequest,
    this.requestId,
    this.status,
    this.pickupLabel,
    this.dropLabel,
    this.date,
    this.timeWindow,
    this.timeRemainingSeconds,
    this.pendingConfirmationGroupId,
    this.myReadinessConfirmed = false,
    this.matchScore,
    this.routeOverlapScore,
    this.timeCompatibilityScore,
  });

  factory MatchingStatus.fromJson(Map<String, dynamic> json) {
    return MatchingStatus(
      request: json['request'] is Map<String, dynamic>
          ? RideRequest.fromJson(json['request'] as Map<String, dynamic>)
          : null,
      groupId: json['group_id'] as String?,
      groupStatus: json['group_status'] as String?,
      groupSize: json['group_size'] as int? ?? 0,
      hasActiveRequest: json['has_active_request'] as bool,
      requestId: json['request_id'] as String?,
      status: json['status'] as String?,
      pickupLabel: json['pickup_label'] as String?,
      dropLabel: json['drop_label'] as String?,
      date: json['date'] as String?,
      timeWindow: json['time_window'] != null
          ? TimeWindow.fromJson(json['time_window'] as Map<String, dynamic>)
          : null,
      timeRemainingSeconds: json['time_remaining_seconds'] as int?,
      pendingConfirmationGroupId:
          json['pending_confirmation_group_id'] as String?,
      myReadinessConfirmed: json['my_readiness_confirmed'] as bool? ?? false,
      matchScore: (json['match_score'] as num?)?.toDouble(),
      routeOverlapScore: (json['route_overlap_score'] as num?)?.toDouble(),
      timeCompatibilityScore: (json['time_compatibility_score'] as num?)
          ?.toDouble(),
    );
  }
}

class RouteSuggestion {
  final String groupId;
  final String pickupLabel;
  final String dropLabel;
  final int memberCount;
  final double score;
  final DateTime departure;
  final bool roadVerificationRequired;
  const RouteSuggestion({
    required this.groupId,
    required this.pickupLabel,
    required this.dropLabel,
    required this.memberCount,
    required this.score,
    required this.departure,
    required this.roadVerificationRequired,
  });
  factory RouteSuggestion.fromJson(Map<String, dynamic> json) =>
      RouteSuggestion(
        groupId: json['group_id'] as String,
        pickupLabel: json['pickup_label'] as String,
        dropLabel: json['drop_label'] as String,
        memberCount: json['member_count'] as int,
        score: (json['match_score'] as num).toDouble(),
        departure: TimeUtils.parseUtc(json['departure'] as String),
        roadVerificationRequired:
            json['road_verification_required'] as bool? ?? false,
      );
}

/// Own trip state and route-compatible group reservations, scoped to one account.
class RideProvider extends ChangeNotifier {
  final ApiService _apiService;
  String? _userId;
  String? _identity;
  int _session = 0;
  int _statusRequest = 0;
  bool _disposed = false;
  bool _notificationScheduled = false;
  bool _isLoading = false;
  bool _statusBusy = false;
  bool _suggestionsBusy = false;
  bool _historyBusy = false;
  String? _error;
  MatchingStatus? _matchingStatus;
  List<RideRequest> _rideHistory = [];
  List<RouteSuggestion> _suggestions = [];
  String? _historyCursor;
  DateTime? _lastStatusFetch;

  RideProvider(this._apiService);
  bool get isLoading => _isLoading || _statusBusy || _historyBusy;
  bool get isMutating => _isLoading;
  bool get isLoadingSuggestions => _suggestionsBusy;
  String? get error => _error;
  MatchingStatus? get matchingStatus => _matchingStatus;
  RideRequest? get activeRequest => _matchingStatus?.request;
  List<RideRequest> get rideHistory => List.unmodifiable(_rideHistory);
  List<RouteSuggestion> get suggestions => List.unmodifiable(_suggestions);
  bool get hasMoreHistory => _historyCursor != null;
  DateTime? get lastStatusFetch => _lastStatusFetch;
  bool get hasActiveRequest => _matchingStatus?.hasActiveRequest ?? false;
  bool _valid(int session) =>
      !_disposed &&
      session == _session &&
      _identity == _apiService.authIdentity;

  void _notify() {
    if (_disposed) return;
    if (WidgetsBinding.instance.schedulerPhase ==
        SchedulerPhase.persistentCallbacks) {
      if (_notificationScheduled) return;
      _notificationScheduled = true;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        _notificationScheduled = false;
        if (!_disposed) notifyListeners();
      });
    } else {
      notifyListeners();
    }
  }

  void updateUser(String? id) {
    if (id == _userId) return;
    _session++;
    _userId = id;
    _identity = id == null ? null : _apiService.authIdentity;
    _matchingStatus = null;
    _rideHistory = [];
    _suggestions = [];
    _historyCursor = null;
    _error = null;
    _lastStatusFetch = null;
    _isLoading = _statusBusy = _suggestionsBusy = _historyBusy = false;
    _notify();
  }

  Future<void> fetchMatchingStatus() async {
    if (_userId == null || _statusBusy) return;
    final session = _session;
    final request = ++_statusRequest;
    _statusBusy = true;
    _notify();
    try {
      final response = await _apiService.get<Map<String, dynamic>>(
        '/rides/status',
        forceRefresh: true,
      );
      if (!_valid(session) || request != _statusRequest) return;
      if (response.success && response.data != null) {
        _matchingStatus = MatchingStatus.fromJson(response.data!);
        _lastStatusFetch = DateTime.now();
        _error = null;
        if (_matchingStatus?.canSearch != true) _suggestions = [];
      } else {
        _error = response.error;
      }
    } catch (_) {
      if (_valid(session) && request == _statusRequest)
        _error = 'Could not refresh your trip.';
    } finally {
      if (_valid(session) && request == _statusRequest) {
        _statusBusy = false;
        _notify();
      }
    }
  }

  Future<void> fetchSuggestions() async {
    final id = _matchingStatus?.requestId;
    if (_userId == null ||
        id == null ||
        _matchingStatus?.canSearch != true ||
        _suggestionsBusy)
      return;
    final session = _session;
    _suggestionsBusy = true;
    _notify();
    try {
      final response = await _apiService.get<Map<String, dynamic>>(
        '/rides/${Uri.encodeComponent(id)}/suggestions',
        forceRefresh: true,
      );
      if (!_valid(session) || _matchingStatus?.requestId != id) return;
      if (response.success && response.data != null) {
        _suggestions = (response.data!['suggestions'] as List<dynamic>? ?? [])
            .map(
              (data) => RouteSuggestion.fromJson(data as Map<String, dynamic>),
            )
            .toList();
      } else if (response.statusCode == 409) {
        _suggestions = [];
        await fetchMatchingStatus();
      } else {
        _error = response.error;
      }
    } catch (_) {
      if (_valid(session)) _error = 'Could not load compatible groups.';
    } finally {
      if (_valid(session)) {
        _suggestionsBusy = false;
        _notify();
      }
    }
  }

  Future<String?> joinSuggestedGroup(String groupId) async {
    final id = _matchingStatus?.requestId;
    if (_userId == null || id == null || _isLoading) return null;
    final session = _session;
    // Invalidate reads started before a mutation; the committed state wins.
    _statusRequest++;
    _statusBusy = false;
    _isLoading = true;
    _error = null;
    _notify();
    try {
      final response = await _apiService.post<Map<String, dynamic>>(
        '/rides/${Uri.encodeComponent(id)}/group',
        body: {'group_id': groupId},
      );
      if (!_valid(session)) return null;
      if (!response.success) {
        _error = response.error;
        return null;
      }
      await fetchMatchingStatus();
      return response.data?['group_id'] as String?;
    } catch (_) {
      if (_valid(session))
        _error = 'Could not confirm group membership. Refresh and try again.';
      return null;
    } finally {
      if (_valid(session)) {
        _isLoading = false;
        _notify();
      }
    }
  }

  Future<RideRequest?> createRideRequest({
    String? pickupPlaceId,
    String? dropPlaceId,
    required double pickupLat,
    required double pickupLng,
    required String pickupLabel,
    required double dropLat,
    required double dropLng,
    required String dropLabel,
    required String date,
    required String timeWindowStart,
    required String timeWindowEnd,
    bool allowRiders = false,
    bool femaleOnly = false,
    String mode = 'auto',
    bool driver = false,
    int seats = 0,
    String? requestId,
  }) async {
    if (_userId == null || _isLoading) return null;
    final session = _session;
    // Invalidate reads started before a mutation; the committed state wins.
    _statusRequest++;
    _statusBusy = false;
    _isLoading = true;
    _error = null;
    _notify();
    final body = <String, dynamic>{
      if (pickupPlaceId != null) 'pickup_place_id': pickupPlaceId,
      if (dropPlaceId != null) 'drop_place_id': dropPlaceId,
      'pickup_lat': pickupLat,
      'pickup_lng': pickupLng,
      'pickup_label': pickupLabel,
      'drop_lat': dropLat,
      'drop_lng': dropLng,
      'drop_label': dropLabel,
      'date': date,
      'time_window_start': timeWindowStart,
      'time_window_end': timeWindowEnd,
      'timezone_offset_minutes': DateTime.parse(
        '${date}T$timeWindowStart:00',
      ).timeZoneOffset.inMinutes,
      'mode': mode,
      'driver': driver,
      'seats': driver ? seats : 0,
      'allow_riders': allowRiders,
      'female_only': femaleOnly,
    };
    try {
      final response = requestId == null
          ? await _apiService.post<Map<String, dynamic>>('/rides', body: body)
          : await _apiService.put<Map<String, dynamic>>(
              '/rides/${Uri.encodeComponent(requestId)}',
              body: body,
            );
      if (!_valid(session)) return null;
      if (response.success && response.data != null) {
        final ride = RideRequest.fromJson(response.data!);
        await fetchMatchingStatus();
        return ride;
      }
      final reason = response.error;
      _error = reason;
      // A lost response can still mean a committed request. Refresh the single
      // active slot instead of claiming that a second request should be created.
      if (response.statusCode == 0 || response.statusCode == 409) {
        await fetchMatchingStatus();
        if (_valid(session)) _error = reason;
      }
    } catch (_) {
      if (_valid(session))
        _error = 'Could not confirm this trip. Refresh before trying again.';
    } finally {
      if (_valid(session)) {
        _isLoading = false;
        _notify();
      }
    }
    return null;
  }

  Future<bool> cancelRideRequest(String requestId) async {
    if (_userId == null || _isLoading) return false;
    final session = _session;
    // Invalidate reads started before a mutation; the committed state wins.
    _statusRequest++;
    _statusBusy = false;
    _isLoading = true;
    _error = null;
    _notify();
    try {
      final response = await _apiService.delete<Map<String, dynamic>>(
        '/rides/${Uri.encodeComponent(requestId)}',
      );
      if (!_valid(session)) return false;
      if (!response.success) {
        _error = response.error;
        return false;
      }
      _matchingStatus = MatchingStatus(hasActiveRequest: false);
      _suggestions = [];
      return true;
    } catch (_) {
      if (_valid(session))
        _error = 'Could not confirm cancellation. Refresh your trip.';
      return false;
    } finally {
      if (_valid(session)) {
        _isLoading = false;
        _notify();
      }
    }
  }

  Future<void> fetchRideHistory({
    bool includeHistory = true,
    bool loadMore = false,
  }) async {
    if (_userId == null || _historyBusy || (loadMore && _historyCursor == null))
      return;
    final session = _session;
    _historyBusy = true;
    _notify();
    try {
      final response = await _apiService.get<List<dynamic>>(
        '/rides',
        forceRefresh: true,
        queryParams: {
          'include_history': includeHistory.toString(),
          'limit': '50',
          if (loadMore) 'before': _historyCursor!,
        },
      );
      if (!_valid(session)) return;
      if (response.success && response.data != null) {
        final rows = response.data!
            .map((data) => RideRequest.fromJson(data as Map<String, dynamic>))
            .toList();
        final byId = <String, RideRequest>{
          if (loadMore)
            for (final ride in _rideHistory) ride.requestId: ride,
        };
        for (final ride in rows) {
          byId[ride.requestId] = ride;
        }
        _rideHistory = byId.values.toList()
          ..sort((a, b) => b.createdAt.compareTo(a.createdAt));
        _historyCursor = response.nextCursor;
      } else {
        _error = response.error;
      }
    } catch (_) {
      if (_valid(session)) _error = 'Could not load trip history.';
    } finally {
      if (_valid(session)) {
        _historyBusy = false;
        _notify();
      }
    }
  }

  void clearMatchingStatus() {
    _statusRequest++;
    _matchingStatus = null;
    _suggestions = [];
    _error = null;
    _statusBusy = false;
    _notify();
  }

  @override
  void dispose() {
    _disposed = true;
    _session++;
    super.dispose();
  }
}
