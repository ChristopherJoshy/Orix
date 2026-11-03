import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:go_router/go_router.dart';
import 'package:orix/core/widgets/pixel_ui.dart';
import 'package:orix/core/providers/ride_provider.dart';
import 'package:orix/core/theme/app_theme.dart';

class RideHistoryScreen extends StatefulWidget {
  const RideHistoryScreen({super.key});
  @override
  State<RideHistoryScreen> createState() => _RideHistoryScreenState();
}

class _RideHistoryScreenState extends State<RideHistoryScreen> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted)
        context.read<RideProvider>().fetchRideHistory(includeHistory: true);
    });
  }

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<RideProvider>();
    return OrixPage(
      title: 'Your journeys',
      description: 'Current requests and previous journeys.',
      bottom: const OrixNavigation(selected: 1),
      children: [
        if (provider.isLoading) const PixelLoader(),
        if (provider.error != null) Text(provider.error!),
        if (provider.rideHistory.isEmpty &&
            !provider.isLoading &&
            provider.error == null)
          PixelSection(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const PixelAnimation(
                  scene: PixelScene.noMatch,
                  label: 'No journeys',
                ),
                Text(
                  'No journeys yet.',
                  style: Theme.of(context).textTheme.headlineSmall,
                ),
                const SizedBox(height: 20),
                FilledButton(
                  onPressed: () => context.go('/ride/create'),
                  child: const Text('Plan a journey'),
                ),
              ],
            ),
          ),
        for (final ride in provider.rideHistory)
          PixelSection(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(
                  ride.status.replaceAll('_', ' '),
                  style: const TextStyle(
                    color: AppColors.textSecondary,
                    fontSize: 12,
                  ),
                ),
                const SizedBox(height: 12),
                Text(
                  '${ride.pickupLabel} → ${ride.dropLabel}',
                  style: Theme.of(context).textTheme.titleLarge,
                ),
                const SizedBox(height: 12),
                Text(
                  '${ride.date} · ${ride.timeWindow.start} – ${ride.timeWindow.end}',
                ),
                if (ride.groupId != null) ...[
                  const SizedBox(height: 20),
                  OutlinedButton(
                    onPressed: () => context.push(
                      '/groups/${Uri.encodeComponent(ride.groupId!)}',
                    ),
                    child: const Text('Open journey'),
                  ),
                ] else if (const {
                  'pending',
                  'matching',
                }.contains(ride.status)) ...[
                  const SizedBox(height: 20),
                  OutlinedButton(
                    onPressed: () => context.push('/ride/matching'),
                    child: const Text('Matching status'),
                  ),
                ],
              ],
            ),
          ),
        if (provider.hasMoreHistory)
          TextButton(
            onPressed: provider.isLoading
                ? null
                : () => provider.fetchRideHistory(
                    includeHistory: true,
                    loadMore: true,
                  ),
            child: const Text('Earlier journeys'),
          ),
        TextButton(
          onPressed: provider.isLoading
              ? null
              : () => provider.fetchRideHistory(includeHistory: true),
          child: const Text('Refresh journeys'),
        ),
      ],
    );
  }
}
