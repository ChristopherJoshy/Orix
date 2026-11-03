import 'dart:convert';
import 'dart:ui' show Color;

import 'package:firebase_core/firebase_core.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_local_notifications/flutter_local_notifications.dart';

import '../../firebase_options.dart';

/// Push identity is separate from the installation token: tokens can outlive a
/// sign-in. New Go notifications always carry the intended Firebase identity.
class PushRecipient {
  static bool matches(Map<String, dynamic> data, String? currentUid) =>
      currentUid != null &&
      currentUid.isNotEmpty &&
      data['recipient_firebase_uid'] == currentUid;
}

/// Conversation taps retain identifiers only, never report content or a URL.
class ConversationNotificationPayload {
  static Map<String, String>? fromData(Map<String, dynamic> data) {
    final report = data['report_id'];
    final ticket = data['ticket_id'];
    final group = data['group_id'];
    final recipient = data['recipient_firebase_uid'];
    final user = data['recipient_user_id'];
    if (recipient is! String ||
        recipient.isEmpty ||
        user is! String ||
        user.isEmpty)
      return null;
    final String field;
    final String id;
    if (report is String && report.isNotEmpty && report.length <= 100) {
      field = 'report_id';
      id = report;
    } else if (ticket is String && ticket.isNotEmpty && ticket.length <= 100) {
      field = 'ticket_id';
      id = ticket;
    } else if (group is String && group.isNotEmpty && group.length <= 100) {
      field = 'group_id';
      id = group;
    } else {
      return null;
    }
    return {
      field: id,
      if (data['type'] is String) 'type': data['type'] as String,
      'recipient_firebase_uid': recipient,
      'recipient_user_id': user,
    };
  }
}

/// One payload format for foreground delivery, background delivery and taps.
class RideAlarmPayload {
  static bool isAlarm(Map<String, dynamic> data) {
    final type = data['type'];
    return (type is String &&
            const {'ride_alarm', 'ride_start'}.contains(type.toLowerCase())) ||
        data['action'] == 'show_alarm';
  }

  static Map<String, String>? fromData(Map<String, dynamic> data) {
    if (!isAlarm(data)) return null;
    final groupId = data['group_id'];
    if (groupId is! String || groupId.trim().isEmpty) return null;
    return {
      'group_id': groupId,
      if (data['recipient_firebase_uid'] is String)
        'recipient_firebase_uid': data['recipient_firebase_uid'] as String,
      if (data['recipient_user_id'] is String)
        'recipient_user_id': data['recipient_user_id'] as String,
      'ride_time': data['ride_time'] is String
          ? data['ride_time'] as String
          : '',
      'pickup_summary': data['pickup_summary'] is String
          ? data['pickup_summary'] as String
          : '',
    };
  }

  static Map<String, String>? decode(String? payload) {
    if (payload == null || payload.isEmpty) return null;
    try {
      final data = jsonDecode(payload);
      if (data is Map<String, dynamic>) return fromData(data);
    } on FormatException {
      // Notifications already delivered by older versions use pipes.
      if (payload.startsWith('show_alarm|')) {
        final parts = payload.split('|');
        if (parts.length >= 4) {
          return fromData({
            'action': 'show_alarm',
            'group_id': parts[1],
            'ride_time': parts[2],
            'pickup_summary': parts.sublist(3).join('|'),
          });
        }
      }
    }
    return null;
  }

  static String encode(Map<String, dynamic> data) =>
      jsonEncode({...?fromData(data), 'type': 'ride_alarm'});
}

const _localInitialization = InitializationSettings(
  android: AndroidInitializationSettings('@drawable/ic_notification'),
  iOS: DarwinInitializationSettings(
    requestAlertPermission: false,
    requestBadgePermission: false,
    requestSoundPermission: false,
  ),
);

int _reminderNotificationId(String groupId) {
  var hash = 17;
  for (final unit in groupId.codeUnits) {
    hash = (hash * 31 + unit) & 0x7fffffff;
  }
  return hash;
}

Future<void> _showRideReminder(
  FlutterLocalNotificationsPlugin plugin,
  Map<String, dynamic> data,
) async {
  final payload = RideAlarmPayload.fromData(data);
  if (payload == null) return;
  final when = DateTime.tryParse(payload['ride_time'] ?? '');
  if (when != null && !when.isAfter(DateTime.now())) return;
  final minutes = when == null
      ? null
      : (when.difference(DateTime.now()).inMilliseconds /
                Duration.millisecondsPerMinute)
            .ceil();
  try {
    await plugin.show(
      _reminderNotificationId(payload['group_id']!),
      minutes == null
          ? 'Ride reminder'
          : 'Your pickup is in $minutes ${minutes == 1 ? 'minute' : 'minutes'}',
      'Tap for your current pickup details.',
      const NotificationDetails(
        android: AndroidNotificationDetails(
          'ride_reminders_v2',
          'Ride reminders',
          channelDescription: 'Reminders when your coordinated ride starts',
          icon: '@drawable/ic_notification',
          color: const Color(0xFF222222),
          colorized: false,
          importance: Importance.high,
          priority: Priority.high,
          visibility: NotificationVisibility.private,
          category: AndroidNotificationCategory.reminder,
          onlyAlertOnce: true,
        ),
        iOS: DarwinNotificationDetails(presentAlert: true, presentSound: true),
      ),
      payload: RideAlarmPayload.encode(data),
    );
  } catch (_) {
    if (kDebugMode) debugPrint('Ride reminder could not be displayed.');
  }
}

@pragma('vm:entry-point')
Future<void> _firebaseBackgroundHandler(RemoteMessage message) async {
  await Firebase.initializeApp(options: DefaultFirebaseOptions.currentPlatform);
  if (!RideAlarmPayload.isAlarm(message.data)) return;
  // The background isolate must load persisted auth before deciding who owns
  // this push. Never display a prior account's reminder after sign-out/switch.
  final user = await FirebaseAuth.instance.authStateChanges().first.timeout(
    const Duration(seconds: 3),
    onTimeout: () => null,
  );
  if (!PushRecipient.matches(message.data, user?.uid)) return;
  if (defaultTargetPlatform == TargetPlatform.iOS &&
      message.notification != null)
    return;
  final plugin = FlutterLocalNotificationsPlugin();
  await plugin.initialize(_localInitialization);
  await _showRideReminder(plugin, message.data);
}

/// Notification service
class NotificationService extends ChangeNotifier {
  FirebaseMessaging get _messaging => FirebaseMessaging.instance;
  final FlutterLocalNotificationsPlugin _localNotifications =
      FlutterLocalNotificationsPlugin();

  Future<void> dismissRideReminder(String groupId) async {
    await _localNotifications.cancel(_reminderNotificationId(groupId));
    await _localNotifications.cancel(
      999,
    ); // Previously delivered legacy reminder.
  }

  /// FCM token
  String? _fcmToken;
  String? get fcmToken => _fcmToken;

  /// Callback for when token changes (to sync with backend)
  void Function(String token)? onTokenRefresh;

  /// Callback for when a notification is received in foreground
  void Function(RemoteMessage message)? onNotificationReceived;

  /// Pending alarm data from notification tap
  Map<String, String>? _pendingAlarmData;
  Map<String, String>? get pendingAlarmData => _pendingAlarmData;

  void _queueAlarm(Map<String, String>? data) {
    if (data == null) return;
    _pendingAlarmData = data;
    notifyListeners();
  }

  void clearPendingAlarmData() {
    _pendingAlarmData = null;
  }

  Map<String, String>? _pendingConversation;
  Map<String, String>? get pendingConversation => _pendingConversation;
  void clearPendingConversation() => _pendingConversation = null;

  void _queueNotificationTap(Map<String, dynamic> data) {
    if (RideAlarmPayload.isAlarm(data)) {
      _queueAlarm(RideAlarmPayload.fromData(data));
      return;
    }
    final conversation = ConversationNotificationPayload.fromData(data);
    if (conversation == null) return;
    _pendingConversation = conversation;
    notifyListeners();
  }

  void _localNotificationTap(String? payload) {
    if (payload == null) return;
    try {
      final data = jsonDecode(payload);
      if (data is Map<String, dynamic>) {
        _queueNotificationTap(data);
        return;
      }
    } on FormatException {
      // Preserve alarm taps from older app versions.
      _queueAlarm(RideAlarmPayload.decode(payload));
    }
  }

  /// Stream of incoming messages when app is in foreground
  Stream<RemoteMessage> get onMessage => FirebaseMessaging.onMessage;

  /// Stream for when user taps notification (app was in background)
  Stream<RemoteMessage> get onMessageOpenedApp =>
      FirebaseMessaging.onMessageOpenedApp;

  /// Initialization flag
  bool _isInitialized = false;

  /// Android Notification Channel for High Importance
  static const AndroidNotificationChannel _androidChannel =
      AndroidNotificationChannel(
        'high_importance_channel', // id
        'High Importance Notifications', // title
        description:
            'This channel is used for important notifications.', // description
        importance: Importance.max,
        playSound: true,
      );

  /// Dismissible high-priority pickup reminders; no repeating alarm or fullscreen.
  static const AndroidNotificationChannel _alarmChannel =
      AndroidNotificationChannel(
        'ride_reminders_v2', // id
        'Ride reminders', // title
        description:
            'Urgent ride notifications that require immediate attention.',
        importance: Importance.max,
        playSound: true,
        enableVibration: true,
        enableLights: true,
      );

  /// Initialize notifications
  Future<void> initialize() async {
    if (_isInitialized) return;

    FirebaseMessaging.onBackgroundMessage(_firebaseBackgroundHandler);
    await _localNotifications.initialize(
      _localInitialization,
      onDidReceiveNotificationResponse: (details) =>
          _localNotificationTap(details.payload),
    );
    final launch = await _localNotifications.getNotificationAppLaunchDetails();
    if (launch?.didNotificationLaunchApp ?? false) {
      _localNotificationTap(launch?.notificationResponse?.payload);
    }
    FirebaseMessaging.onMessageOpenedApp.listen((message) {
      _queueNotificationTap(message.data);
    });
    final initial = await _messaging.getInitialMessage();
    if (initial != null) _queueNotificationTap(initial.data);
    _isInitialized = true;

    // Create Android channels
    final resolve = _localNotifications
        .resolvePlatformSpecificImplementation<
          AndroidFlutterLocalNotificationsPlugin
        >();
    if (resolve != null) {
      await resolve.createNotificationChannel(_androidChannel);
      await resolve.createNotificationChannel(_alarmChannel);
    }

    FirebaseMessaging.onMessage.listen(_handleForegroundMessage);
    final settings = await _messaging.getNotificationSettings();
    if (settings.authorizationStatus == AuthorizationStatus.authorized ||
        settings.authorizationStatus == AuthorizationStatus.provisional) {
      await _enablePush();
    }
  }

  bool _pushEnabled = false;

  /// Called only after the user chooses notifications during setup/settings.
  Future<bool> requestPermission() async {
    final settings = await _messaging.requestPermission(
      alert: true,
      badge: true,
      sound: true,
      provisional: false,
    );
    final granted =
        settings.authorizationStatus == AuthorizationStatus.authorized ||
        settings.authorizationStatus == AuthorizationStatus.provisional;
    if (granted) await _enablePush();
    return granted;
  }

  Future<void> _enablePush() async {
    if (_pushEnabled) return;
    _fcmToken = await _messaging.getToken();
    _pushEnabled = true;
    _messaging.onTokenRefresh.listen((token) {
      _fcmToken = token;
      onTokenRefresh?.call(token);
    });
    await _messaging.setForegroundNotificationPresentationOptions(
      alert: true,
      badge: true,
      sound: true,
    );
    if (_fcmToken != null) onTokenRefresh?.call(_fcmToken!);
    try {
      await _messaging.unsubscribeFromTopic('promotions');
    } catch (_) {
      /* Retry on next launch. */
    }
  }

  /// Processed message IDs to prevent duplicates
  final Set<String> _processedMessageIds = {};

  void _handleForegroundMessage(RemoteMessage message) {
    if (!PushRecipient.matches(
      message.data,
      FirebaseAuth.instance.currentUser?.uid,
    ))
      return;
    final suppliedId = message.data['notification_id'];
    final dedupeId = suppliedId is String && suppliedId.isNotEmpty
        ? suppliedId
        : message.messageId;
    if (dedupeId != null) {
      if (_processedMessageIds.contains(dedupeId)) {
        return;
      }
      _processedMessageIds.add(dedupeId);

      if (_processedMessageIds.length > 50) {
        _processedMessageIds.remove(_processedMessageIds.first);
      }
    }

    RemoteNotification? notification = message.notification;
    AndroidNotification? android = message.notification?.android;

    if (RideAlarmPayload.isAlarm(message.data)) {
      if (defaultTargetPlatform != TargetPlatform.iOS ||
          message.notification == null) {
        _showRideReminder(_localNotifications, message.data);
      }
      onNotificationReceived?.call(message);
      return;
    }

    // Show local notification using the appropriate channel
    if (notification != null && android != null) {
      _localNotifications.show(
        notification.hashCode,
        notification.title,
        notification.body,
        NotificationDetails(
          android: AndroidNotificationDetails(
            _androidChannel.id,
            _androidChannel.name,
            channelDescription: _androidChannel.description,
            icon: '@drawable/ic_notification',
            color: const Color(0xFF222222),
            colorized: false,
            visibility: NotificationVisibility.private,
            priority: Priority.high,
            importance: Importance.max,
            playSound: true,
          ),
        ),
        payload: jsonEncode({
          ...?ConversationNotificationPayload.fromData(message.data),
          if (message.data['type'] is String) 'type': message.data['type'],
        }),
      );
    }

    // Notify callback for in-app handling
    onNotificationReceived?.call(message);
  }

  /// Check if app was opened from notification
  Future<RemoteMessage?> getInitialMessage() async {
    return await _messaging.getInitialMessage();
  }

  /// Route notification to appropriate screen based on type
  Future<void> handleNotificationTap(
    RemoteMessage message, {
    required Function(String route, Map<String, dynamic> args) navigateTo,
  }) async {
    final data = message.data;
    final type = data['type'] ?? '';

    switch (type) {
      case 'ride_match':
      case 'match_found':
        // Navigate to group/ride view when match is found
        navigateTo('/group', {'groupId': data['group_id']});
        break;
      case 'ride_alarm':
      case 'RIDE_ALARM':
        // Navigate to group view for ride alarm
        navigateTo('/group', {'groupId': data['group_id']});
        break;
      case 'chat_message':
      case 'new_message':
        // Navigate to chat screen
        navigateTo('/chat', {'groupId': data['group_id']});
        break;
      case 'rating_pending':
      case 'rating_request':
        // Navigate to ratings
        navigateTo('/ratings', {'groupId': data['group_id']});
        break;
      case 'user_match_update':
        // Generic group update
        navigateTo('/group', {'groupId': data['group_id']});
        break;
      default:
        // Default: go to home
        navigateTo('/', {});
    }
  }

  /// Subscribe to topic
  Future<void> subscribeToTopic(String topic) async {
    await _messaging.subscribeToTopic(topic);
  }

  /// Unsubscribe from topic
  Future<void> unsubscribeFromTopic(String topic) async {
    await _messaging.unsubscribeFromTopic(topic);
  }
}
