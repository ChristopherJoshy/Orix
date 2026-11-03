import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import '../../core/widgets/pixel_ui.dart';

class RiderSetupScreen extends StatelessWidget {
  const RiderSetupScreen({super.key});
  @override
  Widget build(BuildContext context) => OrixPage(
    title: 'Offer a lift',
    description: 'Share a seat on a route you’re already travelling.',
    maxWidth: 640,
    children: [
      const PixelAnimation(
        scene: PixelScene.liftOffered,
        label: 'Offer a lift on your route',
      ),
      const SizedBox(height: 24),
      PixelSection(
        title: 'Register your trip',
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const Text(
              'Set the route, pickup window and available passenger seats. Check the pickup plan with your group before leaving.',
            ),
            const SizedBox(height: 24),
            FilledButton(
              onPressed: () =>
                  context.push('/ride/create?mode=lift&offer=true'),
              child: const Text('Register a lift route'),
            ),
          ],
        ),
      ),
    ],
  );
}
