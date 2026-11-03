/// Terms and Conditions Screen
///
/// Displays full Terms & Conditions document.
library;

import 'package:flutter/material.dart';
import 'package:orix/core/widgets/pixel_ui.dart';
import 'package:provider/provider.dart';
import 'package:orix/core/services/api_service.dart';
import 'package:orix/core/theme/app_theme.dart';

class TermsScreen extends StatelessWidget {
  const TermsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return OrixPage(
      title: 'Community terms',
      maxWidth: 720,
      children: [
        _buildSection(
          'Welcome to ORIX',
          'ORIX is a travel coordination platform for shared autos, travel partners, and lifts.',
        ),
        _buildSection(
          '1. Eligibility & Verification',
          'You may register with a Google account from any email domain. You must complete safety onboarding. By registering, you confirm you are at least 18 years of age.',
        ),
        _buildSection(
          '2. Cost Sharing & Payments',
          'ORIX is a coordination tool ONLY. We do NOT process payments within the app. All cost-sharing arrangements (fuel costs, tolls) are to be settled directly between the Rider and Passenger via cash, UPI, or other mutual agreement. ORIX does not charge a commission on rides.',
        ),
        _buildSection(
          '3. Service Availability',
          'Features and availability may change. Check the published privacy policy for information about data use.',
        ),
        _buildSection(
          '4. Community Guidelines',
          '''As a member of the ORIX community, you agree to:
• Use your real identity and accurate profile information
• Be punctual and respectful
• Maintain a clean and safe vehicle (Riders)
• Follow local traffic laws and applicable rules
• Not engage in harassment, discrimination, or unsafe behavior''',
        ),
        _buildSection(
          '5. Safety & Liability Disclaimer',
          'ORIX is a technology platform that connects users; we do not provide transportation services. Users travel at their own risk. We strongly encourage confirming identities and vehicle details before starting a ride. ORIX is not liable for accidents, damages, losses, or disputes arising from rides coordinated through the platform.',
        ),
        _buildSection(
          '6. Privacy & Data',
          'We collect minimal data necessary for ride matching (Location, Name, Email). Your location is shared only during active ride coordination. We do not sell your personal data to third parties.',
        ),
        _buildSection(
          '7. Account Termination',
          'We reserve the right to suspend or ban users who violate these terms, falsify their identity, or are reported for unsafe/inappropriate conduct.',
        ),
        _buildSection(
          '8. Contact',
          'For support or to report safety concerns, please use the "Help & Support" section in the app.',
        ),
        const SizedBox(height: 24),
        const _TermsAcceptance(),
        const SizedBox(height: 24),
        Container(
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: AppColors.surfaceVariant,
            borderRadius: BorderRadius.zero,
            border: Border.all(color: AppColors.outline),
          ),
          child: Column(
            children: [
              Text(
                'Last Updated',
                style: TextStyle(color: AppColors.textTertiary, fontSize: 12),
              ),
              const SizedBox(height: 4),
              const Text(
                'December 26, 2025',
                style: TextStyle(fontWeight: FontWeight.w600, fontSize: 14),
              ),
            ],
          ),
        ),
      ],
    );
  }

  Widget _buildSection(String title, String content) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            title,
            style: const TextStyle(
              fontFamily: AppTextStyles.displayFamily,
              fontSize: 24,
              fontWeight: FontWeight.w500,
              color: AppColors.textPrimary,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            content,
            style: TextStyle(
              fontSize: 14,
              height: 1.6,
              color: AppColors.textSecondary,
            ),
          ),
        ],
      ),
    );
  }
}

class _TermsAcceptance extends StatefulWidget {
  const _TermsAcceptance();
  @override
  State<_TermsAcceptance> createState() => _TermsAcceptanceState();
}

class _TermsAcceptanceState extends State<_TermsAcceptance> {
  static const _displayedVersion = '1.0';
  bool _busy = false;
  bool? _needsAcceptance;
  String? _version;
  String? _error;
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) _load();
    });
  }

  bool _valid(ApiService api, String? identity) =>
      mounted && identity != null && api.authIdentity == identity;
  Future<void> _load() async {
    final api = context.read<ApiService>();
    final identity = api.authIdentity;
    if (_busy || identity == null) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final response = await api.get<Map<String, dynamic>>(
        '/users/me/terms-status',
        forceRefresh: true,
      );
      if (!_valid(api, identity)) return;
      if (!response.success || response.data == null) {
        setState(
          () => _error = response.error ?? 'Could not load acceptance status.',
        );
        return;
      }
      setState(() {
        _version = response.data!['current_version'] as String?;
        _needsAcceptance = response.data!['needs_acceptance'] as bool?;
      });
    } catch (_) {
      if (_valid(api, identity))
        setState(() => _error = 'Could not load acceptance status.');
    } finally {
      if (_valid(api, identity)) setState(() => _busy = false);
    }
  }

  Future<void> _accept() async {
    final api = context.read<ApiService>();
    final identity = api.authIdentity;
    if (_busy || identity == null || _version != _displayedVersion) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final response = await api.post<Map<String, dynamic>>(
        '/users/me/accept-terms',
        body: {'terms_version': _displayedVersion},
      );
      if (!_valid(api, identity)) return;
      if (response.success) {
        setState(() => _needsAcceptance = false);
      } else {
        setState(
          () => _error = response.error ?? 'Could not record acceptance.',
        );
      }
    } catch (_) {
      if (_valid(api, identity))
        setState(() => _error = 'Could not record acceptance.');
    } finally {
      if (_valid(api, identity)) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) => Column(
    crossAxisAlignment: CrossAxisAlignment.stretch,
    children: [
      if (_busy) const LinearProgressIndicator(),
      if (_error != null)
        Row(
          children: [
            Expanded(child: Text(_error!)),
            TextButton(
              onPressed: _busy ? null : _load,
              child: const Text('Retry'),
            ),
          ],
        ),
      if (_version != null && _version != _displayedVersion)
        const Text(
          'Updated terms are unavailable in this app version. Please update before accepting.',
        ),
      if (_version == _displayedVersion && _needsAcceptance == true)
        FilledButton(
          onPressed: _busy ? null : _accept,
          child: const Text('Accept these terms'),
        ),
      if (_version == _displayedVersion && _needsAcceptance == false)
        const Text('You have accepted these terms.'),
    ],
  );
}
