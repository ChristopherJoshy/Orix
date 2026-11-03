/// Notification Provider
///
/// Manages app notifications with optimized rebuild behavior.
library;

import 'dart:async';

import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/scheduler.dart' show SchedulerPhase;
import 'package:flutter/widgets.dart';

import '../services/api_service.dart';
import '../services/notification_service.dart';
import '../utils/time_utils.dart';

/// App notification model
class AppNotification {
  final String id;
  final String title;
  final String body;
  final String type;
  final bool isRead;
  final DateTime createdAt;
  final Map<String, dynamic>? data;

  AppNotification({
    required this.id,
    required this.title,
    required this.body,
    required this.type,
    required this.isRead,
    required this.createdAt,
    this.data,
  });

  factory AppNotification.fromJson(Map<String, dynamic> json) {
    return AppNotification(
      id: json['notification_id'] as String,
      title: json['title'] as String,
      body: json['body'] as String,
      type: json['type'] as String? ?? 'system',
      isRead: json['read'] as bool? ?? false,
      createdAt: TimeUtils.parseUtc(json['created_at'] as String),
      data: json['data'] as Map<String, dynamic>?,
    );
  }

  AppNotification copyWith({bool? isRead}) {
    return AppNotification(
      id: id,
      title: title,
      body: body,
      type: type,
      isRead: isRead ?? this.isRead,
      createdAt: createdAt,
      data: data,
    );
  }
}

/// An account-scoped inbox. Late responses from a previous Firebase identity
/// must never populate another account, even before auth routing finishes.
class NotificationProvider extends ChangeNotifier {
  final ApiService _apiService;
  final NotificationService? _notificationService;
  String? _userId;
  String? _identity;
  int _session = 0;
  int _readVersion = 0;
  bool _disposed = false;
  bool _scheduled = false;
  bool _isLoading = false;
  bool _isMutating = false;
  String? _error;
  List<AppNotification> _notifications = [];
  int _unreadCount = 0;
  String? _cursor;
  Timer? _debounceTimer;
  String? _pendingAlarmGroupId;
  String? _pendingAlarmRideTime;
  String? _pendingAlarmPickupSummary;

  NotificationProvider(this._apiService, [this._notificationService]) {
    _notificationService?.onNotificationReceived = _onMessage;
  }
  bool isForUser(String? id) => id != null && id == _userId && _valid(_session);
  String? get accountId => _userId;
  bool get isLoading => _isLoading;
  bool get isMutating => _isMutating;
  bool get hasMore => _cursor != null;
  String? get error => _error;
  List<AppNotification> get notifications => List.unmodifiable(_notifications);
  int get unreadCount => _unreadCount;
  bool get hasUnread => _unreadCount > 0;
  String? get pendingAlarmGroupId => _pendingAlarmGroupId;
  String? get pendingAlarmRideTime => _pendingAlarmRideTime;
  String? get pendingAlarmPickupSummary => _pendingAlarmPickupSummary;
  bool _valid(int session) =>
      !_disposed &&
      session == _session &&
      _userId != null &&
      _identity == _apiService.authIdentity;

  void _notify() {
    if (_disposed) return;
    if (WidgetsBinding.instance.schedulerPhase ==
        SchedulerPhase.persistentCallbacks) {
      if (_scheduled) return;
      _scheduled = true;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        _scheduled = false;
        if (!_disposed) notifyListeners();
      });
    } else {
      notifyListeners();
    }
  }

  void updateUser(String? userId) {
    final identity = userId == null ? null : _apiService.authIdentity;
    if (_userId == userId && _identity == identity) return;
    _session++;
    _readVersion++;
    _userId = userId;
    _identity = identity;
    _debounceTimer?.cancel();
    _notifications = [];
    _unreadCount = 0;
    _cursor = null;
    _error = null;
    _isLoading = _isMutating = false;
    _clearAlarm();
    _notify();
  }

  void _onMessage(RemoteMessage message) {
    if (!_valid(_session)) return;
    // New pushes identify their recipient; never queue a previous account's
    // pickup after switching users on a shared installation.
    final recipient = message.data['recipient_user_id'];
    if (recipient is String && recipient != _userId) return;
    final payload = RideAlarmPayload.fromData(message.data);
    if (payload != null)
      setPendingAlarmFromService(
        groupId: payload['group_id']!,
        rideTime: payload['ride_time'] ?? '',
        pickupSummary: payload['pickup_summary'] ?? '',
        recipientUserId: payload['recipient_user_id'],
      );
    final session = _session;
    _debounceTimer?.cancel();
    _debounceTimer = Timer(const Duration(milliseconds: 500), () {
      if (_valid(session)) unawaited(fetchNotifications());
    });
  }

  void _clearAlarm() {
    _pendingAlarmGroupId = _pendingAlarmRideTime = _pendingAlarmPickupSummary =
        null;
  }

  void clearPendingAlarm() {
    _clearAlarm();
    _notify();
  }

  void setPendingAlarmFromService({
    required String groupId,
    required String rideTime,
    required String pickupSummary,
    String? recipientUserId,
  }) {
    if (!_valid(_session) ||
        groupId.isEmpty ||
        (recipientUserId != null && recipientUserId != _userId))
      return;
    _pendingAlarmGroupId = groupId;
    _pendingAlarmRideTime = rideTime;
    _pendingAlarmPickupSummary = pickupSummary;
    _notify();
  }

  Future<void> fetchNotifications({bool loadMore = false}) async {
    if (!_valid(_session) ||
        _isLoading ||
        _isMutating ||
        (loadMore && _cursor == null))
      return;
    final session = _session;
    final read = ++_readVersion;
    _isLoading = true;
    _error = null;
    _notify();
    try {
      final response = await _apiService.get<Map<String, dynamic>>(
        '/notifications',
        forceRefresh: true,
        queryParams: {'limit': '50', if (loadMore) 'before': _cursor!},
      );
      if (!_valid(session) || read != _readVersion) return;
      if (!response.success || response.data == null) {
        _error = response.error;
        return;
      }
      final data = response.data!;
      final records = (data['notifications'] as List<dynamic>? ?? []).map(
        (row) => AppNotification.fromJson(row as Map<String, dynamic>),
      );
      final byId = <String, AppNotification>{
        if (loadMore)
          for (final item in _notifications) item.id: item,
      };
      for (final item in records) {
        byId[item.id] = item;
      }
      _notifications = byId.values.toList()
        ..sort((a, b) {
          final time = b.createdAt.compareTo(a.createdAt);
          return time == 0 ? b.id.compareTo(a.id) : time;
        });
      // This is a server-wide inbox count, not the count on this loaded page.
      _unreadCount = (data['unread_count'] as num?)?.toInt() ?? 0;
      _cursor = response.nextCursor;
    } catch (_) {
      if (_valid(session) && read == _readVersion)
        _error = 'Could not load notifications. Try again.';
    } finally {
      if (_valid(session) && read == _readVersion) {
        _isLoading = false;
        _notify();
      }
    }
  }

  Future<bool> markAsRead(String id) => _change('read', id: id);
  Future<bool> markAllAsRead() => _change('read-all');
  Future<bool> deleteNotification(String id) => _change('delete', id: id);
  Future<bool> clearAllNotifications() => _change('delete');

  Future<bool> _change(String action, {String? id}) async {
    if (!_valid(_session) || _isMutating) return false;
    final session = _session;
    // A response captured before this write cannot restore obsolete read/deleted
    // state. The final refresh obtains the authoritative global unread count.
    _readVersion++;
    _isLoading = false;
    _isMutating = true;
    _error = null;
    _notify();
    var success = false;
    try {
      final endpoint = id != null
          ? '/notifications/${Uri.encodeComponent(id)}${action == 'read' ? '/read' : ''}'
          : '/notifications${action == 'read-all' ? '/read-all' : ''}';
      final response = action == 'delete'
          ? await _apiService.delete<Map<String, dynamic>>(endpoint)
          : await _apiService.put<Map<String, dynamic>>(endpoint);
      if (!_valid(session)) return false;
      if (!response.success) {
        _error = response.error ?? 'Could not update notifications.';
        return false;
      }
      if (action == 'read-all') {
        _notifications = _notifications
            .map((item) => item.copyWith(isRead: true))
            .toList();
        _unreadCount = 0;
      } else if (id == null) {
        _notifications = [];
        _unreadCount = 0;
        _cursor = null;
      } else {
        final index = _notifications.indexWhere((item) => item.id == id);
        if (index >= 0) {
          if (!_notifications[index].isRead && _unreadCount > 0) _unreadCount--;
          if (action == 'delete') {
            _notifications.removeAt(index);
          } else {
            _notifications[index] = _notifications[index].copyWith(
              isRead: true,
            );
          }
        }
      }
      success = true;
    } catch (_) {
      if (_valid(session))
        _error = 'Could not confirm the change. Refresh before retrying.';
    } finally {
      if (_valid(session)) {
        _isMutating = false;
        _notify();
      }
    }
    if (success && _valid(session)) await fetchNotifications();
    return success && _valid(session);
  }

  @override
  void dispose() {
    _disposed = true;
    _session++;
    _debounceTimer?.cancel();
    if (_notificationService?.onNotificationReceived == _onMessage) {
      _notificationService?.onNotificationReceived = null;
    }
    super.dispose();
  }
}
