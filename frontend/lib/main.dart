/// ORIX App - Main Entry Point
///
/// Public travel coordination platform for safe, affordable commuting.
library;

import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:go_router/go_router.dart';
import 'package:firebase_core/firebase_core.dart';
import 'package:firebase_auth/firebase_auth.dart' hide AuthProvider;
import 'core/config/api_config.dart';

import 'core/theme/app_theme.dart';
import 'core/router/app_router.dart';
import 'core/services/auth_service.dart';
import 'core/services/api_service.dart';
import 'core/services/notification_service.dart';
import 'core/services/websocket_service.dart';
import 'core/services/ad_service.dart';
import 'core/services/app_version_service.dart';
import 'core/providers/auth_provider.dart';
import 'core/providers/user_provider.dart';
import 'core/providers/ride_provider.dart';
import 'core/providers/group_provider.dart';
import 'core/providers/notification_provider.dart';
import 'core/providers/connectivity_provider.dart';
import 'features/groups/catch_your_ride_screen.dart';
import 'firebase_options.dart';

final adService = AdService();

void main() async {
  WidgetsFlutterBinding.ensureInitialized();
  ErrorWidget.builder = (_) => const Directionality(
    textDirection: TextDirection.ltr,
    child: ColoredBox(
      color: Color(0xFFF7F7F7),
      child: Center(
        child: Padding(
          padding: EdgeInsets.all(24),
          child: Text(
            'A small bump in the road.\nClose and reopen Orix to try again.',
            textAlign: TextAlign.center,
            style: TextStyle(
              fontFamily: 'Pixelify Sans',
              fontSize: 22,
              color: Color(0xFF222222),
              decoration: TextDecoration.none,
            ),
          ),
        ),
      ),
    ),
  );
  if (!ApiConfig.isConfigured) {
    runApp(const _ServiceConfigurationApp());
    return;
  }

  SystemChrome.setSystemUIOverlayStyle(
    const SystemUiOverlayStyle(
      statusBarColor: Colors.transparent,
      statusBarIconBrightness: Brightness.dark,
      systemNavigationBarColor: AppColors.background,
      systemNavigationBarIconBrightness: Brightness.dark,
    ),
  );

  // Initialize Firebase first (required for other services)
  if (InternalTestConfig.enabled) {
    if (!{
      'localhost',
      '127.0.0.1',
      '10.0.2.2',
      '::1',
    }.contains(InternalTestConfig.host)) {
      throw StateError(
        'Internal testing requires a local authentication emulator',
      );
    }
    await Firebase.initializeApp(
      name: 'orix-internal',
      options: const FirebaseOptions(
        apiKey: 'local-emulator-key',
        appId: '1:123456789:android:0000000000000000',
        messagingSenderId: '123456789',
        projectId: InternalTestConfig.projectId,
      ),
    );
    await FirebaseAuth.instanceFor(
      app: Firebase.app('orix-internal'),
    ).useAuthEmulator(InternalTestConfig.host, InternalTestConfig.port);
  } else {
    await Firebase.initializeApp(
      options: DefaultFirebaseOptions.currentPlatform,
    );
  }

  // Initialize services (instances created immediately)
  final authService = AuthService();
  final apiService = ApiService();
  final notificationService = NotificationService();
  final webSocketService = WebSocketService();
  final appVersionService = AppVersionService(apiService);

  // Run initializations in parallel for faster startup
  await Future.wait([
    authService.initialize(),
    if (!InternalTestConfig.enabled) notificationService.initialize(),
  ]);

  // Initialize AdMob lazily (don't block app launch)
  // adService.initialize(); // DISABLED: Crashing on Emulator

  // Set up FCM token refresh callback to sync with backend
  // Handled internally by UserProvider now

  runApp(
    OrixApp(
      authService: authService,
      apiService: apiService,
      notificationService: notificationService,
      webSocketService: webSocketService,
      appVersionService: appVersionService,
    ),
  );
}

/// Root application widget
class OrixApp extends StatefulWidget {
  final AuthService authService;
  final ApiService apiService;
  final NotificationService notificationService;
  final WebSocketService webSocketService;
  final AppVersionService appVersionService;

  const OrixApp({
    super.key,
    required this.authService,
    required this.apiService,
    required this.notificationService,
    required this.webSocketService,
    required this.appVersionService,
  });

  @override
  State<OrixApp> createState() => _OrixAppState();
}

class _OrixAppState extends State<OrixApp> with WidgetsBindingObserver {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    // Preload interstitial ad
    // adService.loadInterstitialAd(); // DISABLED: Crashing on Emulator
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    widget.webSocketService.dispose();
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) {
      widget.webSocketService.refreshConnection();
    }
  }

  @override
  Widget build(BuildContext context) {
    return MultiProvider(
      providers: [
        // Services
        Provider<AuthService>.value(value: widget.authService),
        Provider<ApiService>.value(value: widget.apiService),
        ChangeNotifierProvider<NotificationService>.value(
          value: widget.notificationService,
        ),
        Provider<WebSocketService>.value(value: widget.webSocketService),
        Provider<AppVersionService>.value(value: widget.appVersionService),
        // Connectivity monitoring
        ChangeNotifierProvider(create: (_) => ConnectivityProvider()),

        // State Providers
        ChangeNotifierProvider(
          create: (_) => AuthProvider(
            widget.authService,
            widget.apiService,
            widget.webSocketService,
            widget.notificationService,
          ),
        ),
        ChangeNotifierProxyProvider<AuthProvider, UserProvider>(
          create: (_) =>
              UserProvider(widget.apiService, widget.notificationService),
          update: (_, auth, user) => user!..updateAuth(auth),
        ),
        ChangeNotifierProxyProvider<AuthProvider, RideProvider>(
          create: (_) => RideProvider(widget.apiService),
          update: (_, auth, rides) => rides!
            ..updateUser(
              auth.isAuthenticated && auth.isOnboarded
                  ? auth.user?.userId
                  : null,
            ),
        ),
        ChangeNotifierProxyProvider<AuthProvider, GroupProvider>(
          create: (_) =>
              GroupProvider(widget.apiService, widget.webSocketService),
          update: (_, auth, group) => group!
            ..updateUser(
              auth.isAuthenticated && auth.isOnboarded
                  ? auth.user?.userId
                  : null,
            ),
        ),
        ChangeNotifierProxyProvider<AuthProvider, NotificationProvider>(
          create: (_) => NotificationProvider(
            widget.apiService,
            widget.notificationService,
          ),
          update: (_, auth, notifications) => notifications!
            ..updateUser(
              auth.isAuthenticated && auth.isOnboarded
                  ? auth.user?.userId
                  : null,
            ),
        ),
      ],
      child: Consumer<AuthProvider>(
        builder: (context, authProvider, _) {
          return _AlarmListener(
            child: MaterialApp.router(
              title: 'Orix',
              debugShowCheckedModeBanner: false,
              theme: AppTheme.lightTheme,
              darkTheme: AppTheme.darkTheme,
              themeMode: ThemeMode.light,
              routerConfig: AppRouter.router(authProvider),
            ),
          );
        },
      ),
    );
  }
}

/// Widget that listens for pending ride alarms and shows them
class _AlarmListener extends StatefulWidget {
  final Widget child;

  const _AlarmListener({required this.child});

  @override
  State<_AlarmListener> createState() => _AlarmListenerState();
}

class _AlarmListenerState extends State<_AlarmListener> {
  NotificationProvider? _notifications;
  NotificationService? _service;
  AuthProvider? _auth;
  bool _showingAlarm = false;
  WebSocketService? _socket;
  Timer? _syncTimer;
  bool _syncRides = false;
  static const _refreshEvents = [
    'connection',
    'readiness_update',
    'member_update',
    'resync',
    'group_finalized',
    'group_cancelled',
    'notification_update',
    'report_update',
    'ride_update',
    'support_update',
    'profile_update',
  ];
  void _queueRealtimeRefresh(Map<String, dynamic> event) {
    if (event['type'] == 'connection' && event['connected'] != true) return;
    if (event['type'] == 'profile_update') {
      unawaited(_auth?.refreshUser() ?? Future<void>.value());
    }
    if (event['type'] != 'notification_update' &&
        event['type'] != 'report_update')
      _syncRides = true;
    if (_syncTimer?.isActive == true) return;
    final owner = _auth?.user?.userId;
    _syncTimer = Timer(const Duration(milliseconds: 150), () {
      if (!mounted ||
          owner == null ||
          _auth?.user?.userId != owner ||
          _notifications?.isForUser(owner) != true)
        return;
      if (_syncRides) {
        _syncRides = false;
        unawaited(context.read<RideProvider>().fetchMatchingStatus());
        unawaited(
          context.read<RideProvider>().fetchRideHistory(includeHistory: true),
        );
      }
      unawaited(_notifications!.fetchNotifications());
    });
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _notifications?.removeListener(_transferPendingAlarm);
    _service?.removeListener(_transferPendingAlarm);
    _auth?.removeListener(_transferPendingAlarm);
    for (final type in _refreshEvents) {
      _socket?.off(type, _queueRealtimeRefresh);
    }
    _socket = context.read<WebSocketService>();
    for (final type in _refreshEvents) {
      _socket!.on(type, _queueRealtimeRefresh);
    }
    _notifications = context.read<NotificationProvider>();
    _service = context.read<NotificationService>();
    _auth = context.read<AuthProvider>();
    _notifications!.addListener(_transferPendingAlarm);
    _service!.addListener(_transferPendingAlarm);
    _auth!.addListener(_transferPendingAlarm);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) _transferPendingAlarm();
    });
  }

  void _transferPendingAlarm() {
    // A cold-start tap arrives before backend verification/onboarding. Keep it
    // queued until the account-scoped provider can validate the recipient.
    if (!mounted ||
        !(_auth?.isAuthenticated ?? false) ||
        !(_auth?.isOnboarded ?? false))
      return;
    if (_notifications?.isForUser(_auth?.user?.userId) != true) return;
    final pending = _service?.pendingAlarmData;
    if (pending != null) {
      _service!.clearPendingAlarmData();
      _notifications!.setPendingAlarmFromService(
        groupId: pending['group_id']!,
        rideTime: pending['ride_time'] ?? '',
        pickupSummary: pending['pickup_summary'] ?? '',
        recipientUserId: pending['recipient_user_id'],
      );
    }
    _handlePendingAlarm();
    _handlePendingConversation();
  }

  bool _openingConversation = false;
  void _handlePendingConversation() {
    final pending = _service?.pendingConversation;
    if (pending == null || _openingConversation || _showingAlarm) return;
    // Tokens and queued taps may outlive sign-out. Both identities must agree.
    if (pending['recipient_user_id'] != _auth?.user?.userId ||
        !PushRecipient.matches(
          pending,
          FirebaseAuth.instance.currentUser?.uid,
        )) {
      _service!.clearPendingConversation();
      return;
    }
    final account = _auth!.user!.userId;
    _openingConversation = true;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _openingConversation = false;
      if (!mounted ||
          _auth?.user?.userId != account ||
          !(_auth?.isAuthenticated ?? false) ||
          !(_auth?.isOnboarded ?? false) ||
          _service?.pendingConversation != pending ||
          _showingAlarm)
        return;
      final navigatorContext =
          AppRouter.rootNavigatorKey.currentState?.overlay?.context;
      if (navigatorContext == null) return;
      _service!.clearPendingConversation();
      final report = pending['report_id'];
      final String path;
      if (report != null) {
        path = '/profile/support/reports/${Uri.encodeComponent(report)}';
      } else if (pending['ticket_id'] != null) {
        path =
            '/profile/support/tickets/${Uri.encodeComponent(pending['ticket_id']!)}';
      } else if (pending['type'] == 'group_rejected') {
        path = '/ride/matching';
      } else {
        final chat = const {
          'new_chat_message',
          'chat_message',
          'new_message',
        }.contains(pending['type']);
        path =
            '/groups/${Uri.encodeComponent(pending['group_id']!)}${chat ? '/chat' : ''}';
      }
      GoRouter.of(navigatorContext).push(path);
    });
  }

  void _handlePendingAlarm() {
    if (!mounted ||
        _showingAlarm ||
        !(_auth?.isOnboarded ?? false) ||
        !(_auth?.isAuthenticated ?? false)) {
      return;
    }
    final groupId = _notifications?.pendingAlarmGroupId;
    if (groupId == null || groupId.isEmpty) return;
    final accountId = _auth?.user?.userId;
    final rideTime = _notifications!.pendingAlarmRideTime ?? '';
    final pickup = _notifications!.pendingAlarmPickupSummary ?? '';
    _showingAlarm = true;
    WidgetsBinding.instance.addPostFrameCallback((_) async {
      if (!mounted) return;
      if (_notifications?.isForUser(accountId) != true ||
          _auth?.user?.userId != accountId ||
          _notifications?.pendingAlarmGroupId != groupId ||
          _notifications?.pendingAlarmRideTime != rideTime) {
        _showingAlarm = false;
        _transferPendingAlarm();
        return;
      }
      final navigatorContext =
          AppRouter.rootNavigatorKey.currentState?.overlay?.context;
      if (navigatorContext == null) {
        _showingAlarm = false;
        return;
      }
      _notifications!.clearPendingAlarm();
      try {
        await showCatchYourRideAlarm(
          context: navigatorContext,
          groupId: groupId,
          rideTime: rideTime,
          pickupSummary: pickup,
        );
      } finally {
        _showingAlarm = false;
        if (mounted) _transferPendingAlarm();
      }
    });
  }

  @override
  void dispose() {
    _notifications?.removeListener(_transferPendingAlarm);
    _service?.removeListener(_transferPendingAlarm);
    _auth?.removeListener(_transferPendingAlarm);
    _syncTimer?.cancel();
    for (final type in _refreshEvents) {
      _socket?.off(type, _queueRealtimeRefresh);
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => widget.child;
}

/// A misconfigured release must not silently contact an old or insecure server.
class _ServiceConfigurationApp extends StatelessWidget {
  const _ServiceConfigurationApp();

  @override
  Widget build(BuildContext context) => MaterialApp(
    title: 'Orix',
    theme: AppTheme.lightTheme,
    home: const Scaffold(
      body: Center(
        child: Padding(
          padding: EdgeInsets.all(24),
          child: Text(
            'ORIX is unavailable in this build. Please use an updated app or contact support.',
            textAlign: TextAlign.center,
          ),
        ),
      ),
    ),
  );
}
