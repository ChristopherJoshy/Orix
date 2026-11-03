import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:orix/core/widgets/pixel_ui.dart';
import 'package:orix/core/router/app_router.dart';
import 'blocked_users_screen.dart';
import 'report_history_screen.dart';

class HelpSupportScreen extends StatelessWidget {
  const HelpSupportScreen({super.key});
  @override
  Widget build(BuildContext context) {
    Widget item(
      String title,
      String subtitle,
      IconData icon,
      VoidCallback tap,
    ) => ListTile(
      contentPadding: EdgeInsets.zero,
      leading: Icon(icon),
      title: Text(title),
      subtitle: Text(subtitle),
      trailing: const Icon(Icons.chevron_right),
      onTap: tap,
    );
    return OrixPage(
      title: 'Help and safety',
      description: 'Find an answer or start a private conversation.',
      children: [
        PixelSection(
          title: 'Safety',
          child: Column(
            children: [
              item(
                'Start a report',
                'Contact the moderation team',
                Icons.flag_outlined,
                () =>
                    context.push(RoutePaths.createTicket, extra: 'report_user'),
              ),
              const Divider(),
              item(
                'Report conversations',
                'Read replies and add context',
                Icons.chat_bubble_outline,
                () => Navigator.of(context).push(
                  MaterialPageRoute<void>(
                    builder: (_) => const ReportHistoryScreen(),
                  ),
                ),
              ),
              const Divider(),
              item(
                'Blocked contacts',
                'Manage your list by email',
                Icons.block_outlined,
                () => Navigator.of(context).push(
                  MaterialPageRoute<void>(
                    builder: (_) => const BlockedUsersScreen(),
                  ),
                ),
              ),
            ],
          ),
        ),
        PixelSection(
          title: 'Support',
          child: Column(
            children: [
              item(
                'Your support conversations',
                'Open tickets and previous replies',
                Icons.forum_outlined,
                () => context.push(RoutePaths.ticketList),
              ),
              const Divider(),
              item(
                'Something isn’t working',
                'Send a message to support',
                Icons.build_outlined,
                () => context.push(RoutePaths.createTicket, extra: 'issue'),
              ),
              const Divider(),
              item(
                'Suggest a change',
                'Share a product idea',
                Icons.lightbulb_outline,
                () => context.push(RoutePaths.createTicket, extra: 'feature'),
              ),
            ],
          ),
        ),
        OutlinedButton(
          onPressed: () => context.push(RoutePaths.faq),
          child: const Text('Common questions'),
        ),
        const SizedBox(height: 24),
        const Text(
          'For immediate danger, contact local emergency services. Orix reports are not an emergency channel.',
        ),
      ],
    );
  }
}
