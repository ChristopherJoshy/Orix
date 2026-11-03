import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/connectivity_provider.dart';
import 'pixel_ui.dart';

/// Offline indicator banner shown when device loses internet connection
class OfflineBanner extends StatelessWidget {
  const OfflineBanner({super.key});

  @override
  Widget build(BuildContext context) {
    return Consumer<ConnectivityProvider>(
      builder: (context, connectivity, child) {
        // Only show banner if offline
        if (connectivity.isOnline) {
          return SizedBox.shrink();
        }

        return Container(
          width: double.infinity,
          padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 16),
          decoration: BoxDecoration(
            color: const Color(0xFF333333),
            boxShadow: [
              BoxShadow(
                color: Colors.black.withValues(alpha: 0.1),
                blurRadius: 8,
                offset: const Offset(0, 2),
              ),
            ],
          ),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              const SizedBox(
                width: 56,
                height: 42,
                child: PixelAnimation(
                  scene: PixelScene.connectionLost,
                  size: 56,
                  color: Colors.white,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Text(
                  'Connection lost. The network is on a chaya break.',
                  style: Theme.of(
                    context,
                  ).textTheme.bodyMedium?.copyWith(color: Colors.white),
                ),
              ),
              TextButton(
                onPressed: () async {
                  try {
                    await connectivity.refresh();
                  } catch (_) {
                    if (context.mounted)
                      ScaffoldMessenger.of(context).showSnackBar(
                        const SnackBar(
                          content: Text(
                            'Still on a chaya break. Check your connection and retry.',
                          ),
                        ),
                      );
                  }
                },
                style: TextButton.styleFrom(foregroundColor: Colors.white),
                child: const Text('Retry'),
              ),
            ],
          ),
        );
      },
    );
  }
}
