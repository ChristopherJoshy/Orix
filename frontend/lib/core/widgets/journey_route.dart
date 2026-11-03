import 'package:flutter/material.dart';
import '../theme/app_theme.dart';

/// Address previews stay readable; the original labels are always expandable.
class JourneyRoute extends StatefulWidget {
  final String pickup, destination;
  const JourneyRoute({
    super.key,
    required this.pickup,
    required this.destination,
  });

  @override
  State<JourneyRoute> createState() => _JourneyRouteState();
}

class _JourneyRouteState extends State<JourneyRoute> {
  bool _expanded = false;

  @override
  void didUpdateWidget(covariant JourneyRoute oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.pickup != widget.pickup ||
        oldWidget.destination != widget.destination) {
      _expanded = false;
    }
  }

  @override
  Widget build(BuildContext context) {
    final canExpand =
        widget.pickup.length > 48 || widget.destination.length > 48;
    Widget stop(String label, String address, bool destination) => Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.only(top: 5),
          child: Container(
            width: 10,
            height: 10,
            decoration: BoxDecoration(
              color: destination ? AppColors.primary : AppColors.surface,
              border: Border.all(color: AppColors.primary, width: 2),
            ),
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                label,
                style: Theme.of(
                  context,
                ).textTheme.bodySmall?.copyWith(color: AppColors.textSecondary),
              ),
              const SizedBox(height: 2),
              Text(
                address,
                maxLines: _expanded ? null : 2,
                overflow: _expanded ? null : TextOverflow.ellipsis,
                style: Theme.of(context).textTheme.titleSmall,
              ),
            ],
          ),
        ),
      ],
    );
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        stop('From', widget.pickup, false),
        Padding(
          padding: const EdgeInsets.symmetric(vertical: 6, horizontal: 4),
          child: Align(
            alignment: Alignment.centerLeft,
            child: Container(width: 2, height: 12, color: AppColors.outline),
          ),
        ),
        stop('To', widget.destination, true),
        if (canExpand)
          Align(
            alignment: Alignment.centerLeft,
            child: TextButton.icon(
              style: TextButton.styleFrom(
                padding: const EdgeInsets.symmetric(horizontal: 0, vertical: 8),
              ),
              onPressed: () => setState(() => _expanded = !_expanded),
              icon: Icon(
                _expanded ? Icons.expand_less : Icons.expand_more,
                size: 18,
              ),
              label: Text(_expanded ? 'Less detail' : 'Full addresses'),
            ),
          ),
      ],
    );
  }
}
