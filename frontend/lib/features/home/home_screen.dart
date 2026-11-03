/// Public travel dashboard backed by current matching/group state.
library;

import 'package:orix/core/widgets/journey_route.dart';
import 'dart:async';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import 'package:url_launcher/url_launcher.dart';

import 'package:orix/core/providers/auth_provider.dart';
import 'package:orix/core/providers/ride_provider.dart';
import 'package:orix/core/providers/group_provider.dart';
import 'package:orix/core/providers/notification_provider.dart';
import 'package:orix/core/providers/connectivity_provider.dart';
import 'package:orix/core/router/app_router.dart';
import 'package:orix/core/services/app_version_service.dart';
import 'package:orix/core/services/websocket_service.dart';
import '../ride/matching_screen.dart';
import 'package:orix/core/widgets/pixel_ui.dart';
import 'package:orix/core/theme/app_theme.dart';
import '../groups/rating_dialog.dart';

class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});
  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> with WidgetsBindingObserver {
  Timer? _poll;
  bool _refreshing = false;
  WebSocketService? _socket;
  static const _events = [
    'readiness_update',
    'group_updated',
    'group_finalized',
    'group_cancelled',
    'group_finish',
    'connection',
  ];
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _socket = context.read<WebSocketService>();
    for (final event in _events) {
      _socket!.on(event, _onEvent);
    }
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) {
        unawaited(_refresh());
        unawaited(_checkUpdates());
      }
    });
    _startPolling();
  }

  bool get _visible =>
      mounted &&
      GoRouter.of(context).routeInformationProvider.value.uri.path ==
          RoutePaths.home;
  void _startPolling() {
    _poll?.cancel();
    _poll = Timer.periodic(const Duration(seconds: 30), (_) {
      if (_visible) unawaited(_refresh());
    });
  }

  void _onEvent(Map<String, dynamic> _) {
    if (_visible) unawaited(_refresh(includeGroups: false));
  }

  Future<void> _refresh({bool includeGroups = true}) async {
    if (!mounted || _refreshing) return;
    _refreshing = true;
    final rides = context.read<RideProvider>();
    final groups = context.read<GroupProvider>();
    final notifications = context.read<NotificationProvider>();
    try {
      await Future.wait([
        rides.fetchMatchingStatus(),
        if (includeGroups) groups.fetchActiveGroups(),
        if (includeGroups) notifications.fetchNotifications(),
      ]);
      if (mounted && rides.matchingStatus?.canSearch == true) {
        await rides.fetchSuggestions();
      }
    } finally {
      _refreshing = false;
    }
  }

  Future<void> _checkUpdates() async {
    final service = context.read<AppVersionService>();
    final info = await service.shouldShowUpdatePopup();
    if (!mounted || info == null) return;
    await showDialog<void>(
      context: context,
      barrierDismissible: !info.updateRequired,
      builder: (context) => PopScope(
        canPop: !info.updateRequired,
        child: AlertDialog(
          title: Text(info.updateTitle ?? 'ORIX update available'),
          content: Text(
            info.updateNotes ??
                'Update ORIX to continue with the latest travel features.',
          ),
          actions: [
            if (!info.updateRequired)
              TextButton(
                onPressed: () {
                  unawaited(service.markUpdateDetailsSeen(info.currentVersion));
                  Navigator.pop(context);
                },
                child: const Text('Later'),
              ),
            if (info.storeUrl.isNotEmpty)
              FilledButton(
                onPressed: () => launchUrl(
                  Uri.parse(info.storeUrl),
                  mode: LaunchMode.externalApplication,
                ),
                child: const Text('Update'),
              )
            else
              const Text(
                'The update link is unavailable. Please contact support.',
              ),
          ],
        ),
      ),
    );
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) {
      _startPolling();
      if (_visible) unawaited(_refresh());
    } else if (state == AppLifecycleState.paused ||
        state == AppLifecycleState.inactive) {
      _poll?.cancel();
    }
  }

  @override
  void dispose() {
    _poll?.cancel();
    WidgetsBinding.instance.removeObserver(this);
    for (final event in _events) {
      _socket?.off(event, _onEvent);
    }
    super.dispose();
  }

  Future<void> _plan() async {
    final rides = context.read<RideProvider>();
    final groups = context.read<GroupProvider>();
    await Future.wait([
      rides.fetchMatchingStatus(),
      groups.fetchActiveGroups(),
    ]);
    if (!mounted) return;
    if (rides.hasActiveRequest || groups.hasActiveGroup) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text(
            'Open or leave your current trip before planning another.',
          ),
        ),
      );
      return;
    }
    await context.push(RoutePaths.createRide);
    if (mounted) await _refresh();
  }

  Future<void> _ratings() async {
    final ids = await context.read<GroupProvider>().checkPendingRatings();
    if (!mounted) return;
    if (ids.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('No trip partners to rate yet.')),
      );
    } else {
      await showRatingDialog(context: context, groupId: ids.first);
    }
  }

  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthProvider>();
    final rides = context.watch<RideProvider>();
    final groups = context.watch<GroupProvider>();
    final online = context.watch<ConnectivityProvider>().isOnline;
    final unread = context.watch<NotificationProvider>().unreadCount;
    final name = auth.user?.displayName.trim().split(RegExp(r'\s+')).first;
    return Scaffold(
      appBar: AppBar(
        title: const OrixLogo(width: 108),
        actions: [
          IconButton(
            tooltip: 'Notifications',
            onPressed: () => context.push(RoutePaths.notifications),
            icon: Badge(
              isLabelVisible: unread > 0,
              label: Text('$unread'),
              child: const Icon(Icons.notifications_outlined),
            ),
          ),
        ],
      ),
      bottomNavigationBar: const OrixNavigation(selected: 0),
      body: RefreshIndicator(
        onRefresh: _refresh,
        child: LayoutBuilder(
          builder: (context, constraints) => Align(
            alignment: Alignment.topCenter,
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 920),
              child: ListView(
                physics: const AlwaysScrollableScrollPhysics(),
                padding: const EdgeInsets.fromLTRB(20, 16, 20, 24),
                children: [
                  Text(
                    name == null || name.isEmpty ? 'Where to?' : 'Hey, $name.',
                    style: Theme.of(context).textTheme.headlineLarge,
                  ),
                  const SizedBox(height: 8),
                  const Text(
                    'Your route. Your people.',
                    style: TextStyle(color: AppColors.textSecondary),
                  ),
                  const SizedBox(height: 24),
                  if (!online)
                    const Padding(
                      padding: EdgeInsets.only(bottom: 24),
                      child: Text(
                        'You’re offline. Reconnect to update your journey.',
                      ),
                    ),
                  if (rides.hasActiveRequest)
                    const TripMatchingPanel()
                  else
                    PixelSection(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          const OrixLogo(symbolOnly: true, width: 36),
                          const SizedBox(height: 24),
                          Text(
                            'Go together.',
                            style: Theme.of(context).textTheme.headlineMedium,
                          ),
                          const SizedBox(height: 12),
                          const Text(
                            'Plan your route and meet people heading the same way.',
                          ),
                          const SizedBox(height: 24),
                          FilledButton.icon(
                            onPressed: !online || rides.isLoading
                                ? null
                                : _plan,
                            icon: const Icon(Icons.add, size: 18),
                            label: const Text('Plan a journey'),
                          ),
                        ],
                      ),
                    ),
                  if (rides.isLoading && rides.matchingStatus == null)
                    const PixelLoader(),
                  if (rides.error != null) Text(rides.error!),
                  if (groups.activeGroups.isNotEmpty)
                    PixelSection(
                      title: 'Your current group',
                      child: Column(
                        children: [
                          for (final group in groups.activeGroups)
                            Padding(
                              padding: const EdgeInsets.only(bottom: 8),
                              child: Column(
                                crossAxisAlignment: CrossAxisAlignment.stretch,
                                children: [
                                  JourneyRoute(
                                    pickup: group.pickupSummary,
                                    destination: group.dropSummary,
                                  ),
                                  const SizedBox(height: 8),
                                  Text(
                                    '${group.members.length} ${group.members.length == 1 ? 'person' : 'people'} · ${group.status.replaceAll('_', ' ')}',
                                    style: Theme.of(
                                      context,
                                    ).textTheme.bodySmall,
                                  ),
                                  const SizedBox(height: 12),
                                  OutlinedButton(
                                    onPressed: () => context.push(
                                      '/groups/${Uri.encodeComponent(group.groupId)}',
                                    ),
                                    child: const Text('Open group'),
                                  ),
                                ],
                              ),
                            ),
                        ],
                      ),
                    ),
                  PixelSection(
                    title: 'On your way',
                    child: Column(
                      children: [
                        ListTile(
                          contentPadding: EdgeInsets.zero,
                          leading: const Icon(Icons.directions_car_outlined),
                          title: const Text('Offer a lift'),
                          subtitle: const Text('Share a seat on your route'),
                          trailing: const Icon(Icons.chevron_right),
                          onTap: online
                              ? () => context.push(RoutePaths.riderSetup)
                              : null,
                        ),
                        const Divider(),
                        ListTile(
                          contentPadding: EdgeInsets.zero,
                          leading: const Icon(Icons.shield_outlined),
                          title: const Text('Safety and support'),
                          trailing: const Icon(Icons.chevron_right),
                          onTap: () => context.push(RoutePaths.helpSupport),
                        ),
                      ],
                    ),
                  ),
                  TextButton.icon(
                    onPressed: _ratings,
                    icon: const Icon(Icons.star_outline),
                    label: const Text('Rate a completed journey'),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
