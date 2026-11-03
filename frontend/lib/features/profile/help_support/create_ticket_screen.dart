import 'dart:convert';
import 'dart:math';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import 'package:orix/core/services/api_service.dart';
import 'package:orix/core/theme/app_theme.dart';
import 'package:orix/core/widgets/pixel_ui.dart';
import 'package:orix/core/utils/error_helper.dart';
import 'support_service.dart';
import 'email_person_picker.dart';

class CreateTicketScreen extends StatefulWidget {
  final String? type;
  const CreateTicketScreen({super.key, this.type});
  @override
  State<CreateTicketScreen> createState() => _CreateTicketScreenState();
}

class _CreateTicketScreenState extends State<CreateTicketScreen> {
  final _form = GlobalKey<FormState>();
  final _subject = TextEditingController();
  final _message = TextEditingController();
  late String _type;
  bool _busy = false;
  String? _error;
  String? _identity, _retryPayload, _retryId;
  @override
  void initState() {
    super.initState();
    _type = widget.type ?? 'issue';
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final identity = context.read<ApiService>().authIdentity;
    if (identity != _identity) {
      _identity = identity;
      _subject.clear();
      _message.clear();
      _type = widget.type ?? 'issue';
      _retryPayload = null;
      _retryId = null;
      _busy = false;
      _error = null;
    }
  }

  @override
  void dispose() {
    _subject.dispose();
    _message.dispose();
    super.dispose();
  }

  Future<void> _send() async {
    if (_busy || _form.currentState?.validate() != true) return;
    final api = context.read<ApiService>();
    final identity = api.authIdentity;
    if (identity == null) return;
    final payload = jsonEncode([
      _type,
      _subject.text.trim(),
      _message.text.trim(),
    ]);
    if (_retryPayload != payload || _retryId == null) {
      final random = Random.secure();
      _retryId = List.generate(
        16,
        (_) => random.nextInt(256).toRadixString(16).padLeft(2, '0'),
      ).join();
      _retryPayload = payload;
    }
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final ticket = await SupportService(api).createTicket(
        type: _type,
        subject: _subject.text.trim(),
        description: _message.text.trim(),
        clientRequestId: _retryId!,
      );
      if (!mounted || api.authIdentity != identity) return;
      context.pushReplacement(
        '/profile/support/tickets/${Uri.encodeComponent(ticket.ticketId)}',
      );
    } catch (error) {
      if (mounted && api.authIdentity == identity) {
        setState(() => _error = ErrorHelper.getUserFriendlyMessage(error));
      }
    } finally {
      if (mounted && api.authIdentity == identity) {
        setState(() => _busy = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_type == 'report_user') return const ReportUserScreen();
    return OrixPage(
      title: 'Contact support',
      description: 'Your message starts a conversation with the support team.',
      maxWidth: 640,
      children: [
        Form(
          key: _form,
          child: PixelSection(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                DropdownButtonFormField<String>(
                  initialValue: _type,
                  decoration: const InputDecoration(labelText: 'Topic'),
                  items: const [
                    DropdownMenuItem(
                      value: 'issue',
                      child: Text('Something isn’t working'),
                    ),
                    DropdownMenuItem(
                      value: 'feature',
                      child: Text('Suggest a change'),
                    ),
                  ],
                  onChanged: _busy
                      ? null
                      : (value) => setState(() => _type = value ?? 'issue'),
                ),
                const SizedBox(height: 24),
                TextFormField(
                  controller: _subject,
                  enabled: !_busy,
                  maxLength: 160,
                  decoration: const InputDecoration(labelText: 'Subject'),
                  validator: (value) => (value?.trim().length ?? 0) < 3
                      ? 'Enter a short subject.'
                      : null,
                ),
                const SizedBox(height: 16),
                TextFormField(
                  controller: _message,
                  enabled: !_busy,
                  maxLength: 2000,
                  minLines: 5,
                  maxLines: 10,
                  decoration: const InputDecoration(labelText: 'First message'),
                  validator: (value) => (value?.trim().length ?? 0) < 10
                      ? 'Add at least 10 characters of detail.'
                      : null,
                ),
              ],
            ),
          ),
        ),
        if (_error != null)
          Padding(
            padding: const EdgeInsets.only(bottom: 20),
            child: Text(_error!),
          ),
        FilledButton(
          onPressed: _busy ? null : _send,
          child: _busy
              ? const PixelButtonLoader()
              : const Text('Start conversation'),
        ),
        const SizedBox(height: 16),
        TextButton(
          onPressed: _busy ? null : () => setState(() => _type = 'report_user'),
          child: const Text('Report a safety concern instead'),
        ),
      ],
    );
  }
}

class ReportUserScreen extends StatefulWidget {
  final String? groupId, userId, displayName, email, photoUrl;
  const ReportUserScreen({
    super.key,
    this.groupId,
    this.userId,
    this.displayName,
    this.email,
    this.photoUrl,
  });
  @override
  State<ReportUserScreen> createState() => ReportUserScreenState();
}

class ReportUserScreenState extends State<ReportUserScreen> {
  final _note = TextEditingController();
  UserSearchResult? _person;
  String _reason = 'harassment';
  String? _error;
  bool _busy = false;
  @override
  void initState() {
    super.initState();
    if (widget.userId != null) {
      _person = UserSearchResult(
        userId: widget.userId!,
        displayName: widget.displayName ?? '',
        email: widget.email ?? '',
        photoUrl: widget.photoUrl,
      );
    }
  }

  @override
  void dispose() {
    _note.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    final person = _person, text = _note.text.trim();
    if (_busy || person == null) return;
    if (text.isEmpty) {
      setState(
        () => _error = 'Describe what happened to start the conversation.',
      );
      return;
    }
    final api = context.read<ApiService>();
    final identity = api.authIdentity;
    if (identity == null) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final response = await api.post<Map<String, dynamic>>(
        '/reports',
        body: {
          'accused_user_id': person.userId,
          'category': _reason,
          'note': text,
          if (widget.groupId != null) 'group_id': widget.groupId,
        },
      );
      if (!mounted || identity != api.authIdentity) return;
      final id = response.data?['report_id'];
      if (!response.success || id is! String) {
        setState(
          () => _error =
              response.error ?? 'Could not create report conversation.',
        );
        return;
      }
      context.pushReplacement(
        '/profile/support/reports/${Uri.encodeComponent(id)}',
      );
    } catch (_) {
      if (mounted && identity == api.authIdentity) {
        setState(
          () => _error =
              'Report not confirmed. Check your report history before retrying.',
        );
      }
    } finally {
      if (mounted && identity == api.authIdentity) {
        setState(() => _busy = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) => OrixPage(
    title: 'Start a report',
    description:
        'Tell the moderation team what happened. You can reply and add details in the conversation afterward.',
    maxWidth: 600,
    children: [
      PixelSection(
        title: 'Who is this about?',
        child: EmailPersonPicker(
          initial: _person,
          enabled: !_busy,
          onSelected: (person) => setState(() => _person = person),
        ),
      ),
      PixelSection(
        title: 'What happened?',
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            DropdownButtonFormField<String>(
              initialValue: _reason,
              decoration: const InputDecoration(labelText: 'Reason'),
              items: const [
                DropdownMenuItem(
                  value: 'harassment',
                  child: Text('Harassment'),
                ),
                DropdownMenuItem(
                  value: 'unsafe_driving',
                  child: Text('Unsafe driving'),
                ),
                DropdownMenuItem(
                  value: 'no_show',
                  child: Text('Missed pickup'),
                ),
                DropdownMenuItem(value: 'spam', child: Text('Spam')),
                DropdownMenuItem(value: 'other', child: Text('Other')),
              ],
              onChanged: _busy
                  ? null
                  : (value) => setState(() => _reason = value ?? _reason),
            ),
            const SizedBox(height: 20),
            TextField(
              controller: _note,
              enabled: !_busy,
              maxLength: 1000,
              minLines: 4,
              maxLines: 8,
              decoration: const InputDecoration(
                labelText: 'First message',
                hintText: 'Describe the incident and when it happened.',
              ),
            ),
            if (widget.groupId != null)
              const Text(
                'Recent messages from this group are included for review.',
                style: TextStyle(color: AppColors.textSecondary, fontSize: 12),
              ),
          ],
        ),
      ),
      if (_error != null)
        Padding(
          padding: const EdgeInsets.only(bottom: 16),
          child: Text(_error!),
        ),
      FilledButton(
        onPressed: _busy || _person == null ? null : _submit,
        child: _busy
            ? const PixelButtonLoader()
            : const Text('Start conversation'),
      ),
      const SizedBox(height: 16),
      const Text(
        'For immediate danger, contact local emergency services. Reports are not an emergency channel.',
        style: TextStyle(color: AppColors.textSecondary, fontSize: 12),
      ),
    ],
  );
}
