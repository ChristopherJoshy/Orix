/// Group Provider
///
/// Manages ride group state, members, and chat.
/// Optimized to minimize unnecessary rebuilds.
library;

import 'dart:async';
import 'dart:convert';
import 'dart:math';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter/scheduler.dart' show SchedulerPhase;

import '../services/api_service.dart';
import '../services/websocket_service.dart';
import '../utils/time_utils.dart';

/// Group member
class GroupMember {
  final String occupation;
  String get occupationLabel => switch (occupation) {
    'student' => 'Student',
    'teacher' => 'Teacher',
    'other' => 'Other',
    _ => 'Not provided',
  };
  final String userId;
  final String displayName;
  final String? email;
  final String? phone;
  final String role;
  final bool readinessConfirmed;
  final bool isMe;
  final double? averageRating;
  final String? photoUrl;
  final String? gender;
  final bool isHere;
  final bool finished;

  bool get isDriver => role == 'rider' || role == 'driver';

  GroupMember({
    required this.userId,
    required this.displayName,
    this.occupation = '',
    this.email,
    this.phone,
    required this.role,
    required this.readinessConfirmed,
    required this.isMe,
    this.averageRating,
    this.photoUrl,
    this.gender,
    this.isHere = false,
    this.finished = false,
  });

  factory GroupMember.fromJson(Map<String, dynamic> json) {
    return GroupMember(
      userId: json['user_id'] as String,
      displayName: json['display_name'] as String,
      occupation: json['occupation'] as String? ?? '',
      email: json['email'] as String?,
      phone: json['phone'] as String?,
      role: json['role'] as String,
      readinessConfirmed: json['readiness_confirmed'] as bool? ?? false,
      isMe: json['is_me'] as bool? ?? false,
      averageRating: (json['average_rating'] as num?)?.toDouble(),
      photoUrl: json['photo_url'] as String?,
      gender: json['gender'] as String?,
      isHere: json['is_here'] as bool? ?? false,
      finished: json['finished'] as bool? ?? false,
    );
  }
}

class RouteCoordinate {
  final double latitude;
  final double longitude;

  const RouteCoordinate(this.latitude, this.longitude);

  factory RouteCoordinate.fromJson(List<dynamic> value) {
    if (value.length != 2 || value[0] is! num || value[1] is! num) {
      throw const FormatException('Invalid route coordinate');
    }
    final longitude = (value[0] as num).toDouble();
    final latitude = (value[1] as num).toDouble();
    if (!latitude.isFinite ||
        !longitude.isFinite ||
        latitude.abs() > 90 ||
        longitude.abs() > 180) {
      throw const FormatException('Invalid route coordinate');
    }
    return RouteCoordinate(latitude, longitude);
  }
}

class PlannedRoadStop {
  final String userId;
  final String kind;
  final RouteCoordinate point;
  final double? offsetSeconds;

  const PlannedRoadStop({
    required this.userId,
    required this.kind,
    required this.point,
    this.offsetSeconds,
  });

  factory PlannedRoadStop.fromJson(Map<String, dynamic> json) =>
      PlannedRoadStop(
        userId: json['user_id'] as String,
        kind: json['kind'] as String,
        point: RouteCoordinate.fromJson(json['coordinates'] as List<dynamic>),
        offsetSeconds: (json['offset_seconds'] as num?)?.toDouble(),
      );
}

class RoadItinerary {
  final double distanceMeters;
  final double durationSeconds;
  final double detourMeters;
  final List<RouteCoordinate> geometry;
  final List<PlannedRoadStop> stops;

  const RoadItinerary({
    required this.distanceMeters,
    required this.durationSeconds,
    required this.detourMeters,
    required this.geometry,
    required this.stops,
  });

  factory RoadItinerary.fromJson(Map<String, dynamic> json) => RoadItinerary(
    distanceMeters: (json['distance_meters'] as num).toDouble(),
    durationSeconds: (json['duration_seconds'] as num).toDouble(),
    detourMeters: (json['detour_meters'] as num).toDouble(),
    geometry: (json['geometry'] as List<dynamic>? ?? [])
        .map((point) => RouteCoordinate.fromJson(point as List<dynamic>))
        .toList(),
    stops: (json['stops'] as List<dynamic>? ?? [])
        .map((stop) => PlannedRoadStop.fromJson(stop as Map<String, dynamic>))
        .toList(),
  );
}

/// Ride group
class RideGroup {
  final String mode;
  final DateTime? departureStart;
  final DateTime? departureEnd;
  final DateTime? myPickupTime;
  final String? myPickupSummary;
  final RoadItinerary? roadPlan;
  final String groupId;
  final String routeSummary;
  final String pickupSummary;
  final String dropSummary;
  final String status;
  final String? date;
  final String? timeWindowStart;
  final DateTime? confirmationDeadline;
  final bool myReadinessConfirmed;
  final bool allConfirmed;
  final List<GroupMember> members;

  RideGroup({
    this.mode = 'auto',
    this.departureStart,
    this.departureEnd,
    this.myPickupTime,
    this.myPickupSummary,
    this.roadPlan,
    required this.groupId,
    required this.routeSummary,
    required this.pickupSummary,
    required this.dropSummary,
    required this.status,
    this.date,
    this.timeWindowStart,
    this.confirmationDeadline,
    required this.myReadinessConfirmed,
    required this.allConfirmed,
    required this.members,
  });

  bool get isLift => mode == 'lift';
  GroupMember? get me {
    for (final member in members) {
      if (member.isMe) return member;
    }
    return null;
  }

  bool get canConfirm =>
      members.length > 1 &&
      (status == 'pending' || status == 'confirming') &&
      !myReadinessConfirmed &&
      (confirmationDeadline == null ||
          confirmationDeadline!.isAfter(DateTime.now()));
  bool get canMarkHere =>
      (status == 'active' || status == 'in_progress') &&
      me != null &&
      !me!.isHere &&
      !me!.finished;
  bool get canStart {
    final now = DateTime.now();
    if (status != 'active' ||
        members.length < 2 ||
        !allConfirmed ||
        departureStart == null ||
        now.isBefore(departureStart!) ||
        (departureEnd != null && !now.isBefore(departureEnd!)))
      return false;
    if (isLift) return me?.isDriver == true && me?.isHere == true;
    return members.every((member) => member.isHere);
  }

  bool get canFinish => status == 'in_progress' && me != null && !me!.finished;
  bool get canReject =>
      members.length > 1 &&
      canLeave &&
      (departureEnd == null || departureEnd!.isAfter(DateTime.now()));
  bool get canLeave =>
      const {'pending', 'confirming', 'active'}.contains(status);
  bool get canChat =>
      const {'pending', 'confirming', 'active', 'in_progress'}.contains(status);

  factory RideGroup.fromJson(Map<String, dynamic> json) {
    String? timeStart;
    if (json['time_window'] != null) {
      timeStart =
          (json['time_window'] as Map<String, dynamic>)['start'] as String?;
    }

    return RideGroup(
      mode: json['mode'] as String? ?? 'auto',
      departureStart: TimeUtils.tryParseUtc(json['departure_start']),
      departureEnd: TimeUtils.tryParseUtc(json['departure_end']),
      myPickupTime: TimeUtils.tryParseUtc(json['my_pickup_time']),
      myPickupSummary: json['my_pickup_summary'] as String?,
      roadPlan: json['road_plan'] is Map<String, dynamic>
          ? RoadItinerary.fromJson(json['road_plan'] as Map<String, dynamic>)
          : null,
      groupId: json['group_id'] as String,
      routeSummary: json['route_summary'] as String,
      pickupSummary: json['pickup_summary'] as String,
      dropSummary: json['drop_summary'] as String,
      status: json['status'] as String,
      date: json['date'] as String?,
      timeWindowStart: timeStart,
      confirmationDeadline: json['confirmation_deadline'] != null
          ? TimeUtils.parseUtc(json['confirmation_deadline'] as String)
          : null,
      myReadinessConfirmed: json['my_readiness_confirmed'] as bool? ?? false,
      allConfirmed: json['all_confirmed'] as bool? ?? false,
      members:
          (json['members'] as List<dynamic>?)
              ?.map((m) => GroupMember.fromJson(m as Map<String, dynamic>))
              .toList() ??
          [],
    );
  }
}

/// Chat message
class ChatMessage {
  final String messageId;
  final String? userId;
  final String? userDisplayName;
  final String messageType;
  final String content;
  final DateTime createdAt;
  final bool isMine;
  final bool isPending;
  final bool failed;
  final String? clientMessageId;
  final String? groupId;

  ChatMessage({
    required this.messageId,
    this.userId,
    this.userDisplayName,
    required this.messageType,
    required this.content,
    required this.createdAt,
    required this.isMine,
    this.isPending = false,
    this.failed = false,
    this.clientMessageId,
    this.groupId,
  });

  factory ChatMessage.fromJson(Map<String, dynamic> json, {String? groupId}) {
    return ChatMessage(
      messageId: json['message_id'] as String,
      clientMessageId: json['client_message_id'] as String?,
      groupId: groupId,
      userId: json['user_id'] as String?,
      userDisplayName: json['user_display_name'] as String?,
      messageType: json['message_type'] as String,
      content: json['content'] as String,
      createdAt: TimeUtils.parseUtc(json['created_at'] as String),
      isMine: json['is_mine'] as bool? ?? false,
      isPending: false,
    );
  }

  /// Create a copy with different pending status
  ChatMessage copyWith({bool? isPending, bool? failed}) {
    return ChatMessage(
      messageId: messageId,
      clientMessageId: clientMessageId,
      groupId: groupId,
      failed: failed ?? this.failed,
      userId: userId,
      userDisplayName: userDisplayName,
      messageType: messageType,
      content: content,
      createdAt: createdAt,
      isMine: isMine,
      isPending: isPending ?? this.isPending,
    );
  }
}

/// Group state is scoped to the signed-in account and to one open chat.
class GroupProvider extends ChangeNotifier {
  final ApiService _apiService;
  final WebSocketService _webSocketService;
  final Random _random = Random.secure();
  String? _userId;
  String? _authIdentity;
  int _session = 0;
  int _detailRequest = 0;
  int _activeRequest = 0;
  int _chatVersion = 0;
  bool _disposed = false;
  bool _notificationScheduled = false;
  bool _isLoading = false;
  String? _error;
  String? _chatError;
  RideGroup? _currentGroup;
  List<RideGroup> _activeGroups = [];
  List<ChatMessage> _messages = [];
  String? _chatGroupId;
  String? _olderCursor;
  bool _hasOlderMessages = false;
  bool _chatReadBusy = false;
  bool _isSendingMessage = false;
  String? _pendingAction;
  Timer? _resyncTimer;

  static const _groupEvents = {
    'group_updated',
    'readiness_update',
    'member_update',
    'resync',
    'group_finalized',
    'group_cancelled',
    'group_confirm',
    'group_mark-here',
    'group_start',
    'group_finish',
    'group_leave',
  };

  GroupProvider(this._apiService, this._webSocketService) {
    _webSocketService.on('new_chat_message', _onChat);
    _webSocketService.on('connection', _onConnection);
    for (final event in _groupEvents) {
      _webSocketService.on(event, _onGroupEvent);
    }
  }

  bool get isLoading => _isLoading;
  String? get error => _error;
  String? get chatError => _chatError;
  RideGroup? get currentGroup => _currentGroup;
  List<ChatMessage> get messages => List.unmodifiable(_messages);
  List<RideGroup> get activeGroups => List.unmodifiable(_activeGroups);
  bool get hasActiveGroup =>
      _activeGroups.any((group) => group.me?.finished != true);
  bool get isRealtimeConnected => _webSocketService.isConnected;
  bool get isSendingMessage => _isSendingMessage;
  bool get isConfirming => _pendingAction == 'confirm';
  bool get isChangingGroup => _pendingAction != null;
  bool get isLoadingMessages => _chatReadBusy;
  bool get canLoadOlderMessages => _olderCursor != null;

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

  bool _currentSession(int session) =>
      !_disposed &&
      session == _session &&
      _authIdentity == _apiService.authIdentity;

  /// Clear immediately on account changes; stale requests cannot repopulate data
  /// after a sign-out or when switching among internal testing accounts.
  void updateUser(String? userId) {
    if (userId == _userId) return;
    _session++;
    _userId = userId;
    _authIdentity = userId == null ? null : _apiService.authIdentity;
    _currentGroup = null;
    _activeGroups = [];
    _pendingAction = null;
    _isLoading = false;
    _error = null;
    _resetChat();
    _resyncTimer?.cancel();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _notify();
      if (!_disposed && _userId != null) unawaited(fetchActiveGroups());
    });
  }

  void _resetChat([String? groupId]) {
    _chatVersion++;
    _chatGroupId = groupId;
    _messages = [];
    _olderCursor = null;
    _hasOlderMessages = false;
    _chatReadBusy = false;
    _isSendingMessage = false;
    _chatError = null;
  }

  Future<void> fetchActiveGroups() async {
    if (_userId == null) return;
    final session = _session;
    final request = ++_activeRequest;
    try {
      final response = await _apiService.get<List<dynamic>>(
        '/groups',
        forceRefresh: true,
      );
      if (!_currentSession(session) || request != _activeRequest) return;
      if (response.success && response.data != null) {
        _activeGroups = response.data!
            .map((data) => RideGroup.fromJson(data as Map<String, dynamic>))
            .toList();
      } else {
        _error = response.error;
      }
    } catch (_) {
      if (_currentSession(session)) _error = 'Could not refresh your groups.';
    }
    if (_currentSession(session)) _notify();
  }

  Future<void> fetchGroup(String groupId) async {
    if (_userId == null) return;
    final session = _session;
    final request = ++_detailRequest;
    _isLoading = _currentGroup?.groupId != groupId;
    _error = null;
    if (_isLoading) _notify();
    try {
      final response = await _apiService.get<Map<String, dynamic>>(
        '/groups/${Uri.encodeComponent(groupId)}',
        forceRefresh: true,
      );
      if (!_currentSession(session) || request != _detailRequest) return;
      if (response.success && response.data != null) {
        _currentGroup = RideGroup.fromJson(response.data!);
      } else {
        _error = response.error;
        if (response.statusCode == 403 || response.statusCode == 404) {
          _currentGroup = null;
          if (_chatGroupId == groupId) _resetChat();
        }
      }
    } catch (_) {
      if (_currentSession(session) && request == _detailRequest) {
        _error = 'Could not refresh this group. Please try again.';
      }
    } finally {
      if (_currentSession(session) && request == _detailRequest) {
        _isLoading = false;
        _notify();
      }
    }
  }

  Future<bool> confirmReadiness(String id) => _changeGroup(id, 'confirm');
  Future<bool> markHere(String id) => _changeGroup(id, 'mark-here');
  Future<bool> startRide(String id) => _changeGroup(id, 'start');
  Future<bool> finishRide(String id) => _changeGroup(id, 'finish');
  Future<bool> leaveGroup(String id) => _changeGroup(id, 'leave');
  Future<bool> rejectGroup(String id) => _changeGroup(id, 'reject');

  Future<bool> _changeGroup(String id, String action) async {
    if (_userId == null || isChangingGroup) return false;
    final session = _session;
    _pendingAction = action;
    _error = null;
    _notify();
    try {
      final response = await _apiService.post<Map<String, dynamic>>(
        '/groups/${Uri.encodeComponent(id)}/$action',
        body: action == 'confirm' ? {'confirmed': true} : null,
      );
      if (!_currentSession(session)) return false;
      if (!response.success) {
        final reason = response.error;
        await fetchGroup(id);
        if (_currentSession(session)) _error = reason;
        return false;
      }
      if (action == 'leave' || action == 'reject') {
        _currentGroup = null;
        if (_chatGroupId == id) _resetChat();
      } else {
        await fetchGroup(id);
      }
      await fetchActiveGroups();
      return _currentSession(session);
    } catch (_) {
      if (_currentSession(session))
        _error = 'Could not confirm this action. Refresh and try again.';
      return false;
    } finally {
      if (_currentSession(session)) {
        _pendingAction = null;
        _notify();
      }
    }
  }

  Future<void> fetchMessages(String groupId, {bool resetHistory = false}) =>
      _readMessages(groupId, resetHistory: resetHistory);

  Future<void> loadOlderMessages(String groupId) async {
    if (_chatGroupId != groupId || _olderCursor == null) return;
    await _readMessages(groupId, before: _olderCursor);
  }

  Future<void> _readMessages(
    String groupId, {
    String? before,
    bool resetHistory = false,
  }) async {
    if (_userId == null) return;
    if (_chatGroupId != groupId || resetHistory) _resetChat(groupId);
    if (_chatReadBusy) return;
    final session = _session;
    final chat = _chatVersion;
    // A REST snapshot may finish after a newer socket delivery or send ack.
    // Preserve those arrivals when replacing the latest history window.
    final knownIds = _messages.map((message) => message.messageId).toSet();
    _chatReadBusy = true;
    _notify();
    try {
      final query = Uri(
        queryParameters: {'limit': '50', if (before != null) 'before': before},
      ).query;
      final response = await _apiService.get<List<dynamic>>(
        '/groups/${Uri.encodeComponent(groupId)}/chat?$query',
        forceRefresh: true,
      );
      if (!_currentSession(session) || chat != _chatVersion) return;
      if (!response.success || response.data == null) {
        _chatError = response.error;
        if (response.statusCode == 403 || response.statusCode == 404)
          _messages = [];
        return;
      }
      final incoming = response.data!
          .map(
            (data) => ChatMessage.fromJson(
              data as Map<String, dynamic>,
              groupId: groupId,
            ),
          )
          .toList();
      _mergeMessages([
        ...incoming,
        ..._messages.where(
          (message) =>
              !message.isPending &&
              !message.failed &&
              !knownIds.contains(message.messageId),
        ),
      ], keepHistory: before != null || _hasOlderMessages);
      if (before != null) _hasOlderMessages = true;
      if (before != null || !_hasOlderMessages)
        _olderCursor = response.nextCursor;
      _chatError = null;
    } catch (_) {
      if (_currentSession(session) && chat == _chatVersion)
        _chatError = 'Could not refresh messages.';
    } finally {
      if (_currentSession(session) && chat == _chatVersion) {
        _chatReadBusy = false;
        _notify();
      }
    }
  }

  void _mergeMessages(List<ChatMessage> incoming, {bool keepHistory = true}) {
    final confirmed = <String, ChatMessage>{};
    if (keepHistory) {
      for (final message in _messages) {
        if (!message.isPending && !message.failed)
          confirmed[message.messageId] = message;
      }
    }
    for (final message in incoming) {
      confirmed[message.messageId] = message;
    }
    final acknowledged = confirmed.values
        .where((m) => m.isMine)
        .map((m) => m.clientMessageId)
        .whereType<String>()
        .toSet();
    final local = _messages.where(
      (m) =>
          (m.isPending || m.failed) &&
          !acknowledged.contains(m.clientMessageId),
    );
    _messages = [...confirmed.values, ...local]
      ..sort((a, b) {
        final order = a.createdAt.compareTo(b.createdAt);
        return order != 0 ? order : a.messageId.compareTo(b.messageId);
      });
  }

  void addMessageFromWebSocket(Map<String, dynamic> data) {
    final id = data['group_id'];
    if (id != _chatGroupId) return;
    final body = data['message'];
    if (body is! Map<String, dynamic>) return;
    _mergeMessages([ChatMessage.fromJson(body, groupId: id as String)]);
    _notify();
  }

  String _newClientId() => base64UrlEncode(
    List<int>.generate(18, (_) => _random.nextInt(256)),
  ).replaceAll('=', '');

  Future<bool> sendMessage(String groupId, String content) async {
    final text = content.trim();
    if (_userId == null || _isSendingMessage || text.isEmpty) return false;
    if (text.runes.length > 1000) {
      _chatError = 'Messages can contain up to 1000 characters.';
      _notify();
      return false;
    }
    if (_chatGroupId != groupId) _resetChat(groupId);
    final clientId = _newClientId();
    _messages = [
      ..._messages,
      ChatMessage(
        messageId: 'local-$clientId',
        clientMessageId: clientId,
        groupId: groupId,
        userId: _userId,
        messageType: 'user',
        content: text,
        createdAt: DateTime.now(),
        isMine: true,
        isPending: true,
      ),
    ];
    return _postMessage(groupId, clientId, text);
  }

  Future<bool> retryMessage(String groupId, String clientId) async {
    if (_chatGroupId != groupId || _isSendingMessage) return false;
    for (final message in _messages) {
      if (message.failed && message.clientMessageId == clientId) {
        return _postMessage(groupId, clientId, message.content);
      }
    }
    return false;
  }

  Future<bool> _postMessage(
    String groupId,
    String clientId,
    String text,
  ) async {
    final session = _session;
    final chat = _chatVersion;
    _isSendingMessage = true;
    _chatError = null;
    _messages = _messages
        .map(
          (m) => m.clientMessageId == clientId
              ? m.copyWith(isPending: true, failed: false)
              : m,
        )
        .toList();
    _notify();
    try {
      // Prefer the realtime path, waiting for the persisted server ack. If it
      // becomes uncertain, REST retries the same ID rather than creating a duplicate.
      if (_webSocketService.isConnected) {
        try {
          final message = await _webSocketService.sendChat(
            groupId: groupId,
            content: text,
            clientMessageId: clientId,
          );
          if (!_currentSession(session) || chat != _chatVersion) return false;
          _mergeMessages([ChatMessage.fromJson(message, groupId: groupId)]);
          return true;
        } catch (_) {
          if (!_currentSession(session) || chat != _chatVersion) return false;
        }
      }
      if (!_currentSession(session) || chat != _chatVersion) return false;
      final response = await _apiService.post<Map<String, dynamic>>(
        '/groups/${Uri.encodeComponent(groupId)}/chat',
        body: {'content': text, 'client_message_id': clientId},
      );
      if (!_currentSession(session) || chat != _chatVersion) return false;
      if (response.success && response.data != null) {
        _mergeMessages([
          ChatMessage.fromJson(response.data!, groupId: groupId),
        ]);
        return true;
      }
      _chatError =
          response.error ?? 'Message was not confirmed. Retry to check it.';
    } catch (_) {
      if (_currentSession(session) && chat == _chatVersion)
        _chatError = 'Message was not confirmed. Retry to check it.';
    } finally {
      if (_currentSession(session) && chat == _chatVersion) {
        _isSendingMessage = false;
        _messages = _messages
            .map(
              (m) => m.clientMessageId == clientId && m.isPending
                  ? m.copyWith(isPending: false, failed: true)
                  : m,
            )
            .toList();
        _notify();
      }
    }
    return false;
  }

  void _onChat(Map<String, dynamic> data) => addMessageFromWebSocket(data);
  void _onConnection(Map<String, dynamic> data) {
    _notify();
    if (data['connected'] == true) _scheduleResync(resetChat: true);
  }

  void _onGroupEvent(Map<String, dynamic> data) {
    if (data['type'] == 'group_cancelled' && data['group_id'] == _chatGroupId)
      _resetChat(_chatGroupId);
    _scheduleResync();
  }

  void _scheduleResync({bool resetChat = false}) {
    if (_userId == null || _disposed) return;
    if (resetChat && _chatGroupId != null) {
      // Preserve unacknowledged IDs/drafts across reconnect so a retry cannot
      // turn a lost acknowledgement into a second message.
      _messages = _messages
          .where((message) => message.isPending || message.failed)
          .toList();
      _olderCursor = null;
      _hasOlderMessages = false;
    }
    _resyncTimer?.cancel();
    _resyncTimer = Timer(const Duration(milliseconds: 150), () {
      if (_disposed || _userId == null) return;
      unawaited(fetchActiveGroups());
      final group = _currentGroup?.groupId;
      if (group != null) unawaited(fetchGroup(group));
      final chat = _chatGroupId;
      if (chat != null) unawaited(fetchMessages(chat));
    });
  }

  void clearGroup() {
    _detailRequest++;
    _currentGroup = null;
    _isLoading = false;
    _resetChat();
    _notify();
  }

  Future<List<String>> checkPendingRatings() async {
    if (_userId == null) return [];
    final session = _session;
    try {
      final response = await _apiService.get<List<dynamic>>(
        '/ratings/pending',
        forceRefresh: true,
      );
      if (_currentSession(session) &&
          response.success &&
          response.data != null) {
        return response.data!.map((id) => id.toString()).toList();
      }
    } catch (_) {
      /* Keep the trip screen usable when ratings cannot refresh. */
    }
    return [];
  }

  @override
  void dispose() {
    _disposed = true;
    _session++;
    _resyncTimer?.cancel();
    _webSocketService.off('new_chat_message', _onChat);
    _webSocketService.off('connection', _onConnection);
    for (final event in _groupEvents) {
      _webSocketService.off(event, _onGroupEvent);
    }
    super.dispose();
  }
}
