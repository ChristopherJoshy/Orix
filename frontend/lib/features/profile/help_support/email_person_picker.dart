import 'dart:async';
import 'package:flutter/material.dart';
import 'package:orix/core/widgets/account_photo.dart';
import 'package:provider/provider.dart';
import 'package:orix/core/services/api_service.dart';
import 'package:orix/core/widgets/pixel_ui.dart';
import 'package:orix/core/theme/app_theme.dart';
import 'support_service.dart';

/// Email is the visible identifier; mutations always use the returned immutable ID.
class EmailPersonPicker extends StatefulWidget {
  final ValueChanged<UserSearchResult?> onSelected;
  final UserSearchResult? initial;
  final bool enabled;
  const EmailPersonPicker({
    super.key,
    required this.onSelected,
    this.initial,
    this.enabled = true,
  });
  @override
  State<EmailPersonPicker> createState() => _EmailPersonPickerState();
}

class _EmailPersonPickerState extends State<EmailPersonPicker> {
  final _query = TextEditingController();
  Timer? _debounce;
  int _version = 0;
  bool _loading = false;
  String? _error;
  List<UserSearchResult> _results = [];
  UserSearchResult? _selected;
  @override
  void initState() {
    super.initState();
    _selected = widget.initial;
    _query.text = widget.initial?.email ?? '';
  }

  @override
  void dispose() {
    _version++;
    _debounce?.cancel();
    _query.dispose();
    super.dispose();
  }

  void _changed(String value) {
    _debounce?.cancel();
    _version++;
    setState(() {
      _selected = null;
      _results = [];
      _error = null;
      _loading = false;
    });
    widget.onSelected(null);
    if (value.trim().length >= 2)
      _debounce = Timer(const Duration(milliseconds: 320), _search);
  }

  Future<void> _search() async {
    _debounce?.cancel();
    if (!widget.enabled) return;
    final api = context.read<ApiService>(),
        identity = context.read<ApiService>().authIdentity;
    if (identity == null) return;
    final version = ++_version;
    final query = _query.text.trim();
    if (query.length < 2) return;
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final response = await api.get<List<dynamic>>(
        '/users/search',
        queryParams: {'query': query},
        forceRefresh: true,
      );
      if (!mounted || version != _version || identity != api.authIdentity)
        return;
      setState(() {
        _loading = false;
        if (!response.success) {
          _error = response.error ?? 'Could not search emails.';
          return;
        }
        _results = (response.data ?? [])
            .whereType<Map<String, dynamic>>()
            .map(UserSearchResult.fromJson)
            .where((person) => person.email.contains('@'))
            .toList();
        if (_results.isEmpty)
          _error = 'No matching email in your travel contacts.';
      });
    } catch (_) {
      if (mounted && version == _version && identity == api.authIdentity)
        setState(() {
          _loading = false;
          _error = 'Could not search emails. Try again.';
        });
    }
  }

  @override
  Widget build(BuildContext context) => Column(
    crossAxisAlignment: CrossAxisAlignment.stretch,
    children: [
      TextField(
        controller: _query,
        enabled: widget.enabled,
        keyboardType: TextInputType.emailAddress,
        autocorrect: false,
        maxLength: 100,
        decoration: const InputDecoration(
          labelText: 'Email address',
          hintText: 'Search your travel contacts',
          prefixIcon: Icon(Icons.search),
        ),
        onChanged: _changed,
        onSubmitted: (_) => _search(),
      ),
      if (_loading)
        const Align(
          alignment: Alignment.centerLeft,
          child: SizedBox(height: 48, width: 48, child: PixelLoader()),
        ),
      if (_error != null)
        Padding(
          padding: const EdgeInsets.only(bottom: 16),
          child: Text(
            _error!,
            style: const TextStyle(color: AppColors.textSecondary),
          ),
        ),
      if (_selected != null) PersonIdentity(person: _selected!),
      for (final person in _results)
        Padding(
          padding: const EdgeInsets.only(bottom: 8),
          child: OutlinedButton(
            onPressed: !widget.enabled
                ? null
                : () {
                    _debounce?.cancel();
                    _version++;
                    setState(() {
                      _selected = person;
                      _query.text = person.email;
                      _results = [];
                      _error = null;
                      _loading = false;
                    });
                    widget.onSelected(person);
                  },
            child: PersonIdentity(person: person),
          ),
        ),
    ],
  );
}

class PersonIdentity extends StatelessWidget {
  final UserSearchResult person;
  const PersonIdentity({super.key, required this.person});
  @override
  Widget build(BuildContext context) => Row(
    children: [
      AccountPhoto(url: person.photoUrl),
      const SizedBox(width: 14),
      Expanded(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              person.email.isEmpty ? 'Selected travel contact' : person.email,
              style: const TextStyle(fontWeight: FontWeight.w600),
              softWrap: true,
            ),
            if (person.displayName.isNotEmpty)
              Text(
                person.displayName,
                style: const TextStyle(
                  fontSize: 12,
                  color: AppColors.textSecondary,
                ),
              ),
          ],
        ),
      ),
    ],
  );
}
