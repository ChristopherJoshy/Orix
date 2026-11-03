import 'dart:async';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:url_launcher/url_launcher.dart';
import '../services/api_service.dart';
import '../services/place_service.dart';
import '../theme/app_theme.dart';
import 'pixel_ui.dart';

/// Search results carry coordinates and a canonical place ID. Editing the text
/// immediately invalidates the previous selection; typed labels are never routes.
class PlaceAutocomplete extends StatefulWidget {
  final String label;
  final PlaceSelection? selection;
  final ValueChanged<PlaceSelection?> onChanged;
  final VoidCallback? onMap;
  final bool enabled;
  final bool pickup;
  const PlaceAutocomplete({
    super.key,
    required this.label,
    required this.selection,
    required this.onChanged,
    this.onMap,
    this.enabled = true,
    this.pickup = false,
  });
  @override
  State<PlaceAutocomplete> createState() => _PlaceAutocompleteState();
}

class _PlaceAutocompleteState extends State<PlaceAutocomplete> {
  late final TextEditingController _text;
  Timer? _debounce;
  int _version = 0;
  bool _busy = false;
  String? _error;
  PlaceSearchResults? _results;
  @override
  void initState() {
    super.initState();
    _text = TextEditingController(text: widget.selection?.label ?? '');
  }

  @override
  void didUpdateWidget(covariant PlaceAutocomplete old) {
    super.didUpdateWidget(old);
    if (widget.selection != null && widget.selection != old.selection) {
      _debounce?.cancel();
      _version++;
      _text.text = widget.selection!.label;
      _results = null;
      _error = null;
      _busy = false;
    }
  }

  @override
  void dispose() {
    _version++;
    _debounce?.cancel();
    _text.dispose();
    super.dispose();
  }

  void _changed(String value) {
    _debounce?.cancel();
    _version++;
    setState(() {
      _results = null;
      _error = null;
      _busy = false;
    });
    widget.onChanged(null);
    if (value.trim().length >= 3)
      _debounce = Timer(const Duration(milliseconds: 500), _search);
  }

  Future<void> _search() async {
    _debounce?.cancel();
    final query = _text.text.trim();
    if (query.length < 3 || !widget.enabled) return;
    final api = context.read<ApiService>();
    final identity = api.authIdentity;
    if (identity == null) return;
    final version = ++_version;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final result = await PlaceService(api).search(query);
      if (!mounted || version != _version || identity != api.authIdentity)
        return;
      setState(() {
        _results = result;
        if (result.places.isEmpty)
          _error = 'No places found. Include a city or area.';
      });
    } on PlaceLookupException catch (error) {
      if (mounted && version == _version && identity == api.authIdentity)
        setState(() => _error = error.message);
    } catch (_) {
      if (mounted && version == _version && identity == api.authIdentity)
        setState(() => _error = 'Could not search places. Try again.');
    } finally {
      if (mounted && version == _version && identity == api.authIdentity)
        setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) => Column(
    crossAxisAlignment: CrossAxisAlignment.stretch,
    children: [
      TextField(
        controller: _text,
        enabled: widget.enabled,
        maxLength: 200,
        decoration: InputDecoration(
          labelText: widget.label,
          hintText: 'Type a place or address',
          counterText: '',
          prefixIcon: Icon(
            widget.pickup ? Icons.trip_origin : Icons.location_on_outlined,
          ),
          suffixIcon: widget.onMap == null
              ? null
              : IconButton(
                  onPressed: widget.enabled ? widget.onMap : null,
                  tooltip: 'Choose on map',
                  icon: const Icon(Icons.map_outlined),
                ),
        ),
        onChanged: _changed,
        onSubmitted: (_) => _search(),
      ),
      if (_busy)
        const Align(
          alignment: Alignment.centerLeft,
          child: SizedBox(height: 48, width: 48, child: PixelLoader()),
        ),
      if (_error != null)
        Padding(
          padding: const EdgeInsets.symmetric(vertical: 12),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                _error!,
                style: const TextStyle(
                  color: AppColors.textSecondary,
                  fontSize: 12,
                ),
              ),
              TextButton(
                onPressed: _busy ? null : _search,
                child: const Text('Retry search'),
              ),
            ],
          ),
        ),
      if (_results != null && _results!.places.isNotEmpty)
        Container(
          margin: const EdgeInsets.only(top: 8),
          decoration: BoxDecoration(
            color: Colors.white,
            border: Border.all(color: AppColors.outline),
          ),
          child: Column(
            children: [
              for (final place in _results!.places)
                ListTile(
                  title: Text(
                    place.label,
                    style: const TextStyle(fontSize: 14),
                  ),
                  subtitle: Text(place.coordinateLabel),
                  leading: const Icon(Icons.location_on_outlined),
                  onTap: !widget.enabled
                      ? null
                      : () {
                          _debounce?.cancel();
                          _version++;
                          setState(() {
                            _text.text = place.label;
                            _results = null;
                            _busy = false;
                            _error = null;
                          });
                          widget.onChanged(place);
                        },
                ),
            ],
          ),
        ),
      if (_results?.attribution.isNotEmpty == true)
        TextButton(
          onPressed: _results?.attributionUrl == null
              ? null
              : () => launchUrl(
                  _results!.attributionUrl!,
                  mode: LaunchMode.externalApplication,
                ),
          child: Text(
            _results!.attribution,
            style: const TextStyle(fontSize: 11),
          ),
        ),
      if (widget.selection != null)
        Padding(
          padding: const EdgeInsets.only(top: 10),
          child: Text(
            widget.pickup
                ? 'Pickup area: within 300 m of this point.'
                : 'Destination: everyone must select this same place.',
            style: const TextStyle(
              fontSize: 12,
              color: AppColors.textSecondary,
            ),
          ),
        ),
    ],
  );
}
