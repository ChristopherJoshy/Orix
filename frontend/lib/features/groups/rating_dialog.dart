import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:orix/core/services/api_service.dart';
import 'package:orix/core/widgets/pixel_ui.dart';
import 'package:orix/core/theme/app_theme.dart';

Future<void> showRatingDialog({
  required BuildContext context,
  required String groupId,
}) async {
  final api = context.read<ApiService>();
  final identity = api.authIdentity;
  if (identity == null) return;
  final response = await api.get<Map<String, dynamic>>(
    '/ratings/pending/${Uri.encodeComponent(groupId)}',
    forceRefresh: true,
  );
  if (!context.mounted || api.authIdentity != identity) return;
  if (!response.success) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(response.error ?? 'Could not load ratings.')),
    );
    return;
  }
  final people = (response.data?['pending_users'] as List<dynamic>? ?? [])
      .whereType<Map<String, dynamic>>()
      .where((p) => p['user_id'] is String)
      .toList();
  await showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    useSafeArea: true,
    builder: (_) =>
        _RatingSheet(groupId: groupId, people: people, identity: identity),
  );
}

class _RatingSheet extends StatefulWidget {
  final String groupId, identity;
  final List<Map<String, dynamic>> people;
  const _RatingSheet({
    required this.groupId,
    required this.people,
    required this.identity,
  });
  @override
  State<_RatingSheet> createState() => _RatingSheetState();
}

class _RatingSheetState extends State<_RatingSheet> {
  final Map<String, int> _stars = {};
  final Set<String> _saved = {};
  bool _busy = false;
  String? _error;
  Future<void> _save() async {
    if (_busy) return;
    final api = context.read<ApiService>();
    if (api.authIdentity != widget.identity) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      for (final entry in _stars.entries) {
        if (_saved.contains(entry.key)) continue;
        final response = await api.post<Map<String, dynamic>>(
          '/ratings',
          body: {
            'group_id': widget.groupId,
            'rated_user_id': entry.key,
            'rating': entry.value,
          },
        );
        if (!mounted || api.authIdentity != widget.identity) return;
        if (!response.success) {
          setState(
            () => _error =
                response.error ??
                'Some ratings were not saved. Retry to finish.',
          );
          return;
        }
        setState(() => _saved.add(entry.key));
      }
    } catch (_) {
      if (mounted)
        setState(
          () => _error =
              'Could not confirm all ratings. Saved ratings will not be sent again.',
        );
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final done = widget.people.isEmpty || _saved.length == widget.people.length;
    return ConstrainedBox(
      constraints: BoxConstraints(
        maxHeight: MediaQuery.sizeOf(context).height * .88,
      ),
      child: SingleChildScrollView(
        padding: EdgeInsets.fromLTRB(
          24,
          24,
          24,
          24 + MediaQuery.viewInsetsOf(context).bottom,
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          mainAxisSize: MainAxisSize.min,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    done ? 'All set.' : 'Rate your group',
                    style: Theme.of(context).textTheme.headlineSmall,
                  ),
                ),
                IconButton(
                  onPressed: _busy ? null : () => Navigator.pop(context),
                  tooltip: 'Close ratings',
                  icon: const Icon(Icons.close),
                ),
              ],
            ),
            if (done) ...[
              const PixelAnimation(
                scene: PixelScene.completed,
                label: 'Ratings complete',
              ),
              const SizedBox(height: 16),
              Text(
                widget.people.isEmpty
                    ? 'No ratings are waiting for this journey.'
                    : 'Your ratings are saved.',
              ),
              const SizedBox(height: 20),
              FilledButton(
                onPressed: () => Navigator.pop(context),
                child: const Text('Done'),
              ),
            ] else ...[
              const SizedBox(height: 12),
              const Text(
                'How was travelling together? You can skip and rate later.',
              ),
              const SizedBox(height: 24),
              for (final person in widget.people)
                Padding(
                  padding: const EdgeInsets.only(bottom: 24),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        person['display_name'] as String? ?? 'Travel partner',
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                      const SizedBox(height: 12),
                      Wrap(
                        spacing: 4,
                        children: List.generate(5, (index) {
                          final id = person['user_id'] as String;
                          final selected = (_stars[id] ?? 0) > index;
                          return IconButton(
                            tooltip:
                                '${index + 1} ${index == 0 ? 'star' : 'stars'}',
                            isSelected: selected,
                            onPressed: _busy || _saved.contains(id)
                                ? null
                                : () => setState(() => _stars[id] = index + 1),
                            icon: Icon(
                              selected ? Icons.star : Icons.star_outline,
                              color: AppColors.primary,
                            ),
                          );
                        }),
                      ),
                      if (_saved.contains(person['user_id']))
                        const Text(
                          'Saved',
                          style: TextStyle(
                            color: AppColors.textSecondary,
                            fontSize: 12,
                          ),
                        ),
                    ],
                  ),
                ),
              if (_error != null) Text(_error!),
              const SizedBox(height: 16),
              FilledButton(
                onPressed: _busy || _stars.isEmpty ? null : _save,
                child: _busy
                    ? const PixelButtonLoader()
                    : const Text('Save ratings'),
              ),
              TextButton(
                onPressed: _busy ? null : () => Navigator.pop(context),
                child: const Text('Later'),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
