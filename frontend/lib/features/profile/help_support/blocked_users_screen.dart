/// Manage blocks for the signed-in account.
library;

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:orix/core/services/api_service.dart';
import 'package:orix/core/widgets/pixel_ui.dart';
import 'email_person_picker.dart';
import 'support_service.dart';

class BlockedUsersScreen extends StatefulWidget {
  const BlockedUsersScreen({super.key});
  @override
  State<BlockedUsersScreen> createState() => _BlockedUsersScreenState();
}

class _BlockedUsersScreenState extends State<BlockedUsersScreen> {
  List<Map<String, dynamic>> _users = [];
  String? _identity;
  String? _error;
  bool _busy = false;
  UserSearchResult? _selected;
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) _load();
    });
  }

  bool _valid(ApiService api, String? id) =>
      mounted && id != null && api.authIdentity == id;
  Future<void> _load() async {
    final api = context.read<ApiService>();
    final id = api.authIdentity;
    if (id == null || _busy) return;
    setState(() {
      _busy = true;
      _error = null;
      if (_identity != id) _users = [];
      _identity = id;
    });
    try {
      final response = await api.get<Map<String, dynamic>>(
        '/users/me/blocks',
        forceRefresh: true,
      );
      if (!_valid(api, id)) return;
      if (!response.success || response.data == null) {
        setState(
          () => _error = response.error ?? 'Could not load blocked travelers.',
        );
        return;
      }
      setState(
        () => _users = (response.data!['blocked_users'] as List<dynamic>? ?? [])
            .whereType<Map<String, dynamic>>()
            .where((item) => item['user_id'] is String)
            .toList(),
      );
    } catch (_) {
      if (_valid(api, id))
        setState(() => _error = 'Could not load blocked travelers.');
    } finally {
      if (_valid(api, id)) setState(() => _busy = false);
    }
  }

  Future<void> _unblock(Map<String, dynamic> user) async {
    final api = context.read<ApiService>();
    final id = _identity;
    final target = user['user_id'] as String;
    final accepted = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Unblock this traveler?'),
        content: const Text('You may be matched with this traveler again.'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('Unblock'),
          ),
        ],
      ),
    );
    if (accepted != true || !_valid(api, id) || _busy) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final response = await api.delete<Map<String, dynamic>>(
        '/users/${Uri.encodeComponent(target)}/block',
      );
      if (!_valid(api, id)) return;
      if (response.success) {
        setState(() => _users.removeWhere((item) => item['user_id'] == target));
      } else {
        setState(
          () => _error = response.error ?? 'Could not unblock traveler.',
        );
      }
    } catch (_) {
      if (_valid(api, id))
        setState(
          () => _error = 'Could not confirm unblock. Refresh and try again.',
        );
    } finally {
      if (_valid(api, id)) setState(() => _busy = false);
    }
  }

  Future<void> _block() async {
    final selected = _selected;
    if (selected == null || _busy) return;
    final api = context.read<ApiService>();
    final identity = api.authIdentity;
    if (identity == null) return;
    final confirm = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Block this contact?'),
        content: Text(
          '${selected.email} will be excluded from future matches. Current shared groups may be cancelled.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('Block'),
          ),
        ],
      ),
    );
    if (confirm != true || !_valid(api, identity)) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final response = await api.put<Map<String, dynamic>>(
        '/users/${Uri.encodeComponent(selected.userId)}/block',
        body: {},
      );
      if (!_valid(api, identity)) return;
      if (!response.success) {
        setState(() => _error = response.error ?? 'Could not block contact.');
        return;
      }
      setState(() => _selected = null);
    } catch (_) {
      if (_valid(api, identity))
        setState(
          () => _error = 'Block not confirmed. Refresh before retrying.',
        );
    } finally {
      if (_valid(api, identity)) setState(() => _busy = false);
    }
    if (_valid(api, identity) && _error == null) await _load();
  }

  @override
  Widget build(BuildContext context) {
    final users = _identity == context.read<ApiService>().authIdentity
        ? _users
        : <Map<String, dynamic>>[];
    return OrixPage(
      title: 'Blocked contacts',
      description: 'Search by email to keep someone out of future matches.',
      children: [
        PixelSection(
          title: 'Block a contact',
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              EmailPersonPicker(
                key: ValueKey(_users.length),
                onSelected: (person) => setState(() => _selected = person),
                enabled: !_busy,
              ),
              const SizedBox(height: 16),
              FilledButton(
                onPressed: _selected == null || _busy ? null : _block,
                child: const Text('Block selected contact'),
              ),
            ],
          ),
        ),
        if (_busy) const PixelLoader(),
        if (_error != null) Text(_error!),
        PixelSection(
          title: 'Your blocked list',
          child: Column(
            children: [
              if (users.isEmpty && !_busy && _error == null)
                const Text('No blocked contacts.'),
              for (final user in users)
                ListTile(
                  contentPadding: EdgeInsets.zero,
                  title: Text(
                    (user['email'] as String?)?.isNotEmpty == true
                        ? user['email'] as String
                        : 'Email unavailable',
                  ),
                  subtitle: Text(user['display_name'] as String? ?? ''),
                  trailing: TextButton(
                    onPressed: _busy ? null : () => _unblock(user),
                    child: const Text('Unblock'),
                  ),
                ),
            ],
          ),
        ),
        TextButton(
          onPressed: _busy ? null : _load,
          child: const Text('Refresh list'),
        ),
      ],
    );
  }
}
