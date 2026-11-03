import 'dart:async';
import 'package:orix/core/services/websocket_service.dart';
import 'package:flutter/material.dart';
import 'package:orix/core/widgets/pixel_ui.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
import 'package:provider/provider.dart';
import 'package:orix/core/services/api_service.dart';
import 'package:orix/core/router/app_router.dart';
import 'support_service.dart';

class TicketListScreen extends StatefulWidget {
  const TicketListScreen({super.key});

  @override
  State<TicketListScreen> createState() => _TicketListScreenState();
}

class _TicketListScreenState extends State<TicketListScreen> {
  final Map<String, SupportTicket> _tickets = {};
  String? _identity;
  String? _cursor;
  String? _error;
  bool _busy = false;
  WebSocketService? _realtime;
  Timer? _refreshDelay, _fallbackRefresh;
  static const _events = [
    'support_update',
    'notification_update',
    'resync',
    'connection',
  ];

  void _onSupportUpdate(Map<String, dynamic> event) {
    if (event['type'] == 'connection' && event['connected'] != true) return;
    _scheduleRefresh();
  }

  void _scheduleRefresh() {
    if (!mounted || _refreshDelay?.isActive == true) return;
    _refreshDelay = Timer(const Duration(milliseconds: 200), () {
      if (!mounted ||
          WidgetsBinding.instance.lifecycleState != AppLifecycleState.resumed)
        return;
      if (_busy) {
        _scheduleRefresh();
        return;
      }
      unawaited(_refresh());
    });
  }

  void _attachRealtime() {
    final service = context.read<WebSocketService>();
    if (identical(service, _realtime)) return;
    for (final event in _events) {
      _realtime?.off(event, _onSupportUpdate);
    }
    _realtime = service;
    for (final event in _events) {
      service.on(event, _onSupportUpdate);
    }
    _fallbackRefresh ??= Timer.periodic(
      const Duration(seconds: 30),
      (_) => _scheduleRefresh(),
    );
  }

  @override
  void dispose() {
    _refreshDelay?.cancel();
    _fallbackRefresh?.cancel();
    for (final event in _events) {
      _realtime?.off(event, _onSupportUpdate);
    }
    super.dispose();
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _attachRealtime();
  }

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) _refresh();
    });
  }

  Future<void> _refresh({bool more = false}) async {
    final api = context.read<ApiService>();
    final identity = api.authIdentity;
    if (_busy ||
        identity == null ||
        (more && (_cursor == null || _identity != identity)))
      return;
    setState(() {
      _busy = true;
      _error = null;
      if (_identity != identity) {
        _tickets.clear();
        _cursor = null;
      }
      _identity = identity;
    });
    try {
      final page = await SupportService(
        api,
      ).getUserTickets(before: more ? _cursor : null);
      if (!mounted || api.authIdentity != identity) return;
      setState(() {
        if (!more) _tickets.clear();
        for (final ticket in page.tickets) {
          _tickets[ticket.ticketId] = ticket;
        }
        _cursor = page.nextCursor;
      });
    } catch (_) {
      if (mounted && api.authIdentity == identity)
        setState(() => _error = 'Could not load tickets.');
    } finally {
      if (mounted && api.authIdentity == identity)
        setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final tickets = _identity == context.read<ApiService>().authIdentity
        ? _tickets.values.toList()
        : <SupportTicket>[];
    tickets.sort((a, b) => b.updatedAt.compareTo(a.updatedAt));
    return OrixPage(
      title: 'Support conversations',
      description: 'Your questions and replies from the support team.',
      children: [
        FilledButton(
          onPressed: () => context.push(RoutePaths.createTicket),
          child: const Text('New conversation'),
        ),
        const SizedBox(height: 24),
        if (_busy) const PixelLoader(),
        if (_error != null) Text(_error!),
        if (tickets.isEmpty && !_busy && _error == null)
          const PixelSection(child: Text('No support conversations yet.')),
        for (final ticket in tickets)
          PixelSection(
            child: ListTile(
              contentPadding: EdgeInsets.zero,
              onTap: () async {
                await context.push(
                  '/profile/support/tickets/${Uri.encodeComponent(ticket.ticketId)}',
                );
                if (mounted) _refresh();
              },
              title: Text(
                ticket.subject,
                style: Theme.of(context).textTheme.titleMedium,
              ),
              subtitle: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const SizedBox(height: 8),
                  Text(
                    '${ticket.status.replaceAll('_', ' ')} · ${DateFormat('d MMM, h:mm a').format(ticket.updatedAt.toLocal())}',
                  ),
                  if (ticket.messages.isNotEmpty) ...[
                    const SizedBox(height: 8),
                    Text(
                      ticket.messages.last.content,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ],
                ],
              ),
              trailing: const Icon(Icons.chevron_right),
            ),
          ),
        if (_cursor != null && tickets.isNotEmpty)
          TextButton(
            onPressed: _busy ? null : () => _refresh(more: true),
            child: const Text('Earlier conversations'),
          ),
        TextButton(
          onPressed: _busy ? null : _refresh,
          child: const Text('Refresh'),
        ),
      ],
    );
  }
}
