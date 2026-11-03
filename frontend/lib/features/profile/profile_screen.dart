import 'package:orix/core/widgets/orix_tour.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:orix/core/providers/auth_provider.dart';
import 'package:orix/core/config/api_config.dart';
import 'package:orix/core/config/privacy_config.dart';
import 'package:orix/core/router/app_router.dart';
import 'package:orix/core/theme/app_theme.dart';
import 'package:orix/core/widgets/custom_alert_dialog.dart';
import 'package:orix/core/widgets/pixel_ui.dart';
import 'help_support/blocked_users_screen.dart';
import 'help_support/report_history_screen.dart';
import 'package:orix/core/widgets/account_photo.dart';

class ProfileScreen extends StatelessWidget {
  const ProfileScreen({super.key});
  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthProvider>();
    final user = auth.user;
    Widget link(String label, IconData icon, VoidCallback tap) => ListTile(
      contentPadding: EdgeInsets.zero,
      leading: Icon(icon),
      title: Text(label),
      trailing: const Icon(Icons.chevron_right),
      onTap: tap,
    );
    return OrixPage(
      title: 'Your account',
      bottom: const OrixNavigation(selected: 3),
      children: [
        PixelSection(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  AccountPhoto(url: user?.photoUrl, size: 48),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          user?.displayName ?? 'Your profile',
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        const SizedBox(height: 4),
                        Text(
                          user?.email ?? '',
                          style: Theme.of(context).textTheme.bodySmall
                              ?.copyWith(color: AppColors.textSecondary),
                        ),
                      ],
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 20),
              OutlinedButton(
                onPressed: () => context.push(RoutePaths.editProfile),
                child: const Text('Edit profile'),
              ),
            ],
          ),
        ),
        PixelSection(
          title: 'Travel',
          child: Column(
            children: [
              link(
                'Offer a lift',
                Icons.directions_car_outlined,
                () => context.push(RoutePaths.riderSetup),
              ),
              const Divider(),
              link(
                'Journey history',
                Icons.history,
                () => context.push(RoutePaths.rideHistory),
              ),
              const Divider(),
              link(
                'Notification preferences',
                Icons.notifications_outlined,
                () => context.push(RoutePaths.notificationPreferences),
              ),
            ],
          ),
        ),
        PixelSection(
          title: 'Safety and support',
          child: Column(
            children: [
              link(
                'Blocked contacts',
                Icons.block_outlined,
                () => Navigator.of(context).push(
                  MaterialPageRoute<void>(
                    builder: (_) => const BlockedUsersScreen(),
                  ),
                ),
              ),
              const Divider(),
              link(
                'Report conversations',
                Icons.flag_outlined,
                () => Navigator.of(context).push(
                  MaterialPageRoute<void>(
                    builder: (_) => const ReportHistoryScreen(),
                  ),
                ),
              ),
              const Divider(),
              link(
                'Help and support',
                Icons.help_outline,
                () => context.push(RoutePaths.helpSupport),
              ),
            ],
          ),
        ),
        PixelSection(
          title: 'About Orix',
          child: Column(
            children: [
              link('App tour', Icons.help_outline, () => showOrixTour(context)),
              const Divider(),
              link(
                'About',
                Icons.info_outline,
                () => context.push(RoutePaths.about),
              ),
              const Divider(),
              link(
                'Community terms',
                Icons.description_outlined,
                () => context.push(RoutePaths.termsAndConditions),
              ),
              if (PrivacyConfig.policy != null) ...[
                const Divider(),
                link(
                  'Privacy policy',
                  Icons.privacy_tip_outlined,
                  () => launchUrl(
                    PrivacyConfig.policy!,
                    mode: LaunchMode.externalApplication,
                  ),
                ),
              ],
            ],
          ),
        ),
        OutlinedButton(
          onPressed: () => _showSignOutDialog(context, auth),
          child: const Text('Sign out'),
        ),
        const SizedBox(height: 16),
        TextButton(
          onPressed: () => _showDeleteDialog(context, auth),
          child: const Text('Request account deletion'),
        ),
      ],
    );
  }

  Future<void> _showDeleteDialog(
    BuildContext context,
    AuthProvider auth,
  ) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Delete your account?'),
        content: const Text(
          'Your account will be queued for deletion and you will be signed out. You will be asked to sign in with Google again.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('Request deletion'),
          ),
        ],
      ),
    );
    if (confirmed != true || !context.mounted) return;
    String? emulatorPassword;
    if (InternalTestConfig.enabled) {
      var value = '';
      emulatorPassword = await showDialog<String>(
        context: context,
        builder: (context) => AlertDialog(
          title: const Text('Confirm test account password'),
          content: TextField(
            obscureText: true,
            autofocus: true,
            onChanged: (text) => value = text,
            decoration: const InputDecoration(labelText: 'Emulator password'),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text('Cancel'),
            ),
            FilledButton(
              onPressed: () => Navigator.pop(context, value),
              child: const Text('Continue'),
            ),
          ],
        ),
      );
      if (emulatorPassword == null || !context.mounted) return;
    }
    final error = await auth.requestAccountDeletion(
      emulatorPassword: emulatorPassword,
    );
    if (!context.mounted) return;
    if (error == null) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Account deletion requested.')),
      );
      context.go(RoutePaths.login);
    } else {
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text(error)));
    }
  }

  void _showSignOutDialog(
    BuildContext context,
    AuthProvider authProvider,
  ) async {
    final confirm = await CustomAlertDialog.show<bool>(
      context: context,
      title: 'Sign Out?',
      message: 'Are you sure you want to sign out?',
      primaryButtonText: 'Sign Out',
      onPrimaryPressed: () => Navigator.pop(context, true),
      secondaryButtonText: 'Cancel',
      onSecondaryPressed: () => Navigator.pop(context, false),
      icon: Icons.logout,
      iconColor: AppColors.error,
    );

    if (confirm == true) {
      await authProvider.signOut();
      if (context.mounted) context.go(RoutePaths.login);
    }
  }
}
