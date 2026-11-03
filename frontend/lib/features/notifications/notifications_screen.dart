/// Notifications Screen
///
/// In-app notification center.
library;

import 'package:orix/core/widgets/pixel_ui.dart';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:go_router/go_router.dart';

import 'package:orix/core/providers/notification_provider.dart';
import 'package:orix/core/theme/app_theme.dart';
import 'package:orix/core/utils/time_utils.dart';
import 'package:orix/core/widgets/custom_alert_dialog.dart';

class NotificationsScreen extends StatefulWidget {
  const NotificationsScreen({super.key});

  @override
  State<NotificationsScreen> createState() => _NotificationsScreenState();
}

class _NotificationsScreenState extends State<NotificationsScreen> {
  bool _isMarkingAll = false;
  bool _isClearing = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) _loadNotifications();
    });
  }

  Future<void> _loadNotifications() async {
    await context.read<NotificationProvider>().fetchNotifications();
  }

  Future<void> _markAllRead() async {
    if (_isMarkingAll) return;
    setState(() => _isMarkingAll = true);
    await context.read<NotificationProvider>().markAllAsRead();
    if (mounted) setState(() => _isMarkingAll = false);
  }

  Future<void> _clearAll() async {
    if (_isClearing) return;
    final confirmed = await CustomAlertDialog.show<bool>(
      context: context,
      title: 'Clear all notifications?',
      message: 'This action cannot be undone.',
      primaryButtonText: 'Clear All',
      onPrimaryPressed: () => Navigator.pop(context, true),
      secondaryButtonText: 'Cancel',
      onSecondaryPressed: () => Navigator.pop(context, false),
      icon: Icons.delete_forever_rounded,
      iconColor: AppColors.error,
    );

    if (confirmed == true && mounted) {
      setState(() => _isClearing = true);
      await context.read<NotificationProvider>().clearAllNotifications();
      if (mounted) setState(() => _isClearing = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final provider = context.watch<NotificationProvider>();
    final notifications = provider.notifications;
    return OrixPage(
      title: 'Notifications',
      description: provider.isLoading
          ? 'Loading your updates…'
          : '${provider.unreadCount} unread',
      actions: [
        IconButton(
          tooltip: 'Refresh notifications',
          onPressed: provider.isLoading ? null : _loadNotifications,
          icon: const Icon(Icons.refresh),
        ),
        PopupMenuButton<String>(
          tooltip: 'Inbox actions',
          enabled:
              notifications.isNotEmpty &&
              !provider.isMutating &&
              !_isClearing &&
              !_isMarkingAll,
          onSelected: (action) {
            if (action == 'read') {
              _markAllRead();
            } else {
              _clearAll();
            }
          },
          itemBuilder: (_) => [
            if (provider.hasUnread)
              const PopupMenuItem(value: 'read', child: Text('Mark all read')),
            if (notifications.isNotEmpty)
              const PopupMenuItem(value: 'clear', child: Text('Clear inbox')),
          ],
        ),
      ],
      children: [
        if (provider.error != null)
          Padding(
            padding: const EdgeInsets.only(bottom: 20),
            child: Text(provider.error!),
          ),
        if (provider.isLoading || _isClearing || _isMarkingAll)
          const PixelLoader(),
        if (notifications.isEmpty &&
            !provider.isLoading &&
            provider.error == null)
          const PixelSection(
            child: Column(
              children: [
                OrixLogo(symbolOnly: true, width: 36),
                SizedBox(height: 20),
                Text('You’re up to date.'),
              ],
            ),
          )
        else
          for (final notification in notifications)
            _buildNotificationCard(notification),
        if (provider.hasMore)
          TextButton(
            onPressed: provider.isLoading || provider.isMutating
                ? null
                : () => provider.fetchNotifications(loadMore: true),
            child: const Text('Earlier notifications'),
          ),
      ],
    );
  }

  Future<void> _open(AppNotification notification) async {
    final provider = context.read<NotificationProvider>();
    if (provider.isMutating) return;
    final account = provider.accountId;
    if (!notification.isRead) await provider.markAsRead(notification.id);
    if (!mounted || !provider.isForUser(account)) return;
    if (notification.type == 'group_rejected' ||
        notification.data?['type'] == 'group_rejected') {
      context.push('/ride/matching');
      return;
    }
    if (notification.type == 'engagement_reminder') {
      context.go('/home');
      return;
    }
    final group = notification.data?['group_id'];
    final ticket = notification.data?['ticket_id'];
    final report = notification.data?['report_id'];
    if (report is String && report.isNotEmpty) {
      await context.push(
        '/profile/support/reports/${Uri.encodeComponent(report)}',
      );
      return;
    }
    if (group is String && group.isNotEmpty) {
      final chat = const {
        'new_chat_message',
        'chat_message',
        'new_message',
      }.contains(notification.type);
      context.push(
        '/groups/${Uri.encodeComponent(group)}${chat ? '/chat' : ''}',
      );
    } else if (ticket is String && ticket.isNotEmpty) {
      context.push('/profile/support/tickets/${Uri.encodeComponent(ticket)}');
    }
  }

  Widget _buildNotificationCard(AppNotification notification) {
    final scene = switch (notification.type) {
      'match_found' || 'match' => PixelScene.found,
      'group_cancelled' || 'ride_cancelled' => PixelScene.cancelled,
      'ride_completed' || 'group_finish' => PixelScene.completed,
      _ => null,
    };
    return Card(
      child: InkWell(
        onTap: context.watch<NotificationProvider>().isMutating
            ? null
            : () => _open(notification),
        child: Padding(
          padding: const EdgeInsets.all(20),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  if (scene != null)
                    SizedBox(
                      width: 64,
                      height: 40,
                      child: PixelAnimation(
                        scene: scene,
                        size: 64,
                        label: notification.type.replaceAll('_', ' '),
                      ),
                    )
                  else
                    const OrixLogo(symbolOnly: true, width: 28),
                  const SizedBox(width: 16),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        if (!notification.isRead)
                          const Padding(
                            padding: EdgeInsets.only(bottom: 6),
                            child: Text(
                              'Unread',
                              style: TextStyle(
                                fontSize: 11,
                                fontWeight: FontWeight.w600,
                              ),
                            ),
                          ),
                        Text(
                          notification.title,
                          style: TextStyle(
                            fontWeight: notification.isRead
                                ? FontWeight.w500
                                : FontWeight.w600,
                            fontSize: 15,
                            height: 1.5,
                          ),
                        ),
                      ],
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 12),
              Text(
                notification.body,
                style: const TextStyle(
                  fontSize: 14,
                  height: 1.6,
                  color: AppColors.textSecondary,
                ),
              ),
              const SizedBox(height: 12),
              Text(
                TimeUtils.formatRelative(notification.createdAt),
                style: const TextStyle(
                  fontSize: 12,
                  color: AppColors.textSecondary,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
