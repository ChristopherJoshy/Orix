import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:orix/core/providers/auth_provider.dart';
import 'package:orix/core/widgets/pixel_ui.dart';

class SuspendedScreen extends StatelessWidget {
  const SuspendedScreen({super.key});
  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthProvider>();
    final expires = auth.user?.suspensionExpiresAt;
    return OrixPage(
      title: 'Account paused',
      description: 'This account is temporarily suspended from journeys.',
      maxWidth: 600,
      children: [
        const PixelAnimation(
          scene: PixelScene.noMatch,
          label: 'Account temporarily suspended',
        ),
        const SizedBox(height: 24),
        if (expires != null)
          PixelSection(
            title: 'Suspension ends',
            child: Text(
              DateFormat('EEE, d MMM yyyy · h:mm a').format(expires.toLocal()),
            ),
          ),
        FilledButton(
          onPressed: () async {
            await auth.refreshUser();
            if (context.mounted && !auth.isSuspended) context.go('/home');
          },
          child: const Text('Check account status'),
        ),
        const SizedBox(height: 12),
        TextButton(
          onPressed: () => launchUrl(
            Uri(
              scheme: 'mailto',
              path: 'support@orix.app',
              queryParameters: {'subject': 'Suspension review request'},
            ),
          ),
          child: const Text('Contact support'),
        ),
      ],
    );
  }
}
