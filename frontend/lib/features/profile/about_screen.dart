import 'package:orix/core/widgets/orix_illustration.dart';
import 'package:flutter/material.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:orix/core/widgets/pixel_ui.dart';
import 'package:orix/core/theme/app_theme.dart';
import 'package:orix/core/config/privacy_config.dart';

class AboutScreen extends StatefulWidget {
  const AboutScreen({super.key});
  @override
  State<AboutScreen> createState() => _AboutScreenState();
}

class _AboutScreenState extends State<AboutScreen> {
  String _version = '';
  @override
  void initState() {
    super.initState();
    PackageInfo.fromPlatform().then((info) {
      if (mounted)
        setState(() => _version = '${info.version} (${info.buildNumber})');
    });
  }

  Future<void> _openProfile(String address) async {
    try {
      if (await launchUrl(
        Uri.parse(address),
        mode: LaunchMode.externalApplication,
      ))
        return;
    } catch (_) {
      /* Keep the failure actionable below. */
    }
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: const Text('Could not open this profile.'),
        action: SnackBarAction(
          label: 'Retry',
          onPressed: () => _openProfile(address),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) => OrixPage(
    title: 'About Orix',
    children: [
      PixelSection(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const OrixLogo(symbolOnly: true, width: 56),
            const SizedBox(height: 24),
            const Center(
              child: OrixIllustration(
                scene: OrixIllustrationScene.companions,
                height: 220,
              ),
            ),
            const SizedBox(height: 24),
            Text(
              'Your way. Our journey.',
              style: Theme.of(context).textTheme.headlineSmall,
            ),
            const SizedBox(height: 16),
            const Text(
              'Coordinate shared autos, travel companions and lifts with people heading the same way.',
            ),
            const SizedBox(height: 20),
            Text(
              'Version $_version',
              style: const TextStyle(color: AppColors.textSecondary),
            ),
          ],
        ),
      ),
      PixelSection(
        title: 'About the builder',
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              'Christopher Joshy',
              style: Theme.of(context).textTheme.titleLarge,
            ),
            const SizedBox(height: 12),
            const Text(
              'A full-stack, systems and mobile developer building practical software, from Flutter apps to real-time multiplayer systems. ORIX brings that interest in connecting people to everyday journeys.',
            ),
            const SizedBox(height: 24),
            Text(
              '“Life is like music. Its highs and lows are what make it beautiful.”',
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 16),
            Wrap(
              spacing: 16,
              children: [
                TextButton(
                  onPressed: () =>
                      _openProfile('https://github.com/ChristopherJoshy/'),
                  child: const Text('GitHub ↗'),
                ),
                TextButton(
                  onPressed: () => _openProfile(
                    'https://www.linkedin.com/in/christopher-joshy-272a77290/',
                  ),
                  child: const Text('LinkedIn ↗'),
                ),
              ],
            ),
          ],
        ),
      ),
      if (PrivacyConfig.policy != null)
        OutlinedButton(
          onPressed: PrivacyConfig.openPolicy,
          child: const Text('Privacy policy'),
        ),
      TextButton(
        onPressed: () => showLicensePage(
          context: context,
          applicationName: 'Orix',
          applicationVersion: _version,
        ),
        child: const Text('Open-source licenses'),
      ),
    ],
  );
}
