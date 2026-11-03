import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:go_router/go_router.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:orix/core/providers/auth_provider.dart';
import 'package:orix/core/widgets/pixel_ui.dart';

class BannedScreen extends StatelessWidget {
  const BannedScreen({super.key});
  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthProvider>();
    final reason = auth.user?.banReason;
    return OrixPage(
      title: 'Account restricted',
      description:
          'This account can no longer join journeys. You can ask for a review.',
      maxWidth: 600,
      children: [
        const PixelAnimation(
          scene: PixelScene.cancelled,
          label: 'Account access restricted',
        ),
        const SizedBox(height: 24),
        if (reason != null && reason.isNotEmpty)
          PixelSection(title: 'Reason', child: Text(reason)),
        FilledButton(
          onPressed: () => launchUrl(
            Uri(
              scheme: 'mailto',
              path: 'support@orix.app',
              queryParameters: {'subject': 'Account review request'},
            ),
          ),
          child: const Text('Request a review'),
        ),
        const SizedBox(height: 12),
        TextButton(
          onPressed: () async {
            await auth.signOut();
            if (context.mounted) context.go('/login');
          },
          child: const Text('Sign out'),
        ),
      ],
    );
  }
}
