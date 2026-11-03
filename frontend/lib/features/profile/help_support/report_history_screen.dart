/// Reports submitted by the signed-in user, paged from the moderation API.
library;

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:orix/core/services/api_service.dart';
import 'package:go_router/go_router.dart';
import 'package:orix/core/widgets/pixel_ui.dart';

class ReportHistoryScreen extends StatefulWidget {
  const ReportHistoryScreen({super.key});
  @override
  State<ReportHistoryScreen> createState() => _ReportHistoryScreenState();
}

class _ReportHistoryScreenState extends State<ReportHistoryScreen> {
  final Map<String, Map<String, dynamic>> _records = {};
  String? _identity;
  String? _cursor;
  String? _error;
  bool _busy = false;
  int _version = 0;
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) _load();
    });
  }

  @override
  void dispose() {
    _version++;
    super.dispose();
  }

  Future<void> _load({bool more = false}) async {
    final api = context.read<ApiService>();
    final identity = api.authIdentity;
    if (_busy ||
        identity == null ||
        (more && (_identity != identity || _cursor == null)))
      return;
    final version = ++_version;
    setState(() {
      _busy = true;
      _error = null;
      if (_identity != identity) {
        _records.clear();
        _cursor = null;
      }
      _identity = identity;
    });
    try {
      final response = await api.get<Map<String, dynamic>>(
        '/reports',
        forceRefresh: true,
        queryParams: {'limit': '50', if (more) 'before': _cursor!},
      );
      if (!mounted || version != _version || api.authIdentity != identity)
        return;
      if (!response.success || response.data == null) {
        setState(() => _error = response.error ?? 'Could not load reports.');
        return;
      }
      final rows = response.data!['reports'] as List<dynamic>? ?? [];
      setState(() {
        if (!more) _records.clear();
        for (final row in rows) {
          if (row is Map<String, dynamic> && row['report_id'] is String)
            _records[row['report_id'] as String] = row;
        }
        _cursor = response.nextCursor;
      });
    } catch (_) {
      if (mounted && version == _version && api.authIdentity == identity)
        setState(() => _error = 'Could not load reports.');
    } finally {
      if (mounted && version == _version && api.authIdentity == identity)
        setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final identity = context.read<ApiService>().authIdentity;
    final reports = identity != null && identity == _identity
        ? _records.values.toList()
        : <Map<String, dynamic>>[];
    return OrixPage(
      title: 'Report conversations',
      description: 'Read moderator replies or add more context.',
      children: [
        if (_busy) const PixelLoader(),
        if (_error != null) Text(_error!),
        if (reports.isEmpty && !_busy && _error == null)
          const PixelSection(child: Text('You have no report conversations.')),
        for (final report in reports)
          PixelSection(
            child: ListTile(
              contentPadding: EdgeInsets.zero,
              title: Text(
                (report['category'] as String? ?? 'Report').replaceAll(
                  '_',
                  ' ',
                ),
              ),
              subtitle: Text(
                '${report['status'] ?? 'pending'}\n${report['note'] ?? ''}',
                maxLines: 3,
                overflow: TextOverflow.ellipsis,
              ),
              trailing: const Icon(Icons.chevron_right),
              onTap: () async {
                await context.push(
                  '/profile/support/reports/${Uri.encodeComponent(report['report_id'] as String)}',
                );
                if (mounted) _load();
              },
            ),
          ),
        if (_cursor != null && reports.isNotEmpty)
          TextButton(
            onPressed: _busy ? null : () => _load(more: true),
            child: const Text('Earlier conversations'),
          ),
        TextButton(
          onPressed: _busy ? null : _load,
          child: const Text('Refresh'),
        ),
      ],
    );
  }
}
