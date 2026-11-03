import 'dart:async';
import '../../core/services/websocket_service.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import '../../core/services/api_service.dart';
import '../../core/services/place_service.dart';
import '../../core/widgets/pixel_ui.dart';
import '../../core/widgets/place_autocomplete.dart';
import 'package:latlong2/latlong.dart';

class RiderSetupScreen extends StatefulWidget {
  const RiderSetupScreen({super.key});
  @override
  State<RiderSetupScreen> createState() => _RiderSetupScreenState();
}

class _RiderSetupScreenState extends State<RiderSetupScreen> {
  final _zone = TextEditingController(text: 'Asia/Kolkata');
  PlaceSelection? _from, _to;
  TimeOfDay _time = const TimeOfDay(hour: 9, minute: 0);
  int _duration = 30, _seats = 2, _editRevision = 0;
  final Set<int> _days = {1, 2, 3, 4, 5};
  bool _women = false, _busy = false, _loading = true, _editing = false;
  bool _editActive = true;
  Map<String, dynamic>? _offer;
  String? _error, _identity;
  WebSocketService? _socket;
  Timer? _refreshTimer;
  bool _refreshPending = false;
  static const _events = ['recurring_lift_update', 'resync', 'connection'];
  void _onUpdate(Map<String, dynamic> event) {
    if (event['type'] == 'connection' && event['connected'] != true) return;
    _refreshPending = true;
    _queueRefresh();
  }

  void _queueRefresh() {
    if (!mounted || _refreshTimer?.isActive == true || !_refreshPending) return;
    _refreshTimer = Timer(const Duration(milliseconds: 250), () {
      if (!mounted) return;
      if (_busy || _loading || _editing) return;
      _refreshPending = false;
      unawaited(_load());
    });
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final socket = context.read<WebSocketService>();
    if (!identical(socket, _socket)) {
      for (final event in _events) {
        _socket?.off(event, _onUpdate);
      }
      _socket = socket;
      for (final event in _events) {
        socket.on(event, _onUpdate);
      }
    }
    final id = context.read<ApiService>().authIdentity;
    if (id != _identity) {
      _identity = id;
      _offer = null;
      _editing = false;
      _busy = false;
      _loading = id != null;
      _error = null;
      _from = null;
      _to = null;
      _time = const TimeOfDay(hour: 9, minute: 0);
      _duration = 30;
      _seats = 2;
      _women = false;
      _zone.text = 'Asia/Kolkata';
      _days
        ..clear()
        ..addAll({1, 2, 3, 4, 5});
      if (id != null) _load();
    }
  }

  @override
  void dispose() {
    _refreshTimer?.cancel();
    for (final event in _events) {
      _socket?.off(event, _onUpdate);
    }
    _zone.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    final api = context.read<ApiService>();
    final identity = api.authIdentity;
    if (identity == null) return;
    setState(() {
      _loading = true;
      _error = null;
    });
    final result = await api.get<Map<String, dynamic>>(
      '/lifts/recurring',
      forceRefresh: true,
    );
    if (!mounted || identity != api.authIdentity) return;
    setState(() {
      _loading = false;
      if (result.success) {
        _offer = result.data?['offer'] as Map<String, dynamic>?;
      } else {
        _error = result.error ?? 'Could not load your offer. Try again.';
      }
    });
    _queueRefresh();
  }

  void _edit() {
    final offer = _offer;
    _editRevision = (offer?['revision'] as num?)?.toInt() ?? 0;
    _editActive = offer == null || offer['active'] == true;
    if (offer != null) {
      final route = offer['route'] as Map<String, dynamic>;
      final schedule = offer['schedule'] as Map<String, dynamic>;
      _from = PlaceSelection(
        LatLng(
          (route['pickup_lat'] as num).toDouble(),
          (route['pickup_lng'] as num).toDouble(),
        ),
        route['pickup_label'] as String,
        id: route['pickup_place_id'] as String?,
      );
      _to = PlaceSelection(
        LatLng(
          (route['drop_lat'] as num).toDouble(),
          (route['drop_lng'] as num).toDouble(),
        ),
        route['drop_label'] as String,
        id: route['drop_place_id'] as String?,
      );
      final start = schedule['start_minute'] as int;
      _time = TimeOfDay(hour: start ~/ 60, minute: start % 60);
      _duration = (schedule['end_minute'] as int) - start;
      _seats = route['seats'] as int;
      _women = route['female_only'] as bool;
      _zone.text = schedule['time_zone'] as String;
      _days
        ..clear()
        ..addAll((schedule['weekdays'] as List).cast<int>());
    }
    setState(() {
      _editing = true;
      _error = null;
    });
  }

  Future<void> _put(Map<String, dynamic> offer) async {
    if (_busy) return;
    final api = context.read<ApiService>();
    final id = api.authIdentity;
    if (id == null) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final response = await api.put<Map<String, dynamic>>(
        '/lifts/recurring',
        body: offer,
      );
      if (!mounted || id != api.authIdentity) return;
      if (!response.success) {
        setState(
          () => _error =
              response.error ?? 'Could not save your offer. Try again.',
        );
        return;
      }
      setState(() {
        _offer = response.data?['offer'] as Map<String, dynamic>?;
        _editing = false;
      });
    } catch (_) {
      if (mounted && id == api.authIdentity) {
        setState(() => _error = 'Could not save your offer. Try again.');
      }
    } finally {
      if (mounted && id == api.authIdentity) {
        setState(() => _busy = false);
        _queueRefresh();
      }
    }
  }

  void _save() {
    final from = _from, to = _to;
    final start = _time.hour * 60 + _time.minute;
    if (from == null ||
        to == null ||
        from.id == null ||
        to.id == null ||
        from.id == to.id ||
        _days.isEmpty ||
        _zone.text.trim().isEmpty ||
        start + _duration >= 1440) {
      setState(
        () => _error =
            'Choose different places, travel days and a window within the same day.',
      );
      return;
    }
    _put({
      'active': _editActive,
      'revision': _editRevision,
      'route': {
        'pickup_place_id': from.id,
        'drop_place_id': to.id,
        'pickup_label': from.label,
        'drop_label': to.label,
        'pickup_lat': from.point.latitude,
        'pickup_lng': from.point.longitude,
        'drop_lat': to.point.latitude,
        'drop_lng': to.point.longitude,
        'date': '',
        'time_window_start': '',
        'time_window_end': '',
        'timezone_offset_minutes': 0,
        'female_only': _women,
        'allow_riders': true,
        'mode': 'lift',
        'driver': true,
        'seats': _seats,
      },
      'schedule': {
        'time_zone': _zone.text.trim(),
        'start_minute': start,
        'end_minute': start + _duration,
        'weekdays': (_days.toList()..sort()),
      },
    });
  }

  @override
  Widget build(BuildContext context) {
    final offer = _offer;
    return OrixPage(
      title: 'Offer a lift',
      description: 'Share a seat on a route you already travel.',
      maxWidth: 640,
      children: [
        PixelSection(
          title: 'Recurring availability',
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Text(
                'Choose a route, days and departure window. An active offer creates a journey when you have no other active journey. Pausing does not cancel an existing journey.',
              ),
              const SizedBox(height: 16),
              if (_loading)
                const PixelLoader()
              else if (_error != null) ...[
                Text(_error!, semanticsLabel: _error),
                TextButton(
                  onPressed: _busy ? null : _load,
                  child: const Text('Retry'),
                ),
              ],
              if (!_loading && !_editing && offer == null && _error == null)
                OutlinedButton(
                  onPressed: _edit,
                  child: const Text('Set up recurring offer'),
                ),
              if (!_loading && !_editing && offer != null) ...[
                Text(
                  '${(offer['route'] as Map)['pickup_label']} → ${(offer['route'] as Map)['drop_label']}',
                ),
                Text(offer['active'] == true ? 'Active' : 'Paused'),
                Wrap(
                  spacing: 12,
                  children: [
                    OutlinedButton(
                      onPressed: _busy ? null : _edit,
                      child: const Text('Edit schedule'),
                    ),
                    OutlinedButton(
                      onPressed: _busy
                          ? null
                          : () => _put({
                              'route': offer['route'],
                              'schedule': offer['schedule'],
                              'active': offer['active'] != true,
                              'revision': offer['revision'] ?? 0,
                            }),
                      child: Text(
                        offer['active'] == true
                            ? 'Pause future offers'
                            : 'Resume offer',
                      ),
                    ),
                  ],
                ),
              ],
              if (_editing) ...[
                PlaceAutocomplete(
                  key: ValueKey('recurring-from-$_identity'),
                  label: 'Starting from',
                  selection: _from,
                  onChanged: (value) => setState(() => _from = value),
                  pickup: true,
                  enabled: !_busy,
                ),
                PlaceAutocomplete(
                  key: ValueKey('recurring-to-$_identity'),
                  label: 'Going to',
                  selection: _to,
                  onChanged: (value) => setState(() => _to = value),
                  enabled: !_busy,
                ),
                OutlinedButton(
                  onPressed: _busy
                      ? null
                      : () async {
                          final value = await showTimePicker(
                            context: context,
                            initialTime: _time,
                          );
                          if (value != null && mounted) {
                            setState(() => _time = value);
                          }
                        },
                  child: Text('Departure: ${_time.format(context)}'),
                ),
                DropdownButtonFormField<int>(
                  initialValue: _duration,
                  decoration: const InputDecoration(
                    labelText: 'Departure window',
                  ),
                  items: [15, 30, 60, 120]
                      .map(
                        (v) => DropdownMenuItem(
                          value: v,
                          child: Text('$v minutes'),
                        ),
                      )
                      .toList(),
                  onChanged: _busy
                      ? null
                      : (v) => setState(() => _duration = v!),
                ),
                DropdownButtonFormField<int>(
                  initialValue: _seats,
                  decoration: const InputDecoration(
                    labelText: 'Passenger seats',
                  ),
                  items: List.generate(
                    8,
                    (i) =>
                        DropdownMenuItem(value: i + 1, child: Text('${i + 1}')),
                  ),
                  onChanged: _busy ? null : (v) => setState(() => _seats = v!),
                ),
                TextField(
                  controller: _zone,
                  enabled: !_busy,
                  decoration: const InputDecoration(
                    labelText: 'IANA timezone',
                    helperText: 'For example, Asia/Kolkata',
                  ),
                ),
                Wrap(
                  spacing: 4,
                  children: List.generate(
                    7,
                    (i) => FilterChip(
                      label: Text(
                        ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'][i],
                      ),
                      selected: _days.contains(i),
                      onSelected: _busy
                          ? null
                          : (value) => setState(() {
                              if (value) {
                                _days.add(i);
                              } else {
                                _days.remove(i);
                              }
                            }),
                    ),
                  ),
                ),
                SwitchListTile(
                  title: const Text('Women-only group'),
                  value: _women,
                  onChanged: _busy
                      ? null
                      : (value) => setState(() => _women = value),
                ),
                FilledButton(
                  onPressed: _busy ? null : _save,
                  child: Text(
                    _busy
                        ? 'Saving…'
                        : _offer == null
                        ? 'Save and activate'
                        : 'Save changes',
                  ),
                ),
                TextButton(
                  onPressed: _busy
                      ? null
                      : () {
                          setState(() => _editing = false);
                          _queueRefresh();
                        },
                  child: const Text('Cancel'),
                ),
              ],
            ],
          ),
        ),
        PixelSection(
          title: 'One-time lift',
          child: FilledButton(
            onPressed: () => context.push('/ride/create?mode=lift&offer=true'),
            child: const Text('Register a lift route'),
          ),
        ),
      ],
    );
  }
}
