import 'dart:math';
import 'dart:async';
import 'package:orix/core/services/websocket_service.dart';
import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'package:provider/provider.dart';
import 'package:orix/core/services/api_service.dart';
import 'package:orix/core/theme/app_theme.dart';
import 'package:orix/core/utils/error_helper.dart';
import 'package:orix/core/widgets/custom_alert_dialog.dart';
import 'package:orix/core/widgets/pixel_ui.dart';
import 'support_service.dart';

class TicketChatScreen extends StatefulWidget {
  final String ticketId;
  const TicketChatScreen({super.key, required this.ticketId});

  @override
  State<TicketChatScreen> createState() => _TicketChatScreenState();
}

class _TicketChatScreenState extends State<TicketChatScreen> {
  final _draft = TextEditingController();
  final _scroll = ScrollController();
  SupportTicket? _ticket;
  String? _identity, _error;
  bool _loading = true, _sending = false;
  int _version = 0;
  String? _retryText, _retryId;

  bool get _closed => const {'closed', 'resolved'}.contains(_ticket?.status);

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
      if (_loading || _sending) {
        _scheduleRefresh();
        return;
      }
      unawaited(_load());
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
  void didChangeDependencies() {
    _attachRealtime();
    super.didChangeDependencies();
    final identity = context.read<ApiService>().authIdentity;
    if (_identity == identity && identity != null) return;
    _identity = identity;
    _version++;
    _ticket = null;
    _draft.clear();
    _retryText = null;
    _retryId = null;
    _sending = false;
    _load();
  }

  @override
  void dispose() {
    _refreshDelay?.cancel();
    _fallbackRefresh?.cancel();
    for (final event in _events) {
      _realtime?.off(event, _onSupportUpdate);
    }

    _version++;
    _draft.dispose();
    _scroll.dispose();
    super.dispose();
  }

  bool _valid(String? identity, int version) =>
      mounted &&
      identity != null &&
      identity == _identity &&
      identity == context.read<ApiService>().authIdentity &&
      version == _version;

  void _followMessages() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted && _scroll.hasClients) {
        _scroll.jumpTo(_scroll.position.maxScrollExtent);
      }
    });
  }

  Future<void> _load() async {
    final api = context.read<ApiService>();
    final identity = api.authIdentity;
    if (identity == null) return;
    final version = ++_version;
    final follow =
        _ticket == null ||
        !_scroll.hasClients ||
        _scroll.position.extentAfter < 100;
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final ticket = await SupportService(api).getTicket(widget.ticketId);
      if (!_valid(identity, version)) return;
      setState(() => _ticket = ticket);
      if (follow) _followMessages();
    } catch (error) {
      if (_valid(identity, version)) {
        setState(() => _error = ErrorHelper.getUserFriendlyMessage(error));
      }
    } finally {
      if (_valid(identity, version)) setState(() => _loading = false);
    }
  }

  Future<void> _send() async {
    final text = _draft.text.trim();
    if (text.isEmpty || _sending || _loading || _ticket == null || _closed)
      return;
    final api = context.read<ApiService>();
    final identity = api.authIdentity;
    if (identity == null || identity != _identity) return;
    if (_retryText != text || _retryId == null) {
      final random = Random.secure();
      _retryId = List.generate(
        16,
        (_) => random.nextInt(256).toRadixString(16).padLeft(2, '0'),
      ).join();
      _retryText = text;
    }
    final version = ++_version;
    setState(() {
      _sending = true;
      _error = null;
    });
    try {
      final ticket = await SupportService(
        api,
      ).sendMessage(widget.ticketId, text, clientMessageId: _retryId!);
      if (!_valid(identity, version)) return;
      setState(() {
        _ticket = ticket;
        _draft.clear();
        _retryText = null;
        _retryId = null;
      });
      _followMessages();
    } catch (error) {
      if (_valid(identity, version)) {
        setState(() => _error = ErrorHelper.getUserFriendlyMessage(error));
      }
    } finally {
      if (_valid(identity, version)) setState(() => _sending = false);
    }
  }

  Future<void> _changeStatus({required bool close}) async {
    if (_loading || _sending || _ticket == null) return;
    final api = context.read<ApiService>();
    final identity = api.authIdentity;
    if (identity == null || identity != _identity) return;
    if (close) {
      final confirmed = await CustomAlertDialog.show<bool>(
        context: context,
        title: 'Close conversation?',
        message: 'You can reopen this conversation if you need more help.',
        primaryButtonText: 'Close conversation',
        onPrimaryPressed: () => Navigator.pop(context, true),
        secondaryButtonText: 'Keep open',
        onSecondaryPressed: () => Navigator.pop(context, false),
      );
      if (!mounted || confirmed != true || identity != api.authIdentity) return;
    }
    final version = ++_version;
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final service = SupportService(api);
      final ticket = close
          ? await service.closeTicket(widget.ticketId)
          : await service.reopenTicket(widget.ticketId);
      if (_valid(identity, version)) setState(() => _ticket = ticket);
    } catch (error) {
      if (_valid(identity, version)) {
        setState(() => _error = ErrorHelper.getUserFriendlyMessage(error));
      }
    } finally {
      if (_valid(identity, version)) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(
      title: const OrixLogo(width: 100),
      actions: [
        IconButton(
          tooltip: 'Refresh conversation',
          onPressed: _loading || _sending ? null : _load,
          icon: const Icon(Icons.refresh),
        ),
      ],
    ),
    body: SafeArea(
      top: false,
      child: Align(
        alignment: Alignment.topCenter,
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 760),
          child: Column(
            children: [
              Expanded(
                child: ListView.builder(
                  controller: _scroll,
                  padding: const EdgeInsets.fromLTRB(20, 24, 20, 16),
                  itemCount: (_ticket?.messages.length ?? 0) + 1,
                  itemBuilder: (context, index) {
                    if (index == 0) return _header();
                    final message = _ticket!.messages[index - 1];
                    return OrixMessageBubble(
                      author: message.isAdmin ? 'Orix support' : 'You',
                      content: message.content,
                      time: DateFormat(
                        'd MMM, h:mm a',
                      ).format(message.timestamp.toLocal()),
                      mine: !message.isAdmin,
                    );
                  },
                ),
              ),
              if (_ticket != null && !_closed) ...[
                const Divider(height: 1),
                Padding(
                  padding: const EdgeInsets.fromLTRB(16, 12, 16, 16),
                  child: Row(
                    crossAxisAlignment: CrossAxisAlignment.end,
                    children: [
                      Expanded(
                        child: TextField(
                          controller: _draft,
                          enabled: !_sending,
                          minLines: 1,
                          maxLines: 3,
                          maxLength: 2000,
                          textCapitalization: TextCapitalization.sentences,
                          decoration: const InputDecoration(
                            labelText: 'Reply to support',
                            counterText: '',
                          ),
                        ),
                      ),
                      const SizedBox(width: 12),
                      IconButton.filled(
                        tooltip: 'Send reply',
                        onPressed: _loading || _sending ? null : _send,
                        icon: _sending
                            ? const SizedBox(
                                width: 24,
                                height: 24,
                                child: PixelLoader(),
                              )
                            : const Icon(Icons.arrow_upward),
                      ),
                    ],
                  ),
                ),
              ],
            ],
          ),
        ),
      ),
    ),
  );

  Widget _header() => Padding(
    padding: const EdgeInsets.only(bottom: 28),
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(
          'Support conversation',
          style: Theme.of(context).textTheme.headlineLarge,
        ),
        if (_ticket case final ticket?) ...[
          const SizedBox(height: 12),
          Text(ticket.subject, style: Theme.of(context).textTheme.titleMedium),
          const SizedBox(height: 8),
          Text(
            _closed
                ? 'This conversation is closed.'
                : 'Orix support can reply here.',
            style: const TextStyle(color: AppColors.textSecondary),
          ),
          Align(
            alignment: Alignment.centerLeft,
            child: TextButton(
              onPressed: _loading || _sending
                  ? null
                  : () => _changeStatus(close: !_closed),
              child: Text(
                _closed ? 'Reopen conversation' : 'Close conversation',
              ),
            ),
          ),
        ],
        if (_error case final error?) ...[
          const SizedBox(height: 12),
          Semantics(liveRegion: true, child: Text(error)),
          if (_ticket == null)
            Align(
              alignment: Alignment.centerLeft,
              child: TextButton(
                onPressed: _loading ? null : _load,
                child: const Text('Retry'),
              ),
            ),
        ],
        if (_loading && _ticket == null)
          const Padding(
            padding: EdgeInsets.symmetric(vertical: 32),
            child: PixelLoader(),
          ),
      ],
    ),
  );
}
