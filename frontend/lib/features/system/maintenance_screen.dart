import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:orix/core/widgets/pixel_ui.dart';

class MaintenanceScreen extends StatelessWidget {
  const MaintenanceScreen({super.key});
  @override
  Widget build(BuildContext context) => OrixPage(
    title: 'Back soon.',
    description:
        'Orix is temporarily unavailable. Your existing plans have not been changed by this screen.',
    maxWidth: 600,
    children: [
      const PixelAnimation(
        scene: PixelScene.error,
        label: 'Service temporarily unavailable',
      ),
      const SizedBox(height: 28),
      FilledButton(
        onPressed: () => context.go('/'),
        child: const Text('Check again'),
      ),
    ],
  );
}
