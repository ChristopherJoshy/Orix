import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';
import '../../core/widgets/pixel_ui.dart';
import '../../core/services/app_version_service.dart';

class UpdateDetailsScreen extends StatelessWidget {
  final AppVersionInfo versionInfo;
  final bool isBlocking;
  const UpdateDetailsScreen({
    super.key,
    required this.versionInfo,
    this.isBlocking = false,
  });
  @override
  Widget build(BuildContext context) => PopScope(
    canPop: !isBlocking,
    child: OrixPage(
      title: versionInfo.updateTitle ?? 'Update Orix',
      description: isBlocking
          ? 'Install the required update to continue.'
          : 'A new version is available.',
      maxWidth: 640,
      canPop: !isBlocking,
      children: [
        const PixelAnimation(
          scene: PixelScene.completed,
          label: 'New version available',
        ),
        const SizedBox(height: 24),
        PixelSection(
          title: 'Version ${versionInfo.currentVersion}',
          child: Text(
            versionInfo.updateNotes?.trim().isNotEmpty == true
                ? versionInfo.updateNotes!
                : 'No release notes are available.',
          ),
        ),
        if (versionInfo.storeUrl.isNotEmpty)
          FilledButton(
            onPressed: () => launchUrl(
              Uri.parse(versionInfo.storeUrl),
              mode: LaunchMode.externalApplication,
            ),
            child: const Text('Open update'),
          )
        else
          const Text('The update link is unavailable. Contact support.'),
        if (!isBlocking)
          TextButton(
            onPressed: () => Navigator.of(context).pop(),
            child: const Text('Later'),
          ),
      ],
    ),
  );
}
