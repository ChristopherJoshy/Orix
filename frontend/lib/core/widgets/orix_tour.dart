import 'package:flutter/material.dart';
import 'pixel_ui.dart';

/// Replayable introduction; it never creates a journey or requests permission.
Future<void> showOrixTour(BuildContext context) =>
    showDialog<void>(context: context, builder: (_) => const _Tour());

class _Tour extends StatefulWidget {
  const _Tour();
  @override
  State<_Tour> createState() => _TourState();
}

class _TourState extends State<_Tour> {
  int _step = 0;
  static const _titles = [
    'Set your route',
    'Meet your group',
    'Travel on your terms',
  ];
  static const _copy = [
    'Choose a starting point and destination from place suggestions. We look for compatible travellers starting within 300 metres and going to the same place.',
    'When a pool is ready, check the people, pickup plan and time. Use your group conversation to coordinate before leaving.',
    'Choose whether lifts are suitable and set your safety preferences. To offer a lift, register a separate route and time. You can block contacts by email or start a private report conversation.',
  ];
  @override
  Widget build(BuildContext context) => AlertDialog(
    scrollable: true,
    title: Text(_titles[_step]),
    content: SizedBox(
      width: 480,
      child: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            PixelAnimation(
              scene: [
                PixelScene.searching,
                PixelScene.found,
                PixelScene.liftOffered,
              ][_step],
              size: 160,
            ),
            const SizedBox(height: 20),
            Text(_copy[_step]),
            const SizedBox(height: 16),
            Text('${_step + 1} of ${_titles.length}'),
          ],
        ),
      ),
    ),
    actions: [
      if (_step > 0)
        TextButton(
          onPressed: () => setState(() => _step--),
          child: const Text('Back'),
        ),
      TextButton(
        onPressed: () => Navigator.pop(context),
        child: const Text('Close tour'),
      ),
      FilledButton(
        onPressed: () =>
            _step < 2 ? setState(() => _step++) : Navigator.pop(context),
        child: Text(_step < 2 ? 'Next' : 'Got it'),
      ),
    ],
  );
}
