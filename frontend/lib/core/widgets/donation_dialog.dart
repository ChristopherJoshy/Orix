import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:url_launcher/url_launcher.dart';
import '../config/donation_config.dart';
import '../theme/app_theme.dart';

/// This preference controls prompting only; it never represents payment status.
class DonationPrompt {
  static const preferenceKey = 'orix.donation.lastPromptDay.v1';
  static bool _showing = false;
  static String? _sessionDay;

  static String day(DateTime now) => '${now.year}-${now.month}-${now.day}';

  static Future<void> maybeShow(BuildContext context) async {
    if (_showing) return;
    final today = day(DateTime.now());
    if (_sessionDay == today) return;
    _showing = true;
    try {
      final prefs = await SharedPreferences.getInstance();
      if (prefs.getString(preferenceKey) == today || !context.mounted) return;
      if (ModalRoute.of(context)?.isCurrent != true) return;
      _sessionDay = today;
      await prefs.setString(preferenceKey, today);
      if (context.mounted) await showDonationDialog(context);
    } on PlatformException {
      // If persistence is unavailable, avoid repeatedly interrupting this session.
      _sessionDay = today;
    } finally {
      _showing = false;
    }
  }
}

Future<void> showDonationDialog(BuildContext context) async {
  DonationPrompt._sessionDay = DonationPrompt.day(DateTime.now());
  try {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(
      DonationPrompt.preferenceKey,
      DonationPrompt._sessionDay!,
    );
  } on PlatformException {
    // Manual access remains available even when preferences cannot be saved.
  }
  if (!context.mounted) return;
  await showDialog<void>(
    context: context,
    builder: (_) => const DonationDialog(),
  );
}

class DonationDialog extends StatefulWidget {
  const DonationDialog({super.key});
  @override
  State<DonationDialog> createState() => _DonationDialogState();
}

class _DonationDialogState extends State<DonationDialog> {
  bool _opening = false;
  String? _error;
  bool get _mobile =>
      defaultTargetPlatform == TargetPlatform.android ||
      defaultTargetPlatform == TargetPlatform.iOS;

  Future<void> _donate() async {
    if (_opening) return;
    setState(() {
      _opening = true;
      _error = null;
    });
    try {
      bool opened;
      if (!kIsWeb && defaultTargetPlatform == TargetPlatform.android) {
        opened =
            await const MethodChannel(
              'orix/donations',
            ).invokeMethod<bool>('openGooglePay', DonationConfig.upi) ??
            false;
      } else {
        opened = await launchUrl(
          Uri.parse(DonationConfig.upi.replaceFirst('upi://', 'gpay://upi/')),
          mode: LaunchMode.externalApplication,
        );
      }
      if (!opened && mounted) {
        setState(
          () => _error =
              'Google Pay could not open. Install or open Google Pay, then pay using the UPI ID below.',
        );
      }
      // Opening an external app is not evidence that a donation was paid.
    } catch (_) {
      if (mounted) {
        setState(
          () => _error =
              'Google Pay could not open. Try again or copy the UPI ID below.',
        );
      }
    } finally {
      if (mounted) setState(() => _opening = false);
    }
  }

  @override
  Widget build(BuildContext context) => AlertDialog(
    title: const Text('Keep ORIX going.'),
    scrollable: true,
    content: SizedBox(
      width: 340,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        mainAxisSize: MainAxisSize.min,
        children: [
          const Icon(Icons.favorite_border, size: 28),
          const SizedBox(height: 16),
          const Text(
            'Donate to help keep ORIX going. Any amount is welcome, and donating is always optional.',
          ),
          const SizedBox(height: 16),
          if (!_mobile) ...[
            Image.asset(
              'assets/images/donation-qr.png',
              width: 246,
              height: 246,
              fit: BoxFit.contain,
              filterQuality: FilterQuality.none,
              semanticLabel: 'Scan to donate with any UPI app',
            ),
            const SizedBox(height: 12),
            const Text('Scan with any UPI app. Choose your amount there.'),
          ] else ...[
            FilledButton(
              onPressed: _opening ? null : _donate,
              child: Text(_opening ? 'Opening Google Pay…' : 'Donate'),
            ),
            const SizedBox(height: 8),
            const Text(
              'Opens Google Pay. Choose any amount there.',
              style: AppTextStyles.bodySmall,
            ),
          ],
          const SizedBox(height: 16),
          const Text(
            'Recipient: ${DonationConfig.payee}',
            style: AppTextStyles.bodySmall,
          ),
          const SelectableText(
            DonationConfig.upiId,
            style: AppTextStyles.bodySmall,
          ),
          TextButton.icon(
            onPressed: () async {
              try {
                await Clipboard.setData(
                  const ClipboardData(text: DonationConfig.upiId),
                );
                if (context.mounted) {
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('UPI ID copied.')),
                  );
                }
              } catch (_) {
                if (mounted) {
                  setState(
                    () => _error =
                        'Could not copy. Select the UPI ID above and copy it.',
                  );
                }
              }
            },
            icon: const Icon(Icons.copy, size: 16),
            label: const Text('Copy UPI ID'),
          ),
          if (_error != null)
            Padding(
              padding: const EdgeInsets.only(top: 12),
              child: Text(_error!, semanticsLabel: _error),
            ),
        ],
      ),
    ),
    actions: [
      TextButton(
        onPressed: () => Navigator.pop(context),
        child: const Text('Maybe later'),
      ),
    ],
  );
}
