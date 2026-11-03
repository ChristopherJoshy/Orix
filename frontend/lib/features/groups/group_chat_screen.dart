/// Group Chat Screen
///
/// In-group chat with system messages using WebSocket for real-time updates.
library;

import 'package:orix/core/widgets/pixel_ui.dart';

import 'dart:async';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:go_router/go_router.dart';

import 'package:orix/core/providers/group_provider.dart';
import 'package:orix/core/theme/app_theme.dart';

class GroupChatScreen extends StatefulWidget {
  final String groupId;

  const GroupChatScreen({super.key, required this.groupId});

  @override
  State<GroupChatScreen> createState() => _GroupChatScreenState();
}

class _GroupChatScreenState extends State<GroupChatScreen>
    with WidgetsBindingObserver {
  final _messageController = TextEditingController();
  final _scrollController = ScrollController();
  Timer? _refreshTimer;
  int _pollTicks = 0;
  String? _lastMessageId;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      unawaited(_loadMessages());
      unawaited(context.read<GroupProvider>().fetchGroup(widget.groupId));
    });
    _startPolling();
  }

  bool get _visible =>
      mounted &&
      GoRouter.of(context).routeInformationProvider.value.uri.path ==
          '/groups/${widget.groupId}/chat';

  void _startPolling() {
    _refreshTimer?.cancel();

    // Periodically refresh messages in case WebSocket misses an event
    _refreshTimer = Timer.periodic(const Duration(seconds: 15), (_) {
      if (!_visible) return;
      _pollTicks++;
      if (!context.read<GroupProvider>().isRealtimeConnected ||
          _pollTicks.isEven)
        _loadMessages();
    });
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) {
      _startPolling();
      if (_visible) {
        unawaited(_loadMessages());
        unawaited(context.read<GroupProvider>().fetchGroup(widget.groupId));
      }
    } else if (state == AppLifecycleState.paused ||
        state == AppLifecycleState.inactive) {
      _refreshTimer?.cancel();
    }
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _refreshTimer?.cancel();
    _messageController.dispose();
    _scrollController.dispose();
    super.dispose();
  }

  Future<void> _loadMessages() async {
    await context.read<GroupProvider>().fetchMessages(widget.groupId);
  }

  Future<void> _sendMessage() async {
    final provider = context.read<GroupProvider>();
    final text = _messageController.text.trim();
    if (text.isEmpty ||
        provider.isSendingMessage ||
        provider.currentGroup?.canChat != true)
      return;
    if (text.runes.length > 1000) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Messages can contain up to 1000 characters.'),
        ),
      );
      return;
    }

    _messageController.clear();

    final success = await context.read<GroupProvider>().sendMessage(
      widget.groupId,
      text,
    );

    if (success && mounted) {
      // Scroll to bottom after sending
      Future.delayed(const Duration(milliseconds: 100), () {
        if (_scrollController.hasClients) {
          _scrollController.animateTo(
            _scrollController.position.maxScrollExtent,
            duration: const Duration(milliseconds: 200),
            curve: Curves.easeOut,
          );
        }
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Consumer<GroupProvider>(
      builder: (context, groupProvider, _) {
        final messages = groupProvider.messages;
        final latest = messages.isEmpty ? null : messages.last.messageId;
        if (latest != null && latest != _lastMessageId) {
          // Follow live arrivals only while the reader is already near the end.
          final follow =
              _lastMessageId == null ||
              !_scrollController.hasClients ||
              _scrollController.position.extentAfter < 100;
          _lastMessageId = latest;
          if (follow)
            WidgetsBinding.instance.addPostFrameCallback((_) {
              if (mounted && _scrollController.hasClients) {
                _scrollController.jumpTo(
                  _scrollController.position.maxScrollExtent,
                );
              }
            });
        }
        final isSending = groupProvider.isSendingMessage;
        final canChat =
            groupProvider.currentGroup?.groupId == widget.groupId &&
            groupProvider.currentGroup?.canChat == true;

        return Scaffold(
          appBar: AppBar(
            title: const OrixLogo(width: 100),
            actions: [
              IconButton(
                tooltip: 'Open trip details',
                onPressed: () => context.push(
                  '/groups/${Uri.encodeComponent(widget.groupId)}',
                ),
                icon: const Icon(Icons.route_outlined),
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
                    Expanded(child: _buildMessageList(messages, groupProvider)),
                    _buildInputBar(isSending, canChat),
                  ],
                ),
              ),
            ),
          ),
        );
      },
    );
  }

  Future<void> _loadEarlier(GroupProvider provider) async {
    final previousHeight = _scrollController.hasClients
        ? _scrollController.position.maxScrollExtent
        : 0.0;
    final previousOffset = _scrollController.hasClients
        ? _scrollController.offset
        : 0.0;
    await provider.loadOlderMessages(widget.groupId);
    if (!mounted) return;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted || !_scrollController.hasClients) return;
      _scrollController.jumpTo(
        (previousOffset +
                _scrollController.position.maxScrollExtent -
                previousHeight)
            .clamp(0.0, _scrollController.position.maxScrollExtent)
            .toDouble(),
      );
    });
  }

  Widget _buildMessageList(
    List<ChatMessage> messages,
    GroupProvider provider,
  ) => ListView.builder(
    controller: _scrollController,
    padding: const EdgeInsets.fromLTRB(20, 24, 20, 16),
    itemCount: messages.length + 1,
    itemBuilder: (context, index) {
      if (index == 0) {
        return Padding(
          padding: const EdgeInsets.only(bottom: 24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(
                'Group conversation',
                style: Theme.of(context).textTheme.headlineLarge,
              ),
              const SizedBox(height: 12),
              const Text('Plan your pickup and stay in touch with your group.'),
              if (!provider.isRealtimeConnected) ...[
                const SizedBox(height: 12),
                const Text(
                  'Reconnecting live updates…',
                  style: TextStyle(color: AppColors.textSecondary),
                ),
              ],
              if (provider.chatError case final error?) ...[
                const SizedBox(height: 12),
                Semantics(liveRegion: true, child: Text(error)),
                Align(
                  alignment: Alignment.centerLeft,
                  child: TextButton(
                    onPressed: provider.isLoadingMessages
                        ? null
                        : _loadMessages,
                    child: const Text('Refresh messages'),
                  ),
                ),
              ],
              if (provider.canLoadOlderMessages)
                Align(
                  alignment: Alignment.centerLeft,
                  child: TextButton(
                    onPressed: provider.isLoadingMessages
                        ? null
                        : () => _loadEarlier(provider),
                    child: const Text('Load earlier messages'),
                  ),
                ),
              if (messages.isEmpty) ...[
                const SizedBox(height: 24),
                if (provider.isLoadingMessages)
                  const PixelLoader()
                else
                  const Text('No messages yet.'),
              ],
            ],
          ),
        );
      }
      final message = messages[index - 1];
      return message.messageType == 'system'
          ? _buildSystemMessage(message)
          : _buildUserMessage(message);
    },
  );

  Widget _buildSystemMessage(ChatMessage msg) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Center(
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
          decoration: BoxDecoration(
            color: AppColors.surfaceVariant,
            borderRadius: BorderRadius.zero,
          ),
          child: Text(
            msg.content,
            style: Theme.of(
              context,
            ).textTheme.bodySmall?.copyWith(color: AppColors.textSecondary),
          ),
        ),
      ),
    );
  }

  Widget _buildUserMessage(ChatMessage message) => OrixMessageBubble(
    author: message.isMine
        ? 'You'
        : message.userDisplayName?.isNotEmpty == true
        ? message.userDisplayName!
        : 'Travel partner',
    content: message.content,
    time:
        '${message.createdAt.toLocal().hour.toString().padLeft(2, '0')}:${message.createdAt.toLocal().minute.toString().padLeft(2, '0')}',
    mine: message.isMine,
    status: message.isPending
        ? const Text(
            'Sending…',
            style: TextStyle(color: Colors.white70, fontSize: 11),
          )
        : message.failed && message.clientMessageId != null
        ? TextButton(
            onPressed: context.read<GroupProvider>().isSendingMessage
                ? null
                : () => context.read<GroupProvider>().retryMessage(
                    widget.groupId,
                    message.clientMessageId!,
                  ),
            style: TextButton.styleFrom(foregroundColor: Colors.white),
            child: const Text('Not confirmed · Retry'),
          )
        : null,
  );

  Widget _buildInputBar(bool isSending, bool canChat) {
    return SafeArea(
      child: Container(
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: AppColors.surface,
          border: Border(
            top: BorderSide(color: AppColors.outline.withAlphaValue(0.5)),
          ),
        ),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.end,
          children: [
            Expanded(
              child: TextField(
                controller: _messageController,
                enabled: canChat && !isSending,
                maxLength: 1000,
                decoration: InputDecoration(
                  labelText: canChat
                      ? 'Message your group'
                      : 'This group is read-only',
                  counterText: '',
                  border: OutlineInputBorder(
                    borderRadius: BorderRadius.zero,
                    borderSide: BorderSide.none,
                  ),
                  filled: true,
                  fillColor: AppColors.surfaceVariant,
                  contentPadding: const EdgeInsets.symmetric(
                    horizontal: 16,
                    vertical: 12,
                  ),
                ),
                minLines: 1,
                maxLines: 3,
                textInputAction: TextInputAction.newline,
              ),
            ),
            const SizedBox(width: 8),
            IconButton(
              onPressed: isSending || !canChat ? null : _sendMessage,
              icon: isSending
                  ? const SizedBox(width: 20, height: 20, child: PixelLoader())
                  : const Icon(Icons.send),
              style: IconButton.styleFrom(
                backgroundColor: AppColors.primary,
                foregroundColor: Colors.white,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
