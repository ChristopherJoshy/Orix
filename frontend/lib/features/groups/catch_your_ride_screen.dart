/// An ordinary in-app reminder opened after checking current trip membership/time.
library;

import 'package:flutter/material.dart';
import 'package:orix/core/widgets/pixel_ui.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
import 'package:provider/provider.dart';

import 'package:orix/core/providers/group_provider.dart';
import 'package:orix/core/services/notification_service.dart';
import 'package:orix/core/utils/time_utils.dart';

Future<void> showCatchYourRideAlarm({
  required BuildContext context,
  required String groupId,
  required String rideTime,
  required String pickupSummary,
}) async {
  final groups = context.read<GroupProvider>();
  await groups.fetchGroup(groupId);
  if (!context.mounted) return;
  final group = groups.currentGroup;
  final planned = group?.myPickupTime ?? group?.departureStart;
  final incoming = TimeUtils.tryParseUtc(rideTime);
  final active =
      group?.status == 'active' ||
      (group?.status == 'in_progress' && group?.isLift == true);
  if (groups.error != null ||
      group?.groupId != groupId ||
      !active ||
      group?.me?.finished == true ||
      planned == null ||
      !planned.isAfter(DateTime.now()) ||
      (incoming != null &&
          incoming.difference(planned).abs() > const Duration(seconds: 2))) {
    ScaffoldMessenger.maybeOf(context)?.showSnackBar(
      const SnackBar(
        content: Text(
          'This reminder has changed or the trip could not be refreshed. Open your trip for current details.',
        ),
      ),
    );
    return;
  }
  // Use refreshed personal pickup data, not a stale notification's private label.
  await Navigator.of(context, rootNavigator: true).push<void>(
    MaterialPageRoute(
      fullscreenDialog: true,
      builder: (_) => _PickupReminder(
        groupId: groupId,
        pickupTime: planned,
        pickupSummary: group!.myPickupSummary ?? group.pickupSummary,
      ),
    ),
  );
}

class _PickupReminder extends StatefulWidget {
  final String groupId;
  final DateTime pickupTime;
  final String pickupSummary;
  const _PickupReminder({
    required this.groupId,
    required this.pickupTime,
    required this.pickupSummary,
  });

  @override
  State<_PickupReminder> createState() => _PickupReminderState();
}

class _PickupReminderState extends State<_PickupReminder> {
  bool _busy = false;
  String? _error;

  Future<void> _arrived() async {
    if (_busy) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    final provider = context.read<GroupProvider>();
    final accepted = await provider.markHere(widget.groupId);
    if (!mounted) return;
    if (!accepted) {
      setState(() {
        _busy = false;
        _error = provider.error ?? 'Could not record arrival.';
      });
      return;
    }
    await context.read<NotificationService>().dismissRideReminder(
      widget.groupId,
    );
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(
        content: Text(
          'Arrival recorded. Finish the ride after reaching your destination.',
        ),
      ),
    );
    _openGroup();
  }

  void _openGroup() {
    final router = GoRouter.of(context);
    Navigator.of(context).pop();
    router.push('/groups/${Uri.encodeComponent(widget.groupId)}');
  }

  @override
  Widget build(BuildContext context) => OrixPage(
    title: 'Your pickup',
    maxWidth: 640,
    children: [
      const Align(
        alignment: Alignment.centerLeft,
        child: PixelAnimation(
          scene: PixelScene.found,
          size: 160,
          label: 'Your pickup reminder',
        ),
      ),
      const SizedBox(height: 24),
      PixelSection(
        title: 'Meet your group',
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              DateFormat('h:mm a').format(widget.pickupTime),
              style: Theme.of(context).textTheme.headlineLarge,
            ),
            const SizedBox(height: 8),
            Text(DateFormat('EEEE, d MMMM').format(widget.pickupTime)),
            const Divider(height: 32),
            Text(widget.pickupSummary),
          ],
        ),
      ),
      if (_error case final error?) ...[
        Semantics(liveRegion: true, child: Text(error)),
        const SizedBox(height: 16),
      ],
      FilledButton(
        onPressed: _busy ? null : _arrived,
        child: _busy
            ? const SizedBox(height: 24, child: PixelLoader())
            : const Text('I am at my pickup point'),
      ),
      const SizedBox(height: 12),
      OutlinedButton(
        onPressed: _busy ? null : _openGroup,
        child: const Text('Open trip details'),
      ),
      const SizedBox(height: 12),
      TextButton(
        onPressed: _busy ? null : () => Navigator.of(context).pop(),
        child: const Text('Dismiss reminder'),
      ),
    ],
  );
}
