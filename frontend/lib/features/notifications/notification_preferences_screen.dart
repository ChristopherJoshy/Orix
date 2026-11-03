/// Account preferences saved by the Go notification service.
library;

import 'package:flutter/material.dart';
import 'package:orix/core/widgets/pixel_ui.dart';
import 'package:provider/provider.dart';
import 'package:orix/core/services/api_service.dart';

class NotificationPreferencesScreen extends StatefulWidget {
  const NotificationPreferencesScreen({super.key});
  @override
  State<NotificationPreferencesScreen> createState() =>
      _NotificationPreferencesScreenState();
}

class _NotificationPreferencesScreenState
    extends State<NotificationPreferencesScreen> {
  Map<String, bool>? _values;
  bool _busy = false;
  String? _error;
  String? _identity;
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
    if (identity == null || _busy) return;
    setState(() {
      _busy = true;
      _error = null;
      _identity = identity;
    });
    try {
      final response = await api.get<Map<String, dynamic>>(
        '/users/me/notification-preferences',
        forceRefresh: true,
      );
      if (!_valid(api, identity)) return;
      if (!response.success || response.data == null) {
        setState(
          () => _error = response.error ?? 'Could not load preferences.',
        );
        return;
      }
      setState(
        () => _values = {
          for (final key in [
            'ride_reminders',
            'group_updates',
            'chat_messages',
            'promotions',
          ])
            key: response.data![key] == true,
        },
      );
    } catch (_) {
      if (_valid(api, identity)) {
        setState(() => _error = 'Could not load preferences.');
      }
    } finally {
      if (_valid(api, identity)) setState(() => _busy = false);
    }
  }

  Future<void> _save(String key, bool value) async {
    final api = context.read<ApiService>();
    final identity = _identity;
    if (!_valid(api, identity) || _busy) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final response = await api.put<Map<String, dynamic>>(
        '/users/me/notification-preferences',
        body: {key: value},
      );
      if (!_valid(api, identity)) return;
      if (!response.success || response.data == null) {
        setState(() => _error = response.error ?? 'Could not save preference.');
        return;
      }
      setState(
        () => _values = {
          for (final key in [
            'ride_reminders',
            'group_updates',
            'chat_messages',
            'promotions',
          ])
            key: response.data![key] == true,
        },
      );
    } catch (_) {
      if (_valid(api, identity)) {
        setState(() => _error = 'Could not save preference.');
      }
    } finally {
      if (_valid(api, identity)) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final values = _identity == context.read<ApiService>().authIdentity
        ? _values
        : null;
    return OrixPage(
      title: 'Notifications',
      description:
          'Choose what you receive. Device notification permissions also apply.',
      maxWidth: 680,
      children: [
        if (_busy) const PixelLoader(),
        if (_error != null) Text(_error!),
        if (values != null)
          PixelSection(
            title: 'Your preferences',
            child: Column(
              children: [
                for (final item in const [
                  (
                    'ride_reminders',
                    'Pickup reminders',
                    'Before your journey starts',
                  ),
                  (
                    'group_updates',
                    'Group updates',
                    'Changes to your travel group',
                  ),
                  (
                    'chat_messages',
                    'Group messages',
                    'New messages from your group',
                  ),
                  (
                    'promotions',
                    'A little nudge',
                    'Optional Manglish reminders, up to every four hours during the day (IST)',
                  ),
                ])
                  PixelToggle(
                    contentPadding: EdgeInsets.zero,
                    title: Text(item.$2),
                    subtitle: Text(item.$3),
                    value: values[item.$1] ?? false,
                    onChanged: _busy ? null : (value) => _save(item.$1, value),
                  ),
              ],
            ),
          ),
        TextButton(
          onPressed: _busy ? null : _load,
          child: const Text('Refresh preferences'),
        ),
      ],
    );
  }
}
