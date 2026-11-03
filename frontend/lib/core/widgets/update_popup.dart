/// Update Popup Widget
///
/// Custom blocking dialog for app updates.
library;

import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';

import '../services/app_version_service.dart';

class UpdatePopup extends StatelessWidget {
  final AppVersionInfo versionInfo;
  final bool isBlocking;
  final VoidCallback? onViewDetails;
  final VoidCallback? onDismiss;

  const UpdatePopup({
    super.key,
    required this.versionInfo,
    this.isBlocking = false,
    this.onViewDetails,
    this.onDismiss,
  });

  static Future<void> show(
    BuildContext context, {
    required AppVersionInfo versionInfo,
    bool isBlocking = false,
    VoidCallback? onViewDetails,
    VoidCallback? onDismiss,
  }) async {
    await showDialog(
      context: context,
      barrierDismissible: !isBlocking,
      builder: (context) => UpdatePopup(
        versionInfo: versionInfo,
        isBlocking: isBlocking,
        onViewDetails: onViewDetails,
        onDismiss: onDismiss,
      ),
    );
  }

  @override
  Widget build(BuildContext context) => PopScope(
    canPop: !isBlocking,
    child: AlertDialog(
      title: Text(versionInfo.updateTitle ?? 'Update Orix'),
      content: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Version ${versionInfo.currentVersion}'),
            const SizedBox(height: 16),
            Text(
              versionInfo.updateNotes ??
                  'Open the release details to learn more.',
            ),
            if (isBlocking) ...[
              const SizedBox(height: 16),
              const Text('This update is required to continue.'),
            ],
          ],
        ),
      ),
      actions: [
        if (onViewDetails != null)
          TextButton(
            onPressed: onViewDetails,
            child: const Text('Release details'),
          ),
        if (!isBlocking)
          TextButton(
            onPressed: onDismiss ?? () => Navigator.pop(context),
            child: const Text('Later'),
          ),
        if (versionInfo.storeUrl.isNotEmpty)
          FilledButton(
            onPressed: () => _openStore(context),
            child: const Text('Update'),
          ),
      ],
    ),
  );

  Future<void> _openStore(BuildContext context) async {
    final url = Uri.parse(versionInfo.storeUrl);
    if (await canLaunchUrl(url)) {
      await launchUrl(url, mode: LaunchMode.externalApplication);
    }
  }
}
