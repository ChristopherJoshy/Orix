/// Own-route matching and compatible-group selection for every travel mode.
library;

import 'dart:async';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
import 'package:provider/provider.dart';

import 'package:orix/core/providers/ride_provider.dart';
import 'package:orix/core/providers/group_provider.dart';
import 'package:orix/core/router/app_router.dart';
import 'package:orix/core/services/websocket_service.dart';
import 'package:orix/core/widgets/trip_route_map.dart';
import 'package:orix/core/widgets/pixel_ui.dart';

class MatchingScreen extends StatefulWidget {
  const MatchingScreen({super.key});
  @override
  State<MatchingScreen> createState() => _MatchingScreenState();
}

class _MatchingScreenState extends State<MatchingScreen>
    with WidgetsBindingObserver {
  Timer? _poll;
  WebSocketService? _socket;
  bool _refreshing = false;
  String? _redirected;
  static const _events = [
    'readiness_update',
    'group_updated',
    'group_finalized',
    'group_cancelled',
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
      if (mounted) _refresh();
    });
    _startPolling();
  }

  bool get _visible =>
      mounted &&
      GoRouter.of(context).routeInformationProvider.value.uri.path ==
          RoutePaths.matching;
  void _startPolling() {
    _poll?.cancel();
    _poll = Timer.periodic(const Duration(seconds: 15), (_) {
      if (_visible) _refresh();
    });
  }

  void _onEvent(Map<String, dynamic> _) {
    if (_visible) unawaited(_refresh());
  }

  Future<void> _refresh() async {
    if (_refreshing || !mounted) return;
    _refreshing = true;
    final provider = context.read<RideProvider>();
    try {
      await provider.fetchMatchingStatus();
      if (!mounted) return;
      final status = provider.matchingStatus;
      if (status?.groupId != null &&
          status!.groupSize > 1 &&
          const {
            'confirming',
            'active',
            'in_progress',
          }.contains(status.groupStatus)) {
        if (_redirected != status.groupId) {
          _redirected = status.groupId;
          context.go('/groups/${Uri.encodeComponent(status.groupId!)}');
        }
        return;
      }
      if (status?.canSearch == true) await provider.fetchSuggestions();
    } finally {
      _refreshing = false;
    }
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

  @override
  Widget build(BuildContext context) => OrixPage(
    title: 'Your pool',
    description: 'Your route, time and preferences are being checked together.',
    actions: [
      IconButton(
        tooltip: 'Refresh matching status',
        onPressed: _refresh,
        icon: const Icon(Icons.refresh),
      ),
    ],
    children: const [TripMatchingPanel()],
  );
}

class TripMatchingPanel extends StatelessWidget {
  const TripMatchingPanel({super.key});
  Future<void> _cancel(BuildContext context, MatchingStatus status) async {
    final accepted = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Cancel this trip request?'),
        content: const Text('You can create a new trip afterward.'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Keep searching'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('Cancel request'),
          ),
        ],
      ),
    );
    if (accepted != true || !context.mounted || status.requestId == null)
      return;
    final rides = context.read<RideProvider>();
    bool success;
    if (status.groupId != null) {
      success = await context.read<GroupProvider>().leaveGroup(status.groupId!);
      if (success) await rides.fetchMatchingStatus();
    } else {
      success = await rides.cancelRideRequest(status.requestId!);
    }
    if (!context.mounted) return;
    if (success) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Row(
            children: [
              PixelAnimation(
                scene: PixelScene.cancelled,
                size: 80,
                color: Colors.white,
                label: 'Journey cancelled',
              ),
              SizedBox(width: 12),
              Expanded(child: Text('Journey request cancelled.')),
            ],
          ),
        ),
      );
      context.go(RoutePaths.home);
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(rides.error ?? 'Could not cancel. Refresh the trip.'),
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) => Consumer<RideProvider>(
    builder: (context, rides, _) {
      final current = rides.matchingStatus;
      final request = current?.request;
      if (current == null && rides.isLoading)
        return const Center(child: PixelLoader());
      if (current?.hasActiveRequest != true)
        return PixelSection(
          title: 'No active request',
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const PixelAnimation(
                scene: PixelScene.noMatch,
                label: 'No active journey request',
              ),
              if (rides.error != null) Text(rides.error!),
              const SizedBox(height: 16),
              FilledButton(
                onPressed: () => context.go(RoutePaths.createRide),
                child: const Text('Plan a journey'),
              ),
            ],
          ),
        );
      final status = current!;
      final waiting = status.canSearch;
      return Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          PixelSection(
            title: waiting ? 'Finding your people.' : 'Your pool is ready.',
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                PixelAnimation(
                  key: ValueKey(waiting),
                  scene: waiting ? PixelScene.searching : PixelScene.found,
                  label: waiting ? 'Finding a compatible pool' : 'Pool found',
                ),
                const SizedBox(height: 20),
                Text(
                  '${status.pickupLabel ?? 'Starting point'} → ${status.dropLabel ?? 'Destination'}',
                  style: Theme.of(context).textTheme.titleLarge,
                ),
                const SizedBox(height: 12),
                if (request?.departureStart != null &&
                    request?.departureEnd != null)
                  Text(
                    '${DateFormat('EEE, d MMM · h:mm a').format(request!.departureStart!)} – ${DateFormat('h:mm a').format(request.departureEnd!)}',
                  ),
                const SizedBox(height: 16),
                Text(
                  waiting
                      ? 'No compatible pool yet. Matching continues while your request is active.'
                      : 'Open your group to confirm the pickup and coordinate your journey.',
                ),
                if (rides.error != null)
                  Padding(
                    padding: const EdgeInsets.only(top: 16),
                    child: Text(rides.error!),
                  ),
                const SizedBox(height: 20),
                if (status.groupId != null && !waiting)
                  FilledButton(
                    onPressed: () => context.push(
                      '/groups/${Uri.encodeComponent(status.groupId!)}',
                    ),
                    child: const Text('Open journey'),
                  ),
                if (status.canEdit)
                  OutlinedButton(
                    onPressed: rides.isMutating
                        ? null
                        : () => context.push(
                            RoutePaths.createRide,
                            extra: request,
                          ),
                    child: const Text('Edit request'),
                  ),
                if (waiting)
                  TextButton(
                    onPressed: rides.isMutating
                        ? null
                        : () => _cancel(context, status),
                    child: const Text('Cancel request'),
                  ),
              ],
            ),
          ),
          if (waiting)
            const PixelSection(
              title: 'A good fit means',
              child: Text(
                'Pickups within 300 m. The same destination. Compatible departure windows and safety preferences. Lift offers are included only when you opt in.',
              ),
            ),
          if (request != null)
            TripRouteMap(
              pickupLat: request.pickupLat,
              pickupLng: request.pickupLng,
              dropLat: request.dropLat,
              dropLng: request.dropLng,
            ),
        ],
      );
    },
  );
}
