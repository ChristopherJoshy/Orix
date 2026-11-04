/// Group Detail Screen
///
/// Shows group members, readiness confirmation, and navigation to chat.
library;

import 'package:orix/core/widgets/journey_route.dart';
import 'dart:async';
import 'package:flutter/material.dart';
import 'package:orix/core/widgets/pixel_ui.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
import 'package:provider/provider.dart';

import 'package:orix/core/providers/group_provider.dart';
import 'package:orix/core/services/api_service.dart';
import 'package:orix/core/providers/notification_provider.dart';
import 'package:orix/core/theme/app_theme.dart';
import '../profile/help_support/email_person_picker.dart';
import '../profile/help_support/support_service.dart';
import 'package:orix/core/widgets/trip_route_map.dart';
import 'rating_dialog.dart';
import '../profile/help_support/create_ticket_screen.dart';

class GroupDetailScreen extends StatefulWidget {
  final String groupId;

  const GroupDetailScreen({super.key, required this.groupId});

  @override
  State<GroupDetailScreen> createState() => _GroupDetailScreenState();
}

class _GroupDetailScreenState extends State<GroupDetailScreen> {
  Timer? _refreshTimer;
  bool _hasShownRatingDialog = false;

  @override
  void initState() {
    super.initState();
    _loadGroup();

    // Start periodic refresh every 10 seconds to check for status changes
    _refreshTimer = Timer.periodic(const Duration(seconds: 10), (_) {
      if (GoRouter.of(context).routeInformationProvider.value.uri.path ==
          '/groups/${widget.groupId}') {
        _loadGroup();
      }
    });
  }

  @override
  void dispose() {
    _refreshTimer?.cancel();
    super.dispose();
  }

  Future<void> _loadGroup() async {
    await context.read<GroupProvider>().fetchGroup(widget.groupId);

    if (!mounted) return;

    // Transient refresh errors stay on this screen with retry controls.
    // Completion comes only from the server lifecycle, never elapsed wall time.
    _checkAndShowRatingDialog();
  }

  void _checkAndShowRatingDialog() {
    final group = context.read<GroupProvider>().currentGroup;
    if (group == null || group.groupId != widget.groupId) return;

    // Show rating dialog only once when status becomes completed
    if (group.status == 'completed' && !_hasShownRatingDialog) {
      _hasShownRatingDialog = true;
      _refreshTimer?.cancel(); // Cancel timer to prevent further fetches

      Future.delayed(const Duration(milliseconds: 500), () {
        if (mounted) {
          showRatingDialog(context: context, groupId: group.groupId).then((_) {
            // After rating dialog is closed, go back to home
            if (mounted) {
              context.go('/home');
            }
          });
        }
      });
    }
  }

  Future<void> _confirmReadiness() async {
    // Prevent double-taps
    final provider = context.read<GroupProvider>();
    if (provider.isConfirming) return;

    final success = await provider.confirmReadiness(widget.groupId);

    if (!mounted) return;

    if (success) {
      // Refresh notifications to show confirmation notification
      context.read<NotificationProvider>().fetchNotifications();

      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('Readiness confirmed!')));
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Failed to confirm. Please try again.')),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<GroupProvider>();
    final group = provider.currentGroup?.groupId == widget.groupId
        ? provider.currentGroup
        : null;
    final scene = switch (group?.status) {
      'completed' => PixelScene.completed,
      'cancelled' => PixelScene.cancelled,
      'expired' => PixelScene.noMatch,
      'in_progress' => PixelScene.travelling,
      'pending' => PixelScene.searching,
      _ => PixelScene.found,
    };
    return OrixPage(
      title: 'Your journey',
      description: group?.status.replaceAll('_', ' '),
      maxWidth: 820,
      actions: [
        IconButton(
          onPressed: provider.isLoading ? null : _loadGroup,
          tooltip: 'Refresh journey',
          icon: const Icon(Icons.refresh),
        ),
        if (group != null)
          IconButton(
            onPressed: () => context.push(
              '/groups/${Uri.encodeComponent(widget.groupId)}/chat',
            ),
            tooltip: 'Group conversation',
            icon: const Icon(Icons.chat_bubble_outline),
          ),
      ],
      bottom: group == null ? null : _buildActions(group, provider),
      children: [
        if (provider.isLoading && group == null) const PixelLoader(),
        if (provider.error != null) Text(provider.error!),
        if (group == null && !provider.isLoading)
          PixelSection(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                const PixelAnimation(
                  scene: PixelScene.error,
                  label: 'Journey unavailable',
                ),
                const Text('This journey could not be loaded.'),
                TextButton(
                  onPressed: _loadGroup,
                  child: const Text('Try again'),
                ),
              ],
            ),
          )
        else if (group != null) ...[
          PixelAnimation(
            key: ValueKey(group.status),
            scene: scene,
            label: group.status.replaceAll('_', ' '),
          ),
          const SizedBox(height: 24),
          PixelSection(
            title: 'Pickup plan',
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  group.myPickupSummary ?? group.pickupSummary,
                  style: Theme.of(context).textTheme.titleMedium,
                ),
                const SizedBox(height: 12),
                if (group.myPickupTime ?? group.departureStart
                    case final planned?)
                  Text(
                    DateFormat('EEE, d MMM · h:mm a').format(planned.toLocal()),
                  ),
                if (group.confirmationDeadline != null &&
                    group.status == 'confirming')
                  Padding(
                    padding: const EdgeInsets.only(top: 12),
                    child: Text(
                      'Confirm by ${DateFormat('h:mm a').format(group.confirmationDeadline!.toLocal())}.',
                    ),
                  ),
                const SizedBox(height: 12),
                Text(
                  group.status == 'pending'
                      ? 'Matching is active. Pickups must be within 300 m and destinations within 100 m.'
                      : group.canConfirm
                      ? 'Confirm that this pickup works for you.'
                      : group.status == 'in_progress'
                      ? 'Finish your part of the journey after reaching your destination.'
                      : 'Open the group conversation to coordinate your pickup.',
                ),
              ],
            ),
          ),
          PixelSection(
            title: 'Route',
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                JourneyRoute(
                  pickup: group.pickupSummary,
                  destination: group.dropSummary,
                ),
                if (group.roadPlan != null) ...[
                  const SizedBox(height: 20),
                  TripRouteMap(roadPlan: group.roadPlan),
                ],
              ],
            ),
          ),
          PixelSection(
            title: 'Your group · ${group.members.length}',
            child: Column(
              children: [
                for (final member in group.members)
                  Padding(
                    padding: const EdgeInsets.only(bottom: 24),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        PersonIdentity(
                          person: UserSearchResult(
                            userId: member.userId,
                            displayName: member.displayName,
                            email: member.email ?? '',
                            photoUrl: member.photoUrl,
                          ),
                        ),
                        const SizedBox(height: 8),
                        Text(
                          member.isMe
                              ? 'You · ${member.readinessConfirmed ? 'Confirmed' : 'Not confirmed'}'
                              : member.isDriver
                              ? 'Lift driver'
                              : member.readinessConfirmed
                              ? 'Confirmed'
                              : 'Not confirmed',
                          style: const TextStyle(
                            color: AppColors.textSecondary,
                            fontSize: 12,
                          ),
                        ),
                        Text(
                          member.occupationLabel,
                          style: const TextStyle(
                            color: AppColors.textSecondary,
                          ),
                        ),
                        if (member.phone?.isNotEmpty == true)
                          Text(member.phone!),
                        if (!member.isMe)
                          Wrap(
                            spacing: 8,
                            runSpacing: 4,
                            children: [
                              TextButton.icon(
                                onPressed: () => Navigator.of(context).push(
                                  MaterialPageRoute<void>(
                                    builder: (_) => ReportUserScreen(
                                      groupId: group.groupId,
                                      userId: member.userId,
                                      displayName: member.displayName,
                                      email: member.email,
                                      photoUrl: member.photoUrl,
                                    ),
                                  ),
                                ),
                                icon: const Icon(Icons.flag_outlined, size: 16),
                                label: const Text('Report'),
                              ),
                              TextButton.icon(
                                onPressed: () => _blockMember(member),
                                icon: const Icon(Icons.block, size: 16),
                                label: const Text('Block contact'),
                              ),
                            ],
                          ),
                      ],
                    ),
                  ),
              ],
            ),
          ),
          if (group.status == 'completed')
            OutlinedButton(
              onPressed: () =>
                  showRatingDialog(context: context, groupId: group.groupId),
              child: const Text('Rate your group'),
            ),
        ],
      ],
    );
  }

  Future<void> _performAction(
    Future<bool> Function() action,
    String successMessage,
  ) async {
    final provider = context.read<GroupProvider>();
    final accepted = await action();
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(
          accepted
              ? successMessage
              : provider.error ?? 'Could not update this trip.',
        ),
      ),
    );
    if (accepted) {
      unawaited(context.read<NotificationProvider>().fetchNotifications());
    }
  }

  Future<void> _rejectGroup(RideGroup group) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        scrollable: true,
        title: const Text('Find another group?'),
        content: const Text(
          'Matching will restart for this journey. We will avoid this group and its members for the rest of this search. Everyone in this group will be notified and matched again.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Stay'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('Reject and search again'),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    final provider = context.read<GroupProvider>();
    final accepted = await provider.rejectGroup(group.groupId);
    if (!mounted) return;
    if (accepted) {
      context.go('/ride/matching');
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            provider.error ??
                'Matching restart was not confirmed. Refresh before retrying.',
          ),
        ),
      );
    }
  }

  Future<void> _leaveGroup(RideGroup group) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Leave this group?'),
        content: Text(
          group.isLift && group.me?.isDriver == true
              ? 'Leaving as the driver cancels this lift for everyone.'
              : 'You will leave this group and can plan a new trip.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Stay'),
          ),
          TextButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('Leave group'),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    final provider = context.read<GroupProvider>();
    final accepted = await provider.leaveGroup(group.groupId);
    if (!mounted) return;
    if (accepted) {
      context.go('/home');
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(provider.error ?? 'Could not leave this group.'),
        ),
      );
    }
  }

  Future<void> _finishRide(RideGroup group) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Have you reached your destination?'),
        content: const Text(
          'Finish your ride when your part of the trip is complete.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Not yet'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('Finish ride'),
          ),
        ],
      ),
    );
    if (confirmed == true && mounted) {
      await _performAction(
        () => context.read<GroupProvider>().finishRide(group.groupId),
        'Your ride is marked finished.',
      );
    }
  }

  Widget? _buildActions(RideGroup group, GroupProvider provider) {
    if (group.canConfirm) {
      return Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          _buildConfirmButton(provider.isChangingGroup),
          if (group.canReject)
            TextButton(
              onPressed: provider.isChangingGroup
                  ? null
                  : () => _rejectGroup(group),
              child: const Text('Reject and search again'),
            ),
          SafeArea(
            top: false,
            child: TextButton(
              onPressed: provider.isChangingGroup
                  ? null
                  : () => _leaveGroup(group),
              child: const Text('Leave group'),
            ),
          ),
        ],
      );
    }
    final busy = provider.isChangingGroup;
    final actions = <Widget>[
      if (group.canMarkHere)
        FilledButton.icon(
          onPressed: busy
              ? null
              : () => _performAction(
                  () => provider.markHere(group.groupId),
                  'Arrival recorded. Finish your ride after reaching your destination.',
                ),
          icon: const Icon(Icons.place_outlined),
          label: const Text('I am at my pickup point'),
        ),
      if (group.canStart)
        FilledButton.icon(
          onPressed: busy
              ? null
              : () => _performAction(
                  () => provider.startRide(group.groupId),
                  'Ride started.',
                ),
          icon: const Icon(Icons.play_arrow),
          label: const Text('Start ride'),
        ),
      if (group.canFinish)
        FilledButton.icon(
          onPressed: busy ? null : () => _finishRide(group),
          icon: const Icon(Icons.flag_outlined),
          label: const Text('Finish my ride'),
        ),
      if (group.canReject)
        TextButton(
          onPressed: busy ? null : () => _rejectGroup(group),
          child: const Text('Reject and search again'),
        ),
      if (group.canLeave)
        TextButton(
          onPressed: busy ? null : () => _leaveGroup(group),
          child: const Text('Leave group'),
        ),
    ];
    if (actions.isEmpty) return null;
    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(20, 8, 20, 12),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: actions,
        ),
      ),
    );
  }

  Future<void> _blockMember(GroupMember member) async {
    final api = context.read<ApiService>();
    final identity = api.authIdentity;
    final accepted = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Block this contact?'),
        content: Text(
          '${member.email ?? 'This contact'} will be excluded from future matches. Shared active groups are cancelled. You can unblock later.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('Block'),
          ),
        ],
      ),
    );
    if (accepted != true ||
        !mounted ||
        identity == null ||
        api.authIdentity != identity) {
      return;
    }
    try {
      final response = await api.put<Map<String, dynamic>>(
        '/users/${Uri.encodeComponent(member.userId)}/block',
      );
      if (!mounted || api.authIdentity != identity) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            response.success
                ? 'Traveler blocked.'
                : response.error ?? 'Could not block traveler.',
          ),
        ),
      );
      if (response.success) await _loadGroup();
    } catch (_) {
      if (mounted && api.authIdentity == identity) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text(
              'Could not confirm the block. Refresh and try again.',
            ),
          ),
        );
      }
    }
  }

  Widget _buildConfirmButton(bool busy) => SafeArea(
    top: false,
    child: Padding(
      padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
      child: FilledButton(
        onPressed: busy ? null : _confirmReadiness,
        child: busy ? const PixelLoader() : const Text('Confirm pickup'),
      ),
    ),
  );
}
