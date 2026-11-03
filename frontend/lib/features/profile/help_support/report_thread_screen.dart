import 'package:orix/core/services/websocket_service.dart';
import 'dart:async';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:orix/core/services/api_service.dart';
import 'package:orix/core/theme/app_theme.dart';
import 'package:orix/core/widgets/pixel_ui.dart';
import 'package:orix/core/utils/time_utils.dart';

class ReportThreadScreen extends StatefulWidget {
  final String reportId;
  const ReportThreadScreen({super.key, required this.reportId});
  @override
  State<ReportThreadScreen> createState() => _ReportThreadScreenState();
}

class _ReportThreadScreenState extends State<ReportThreadScreen>
    with WidgetsBindingObserver {
  final _draft = TextEditingController();
  final _scroll = ScrollController();
  Map<String, dynamic>? _report;
  List<Map<String, dynamic>> _messages = [];
  String? _identity, _error, _retryID, _retryText;
  bool _loading = false, _sending = false;
  int _version = 0;
  Timer? _poll;
  late WebSocketService _socket;
  void _onRealtime(Map<String, dynamic> event) {
    if (event['type'] == 'connection' && event['connected'] != true) return;
    if (mounted && ModalRoute.of(context)?.isCurrent == true)
      _load(quiet: true);
  }

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _socket = context.read<WebSocketService>();
    for (final type in [
      'connection',
      'notification_update',
      'report_update',
      'resync',
    ]) {
      _socket.on(type, _onRealtime);
    }
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) _load();
    });
    _startPoll();
  }

  void _startPoll() {
    _poll?.cancel();
    _poll = Timer.periodic(const Duration(seconds: 20), (_) {
      if (mounted && ModalRoute.of(context)?.isCurrent == true)
        _load(quiet: true);
    });
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) {
      _startPoll();
      _load(quiet: true);
    } else {
      _poll?.cancel();
    }
  }

  @override
  void dispose() {
    _version++;
    _poll?.cancel();
    for (final type in [
      'connection',
      'notification_update',
      'report_update',
      'resync',
    ]) {
      _socket.off(type, _onRealtime);
    }
    WidgetsBinding.instance.removeObserver(this);
    _draft.dispose();
    _scroll.dispose();
    super.dispose();
  }

  bool _valid(ApiService api, String? identity, int version) =>
      mounted &&
      identity != null &&
      identity == api.authIdentity &&
      version == _version;
  void _apply(Map<String, dynamic> data) {
    _report = data['report'] as Map<String, dynamic>?;
    _messages = (data['messages'] as List<dynamic>? ?? [])
        .whereType<Map<String, dynamic>>()
        .toList();
  }

  Future<void> _load({bool quiet = false}) async {
    final api = context.read<ApiService>();
    final identity = api.authIdentity;
    if (identity == null || _loading || _sending) return;
    final version = ++_version;
    setState(() {
      _loading = true;
      _error = null;
      if (_identity != identity) {
        _report = null;
        _messages = [];
        _draft.clear();
      }
      _identity = identity;
    });
    try {
      final response = await api.get<Map<String, dynamic>>(
        '/reports/${Uri.encodeComponent(widget.reportId)}',
        forceRefresh: true,
      );
      if (!_valid(api, identity, version)) return;
      setState(() {
        if (response.success && response.data != null) {
          _apply(response.data!);
        } else {
          _error = response.error ?? 'Could not load report conversation.';
        }
      });
    } catch (_) {
      if (_valid(api, identity, version))
        setState(() => _error = 'Could not load report conversation.');
    } finally {
      if (_valid(api, identity, version)) setState(() => _loading = false);
    }
  }

  Future<void> _send() async {
    final text = _draft.text.trim();
    if (text.isEmpty || _sending || _loading || _report?['status'] != 'pending')
      return;
    final api = context.read<ApiService>();
    final identity = api.authIdentity;
    if (identity == null || identity != _identity) return;
    final version = ++_version;
    // Retain this ID after uncertain delivery so retrying cannot duplicate the reply.
    if (_retryText != text) {
      _retryText = text;
      _retryID = 'reply-${DateTime.now().microsecondsSinceEpoch}';
    }
    setState(() {
      _sending = true;
      _error = null;
    });
    try {
      final response = await api.post<Map<String, dynamic>>(
        '/reports/${Uri.encodeComponent(widget.reportId)}/messages',
        body: {'client_message_id': _retryID, 'content': text},
      );
      if (!_valid(api, identity, version)) return;
      if (response.success && response.data != null) {
        setState(() {
          _apply(response.data!);
          _draft.clear();
          _retryID = null;
          _retryText = null;
        });
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (mounted && _scroll.hasClients)
            _scroll.jumpTo(_scroll.position.maxScrollExtent);
        });
      } else {
        setState(
          () => _error =
              response.error ??
              'Reply not confirmed. Your draft is saved here.',
        );
      }
    } catch (_) {
      if (_valid(api, identity, version))
        setState(
          () => _error = 'Reply not confirmed. Retry with the same draft.',
        );
    } finally {
      if (_valid(api, identity, version)) setState(() => _sending = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final valid = _identity == context.read<ApiService>().authIdentity;
    final report = valid ? _report : null;
    final messages = valid ? _messages : <Map<String, dynamic>>[];
    final pending = report?['status'] == 'pending';
    return Scaffold(
      appBar: AppBar(
        title: const OrixLogo(width: 100),
        actions: [
          IconButton(
            tooltip: 'Refresh conversation',
            onPressed: _loading || _sending ? null : () => _load(),
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
                    itemCount: messages.length + 2,
                    itemBuilder: (context, index) {
                      if (index == 0) {
                        return Padding(
                          padding: const EdgeInsets.only(bottom: 28),
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.stretch,
                            children: [
                              Text(
                                'Report conversation',
                                style: Theme.of(
                                  context,
                                ).textTheme.headlineLarge,
                              ),
                              const SizedBox(height: 12),
                              if (report != null) ...[
                                Text(
                                  (report['category'] as String? ??
                                          'Your report')
                                      .replaceAll('_', ' '),
                                  style: Theme.of(
                                    context,
                                  ).textTheme.titleMedium,
                                ),
                                const SizedBox(height: 8),
                                Text(
                                  pending
                                      ? 'Private conversation with Orix moderation.'
                                      : 'Report ${report['status'] ?? 'closed'}.',
                                  style: const TextStyle(
                                    color: AppColors.textSecondary,
                                  ),
                                ),
                              ],
                              if (_error case final error?) ...[
                                const SizedBox(height: 16),
                                Semantics(liveRegion: true, child: Text(error)),
                                Align(
                                  alignment: Alignment.centerLeft,
                                  child: TextButton(
                                    onPressed: _loading || _sending
                                        ? null
                                        : () => _load(),
                                    child: const Text('Retry'),
                                  ),
                                ),
                              ],
                              if (_loading && messages.isEmpty)
                                const Padding(
                                  padding: EdgeInsets.symmetric(vertical: 32),
                                  child: PixelLoader(),
                                ),
                            ],
                          ),
                        );
                      }
                      if (index == messages.length + 1) {
                        return report != null && !pending
                            ? const Padding(
                                padding: EdgeInsets.symmetric(vertical: 16),
                                child: Text(
                                  'This conversation is closed for replies.',
                                ),
                              )
                            : const SizedBox.shrink();
                      }
                      final message = messages[index - 1];
                      final time = TimeUtils.tryParseUtc(message['created_at']);
                      final admin = message['is_admin'] == true;
                      return OrixMessageBubble(
                        author: admin ? 'Orix moderation' : 'You',
                        content: message['content'] as String? ?? '',
                        time: time == null
                            ? null
                            : TimeUtils.formatRelative(time),
                        mine: !admin,
                      );
                    },
                  ),
                ),
                if (pending) ...[
                  const Divider(height: 1),
                  Padding(
                    padding: const EdgeInsets.fromLTRB(16, 12, 16, 16),
                    child: Row(
                      crossAxisAlignment: CrossAxisAlignment.end,
                      children: [
                        Expanded(
                          child: TextField(
                            controller: _draft,
                            maxLength: 2000,
                            minLines: 1,
                            maxLines: 3,
                            enabled: !_sending,
                            decoration: const InputDecoration(
                              labelText: 'Reply to moderation',
                              counterText: '',
                            ),
                          ),
                        ),
                        const SizedBox(width: 12),
                        IconButton.filled(
                          onPressed: _sending || _loading ? null : _send,
                          tooltip: 'Send reply',
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
  }
}
