import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import 'package:orix/core/providers/group_provider.dart';
import 'package:orix/core/widgets/pixel_ui.dart';

class MyGroupScreen extends StatefulWidget {
  const MyGroupScreen({super.key});
  @override
  State<MyGroupScreen> createState() => _MyGroupScreenState();
}

class _MyGroupScreenState extends State<MyGroupScreen> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) context.read<GroupProvider>().fetchActiveGroups();
    });
  }

  @override
  Widget build(BuildContext context) {
    final groups = context.watch<GroupProvider>();
    return OrixPage(
      title: 'Your groups',
      description: 'Pickup plans and conversations for your journeys.',
      bottom: const OrixNavigation(selected: 2),
      children: [
        if (groups.isLoading) const PixelLoader(),
        if (groups.error != null) Text(groups.error!),
        if (!groups.isLoading &&
            groups.activeGroups.isEmpty &&
            groups.error == null)
          PixelSection(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const PixelAnimation(
                  scene: PixelScene.noMatch,
                  label: 'No active group',
                ),
                const SizedBox(height: 20),
                Text(
                  'No group yet.',
                  style: Theme.of(context).textTheme.headlineSmall,
                ),
                const SizedBox(height: 12),
                const Text('Your group will appear here after you match.'),
                const SizedBox(height: 20),
                FilledButton(
                  onPressed: () => context.go('/ride/create'),
                  child: const Text('Plan a journey'),
                ),
              ],
            ),
          ),
        for (final group in groups.activeGroups)
          PixelSection(
            title: group.routeSummary,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(
                  '${group.members.length} people · ${group.status.replaceAll('_', ' ')}',
                ),
                const SizedBox(height: 20),
                Wrap(
                  spacing: 12,
                  runSpacing: 12,
                  children: [
                    FilledButton(
                      onPressed: () => context.push(
                        '/groups/${Uri.encodeComponent(group.groupId)}',
                      ),
                      child: const Text('Open journey'),
                    ),
                    OutlinedButton(
                      onPressed: () => context.push(
                        '/groups/${Uri.encodeComponent(group.groupId)}/chat',
                      ),
                      child: const Text('Open chat'),
                    ),
                  ],
                ),
              ],
            ),
          ),
        TextButton(
          onPressed: groups.isLoading ? null : () => groups.fetchActiveGroups(),
          child: const Text('Refresh groups'),
        ),
      ],
    );
  }
}
