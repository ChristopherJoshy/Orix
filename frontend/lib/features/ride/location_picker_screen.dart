/// Explicit place search, optional GPS, or manual map/coordinate selection.
library;

import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:go_router/go_router.dart';
import 'package:latlong2/latlong.dart';
import 'package:provider/provider.dart';
import 'package:url_launcher/url_launcher.dart';

import 'package:orix/core/config/map_config.dart';
import 'package:orix/core/providers/location_picker_provider.dart';
import 'package:orix/core/services/api_service.dart';
import 'package:orix/core/services/place_service.dart';
import 'package:orix/core/theme/app_theme.dart';
import 'package:orix/core/widgets/pixel_ui.dart';

class LocationResult {
  final LatLng point;
  final String? address;
  final String? placeId;
  const LocationResult(this.point, this.address, {this.placeId});
}

class LocationPickerScreen extends StatefulWidget {
  final LocationResult? initial;
  const LocationPickerScreen({super.key, this.initial});
  @override
  State<LocationPickerScreen> createState() => _LocationPickerScreenState();
}

class _LocationPickerScreenState extends State<LocationPickerScreen> {
  final _map = MapController();
  final _search = TextEditingController();
  final _label = TextEditingController();
  late final LocationPickerProvider _model;
  late LatLng _center;
  bool _mapReady = false;
  Timer? _debounce;

  @override
  void initState() {
    super.initState();
    final initial = widget.initial;
    _center = initial?.point ?? const LatLng(20, 0);
    _model = LocationPickerProvider(
      PlaceService(context.read<ApiService>()),
      initial: initial == null
          ? null
          : PlaceSelection(
              initial.point,
              initial.address ?? 'Selected map point',
              id: initial.placeId,
            ),
    );
    _label.text = String.fromCharCodes(
      (_model.selection?.label ?? '').runes.take(200),
    );
  }

  @override
  void dispose() {
    _debounce?.cancel();
    _model.dispose();
    _map.dispose();
    _search.dispose();
    _label.dispose();
    super.dispose();
  }

  void _select(PlaceSelection place, {bool move = true}) {
    _debounce?.cancel();
    _center = place.point;
    _model.select(place);
    _label.text = String.fromCharCodes(place.label.runes.take(200));
    FocusScope.of(context).unfocus();
    if (move && _mapReady) _map.move(place.point, 16);
  }

  void _selectPoint(LatLng point) {
    _center = point;
    _model.selectPoint(point);
    _label.text = String.fromCharCodes(
      (_model.selection?.label ?? '').runes.take(200),
    );
  }

  Future<void> _useGPS() async {
    _debounce?.cancel();
    final place = await _model.useCurrentLocation();
    if (mounted && place != null) _select(place);
  }

  Future<void> _coordinates() async {
    _debounce?.cancel();
    final result = await showDialog<LatLng>(
      context: context,
      builder: (_) => _CoordinateDialog(initial: _model.selection?.point),
    );
    if (mounted && result != null)
      _select(PlaceSelection(result, 'Selected coordinates'));
  }

  void _confirm() {
    final place = _model.selection;
    if (place == null) return;
    final label = String.fromCharCodes(_label.text.trim().runes.take(200));
    context.pop(
      LocationResult(
        place.point,
        label.isEmpty ? place.coordinateLabel : label,
        placeId: place.id,
      ),
    );
  }

  Widget _searchPanel() {
    final results = _model.results;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(
          'Choose a place',
          style: Theme.of(context).textTheme.headlineLarge,
        ),
        const SizedBox(height: 12),
        const Text('Search for a place or choose a point on the map.'),
        const SizedBox(height: 24),
        TextField(
          controller: _search,
          maxLength: 200,
          textInputAction: TextInputAction.search,
          decoration: InputDecoration(
            labelText: 'Place name',
            counterText: '',
            hintText: 'Start typing a place name',
            suffixIcon: IconButton(
              tooltip: 'Search places',
              onPressed: _model.busy ? null : _searchNow,
              icon: const Icon(Icons.search),
            ),
          ),
          onChanged: (value) {
            _debounce?.cancel();
            _model.updateQuery(value);
            if (value.trim().length >= 3) {
              _debounce = Timer(const Duration(milliseconds: 500), () {
                if (mounted) _model.search();
              });
            }
          },
          onSubmitted: (_) => _searchNow(),
        ),
        if (_model.busy)
          const Align(
            alignment: Alignment.centerLeft,
            child: SizedBox(height: 48, width: 48, child: PixelLoader()),
          ),
        if (_model.error case final error?)
          Padding(
            padding: const EdgeInsets.only(top: 12),
            child: Semantics(liveRegion: true, child: Text(error)),
          ),
        if (results != null) ...[
          const SizedBox(height: 12),
          for (final place in results.places)
            Column(
              children: [
                ListTile(
                  contentPadding: const EdgeInsets.symmetric(vertical: 4),
                  leading: const Icon(Icons.place_outlined),
                  title: Text(place.label),
                  trailing: const Icon(Icons.add, size: 20),
                  onTap: () => _select(place),
                ),
                const Divider(height: 1),
              ],
            ),
          _attribution(results),
        ],
        const SizedBox(height: 16),
        Wrap(
          spacing: 12,
          runSpacing: 8,
          children: [
            OutlinedButton.icon(
              onPressed: _model.busy ? null : _useGPS,
              icon: const Icon(Icons.my_location, size: 20),
              label: const Text('Use my location'),
            ),
            TextButton(
              onPressed: _coordinates,
              child: const Text('Enter coordinates'),
            ),
          ],
        ),
      ],
    );
  }

  void _searchNow() {
    _debounce?.cancel();
    if (!_model.busy) _model.search();
  }

  Widget _attribution(PlaceSearchResults source) => Align(
    alignment: Alignment.centerLeft,
    child: TextButton(
      onPressed: source.attributionUrl == null
          ? null
          : () => launchUrl(
              source.attributionUrl!,
              mode: LaunchMode.externalApplication,
            ),
      child: Text(source.attribution),
    ),
  );

  Widget _mapPanel(double height) => Container(
    height: height,
    decoration: BoxDecoration(border: Border.all(color: AppColors.outline)),
    child: Stack(
      children: [
        FlutterMap(
          mapController: _map,
          options: MapOptions(
            initialCenter: _center,
            initialZoom: _model.selection == null ? 2 : 16,
            onMapReady: () => _mapReady = true,
            interactionOptions: const InteractionOptions(
              flags: InteractiveFlag.all & ~InteractiveFlag.rotate,
            ),
            onTap: (_, point) =>
                _select(PlaceSelection(point, 'Selected map point')),
            onPositionChanged: (camera, hasGesture) {
              if (hasGesture) {
                _debounce?.cancel();
                _selectPoint(camera.center);
              }
            },
          ),
          children: [MapConfig.tiles(), MapConfig.credits()],
        ),
        const IgnorePointer(
          child: Center(
            child: Padding(
              padding: EdgeInsets.only(bottom: 28),
              child: Icon(
                Icons.location_on,
                color: AppColors.primary,
                size: 36,
              ),
            ),
          ),
        ),
      ],
    ),
  );

  Widget _selectionPanel() {
    final selection = _model.selection;
    return PixelSection(
      title: 'Selected place',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (selection == null)
            const Text('Choose a suggestion or a map point to continue.')
          else ...[
            Text(
              selection.coordinateLabel,
              style: Theme.of(context).textTheme.bodySmall,
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _label,
              maxLength: 200,
              onChanged: _model.rename,
              decoration: const InputDecoration(
                labelText: 'Place or meeting landmark',
                counterText: '',
              ),
            ),
            if (_model.selectedSource case final source?) _attribution(source),
          ],
          const SizedBox(height: 20),
          FilledButton(
            onPressed: selection == null ? null : _confirm,
            child: const Text('Use this place'),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) => ListenableBuilder(
    listenable: _model,
    builder: (context, _) => Scaffold(
      appBar: AppBar(title: const OrixLogo(width: 100)),
      body: SafeArea(
        top: false,
        child: LayoutBuilder(
          builder: (context, constraints) {
            final wide = constraints.maxWidth >= 900;
            final mapHeight = (constraints.maxHeight * .65)
                .clamp(240.0, 520.0)
                .toDouble();
            final fields = Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                _searchPanel(),
                const SizedBox(height: 24),
                if (!wide) ...[
                  _mapPanel(mapHeight),
                  const SizedBox(height: 24),
                ],
                _selectionPanel(),
              ],
            );
            return SingleChildScrollView(
              keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
              padding: EdgeInsets.all(constraints.maxWidth < 360 ? 16 : 24),
              child: Align(
                alignment: Alignment.topCenter,
                child: ConstrainedBox(
                  constraints: BoxConstraints(maxWidth: wide ? 1160 : 680),
                  child: wide
                      ? Row(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Expanded(flex: 2, child: fields),
                            const SizedBox(width: 32),
                            Expanded(flex: 3, child: _mapPanel(mapHeight)),
                          ],
                        )
                      : fields,
                ),
              ),
            );
          },
        ),
      ),
    ),
  );
}

class _CoordinateDialog extends StatefulWidget {
  final LatLng? initial;
  const _CoordinateDialog({this.initial});
  @override
  State<_CoordinateDialog> createState() => _CoordinateDialogState();
}

class _CoordinateDialogState extends State<_CoordinateDialog> {
  final _form = GlobalKey<FormState>();
  late final TextEditingController _latitude;
  late final TextEditingController _longitude;
  @override
  void initState() {
    super.initState();
    _latitude = TextEditingController(
      text: widget.initial?.latitude.toString() ?? '',
    );
    _longitude = TextEditingController(
      text: widget.initial?.longitude.toString() ?? '',
    );
  }

  @override
  void dispose() {
    _latitude.dispose();
    _longitude.dispose();
    super.dispose();
  }

  String? _validate(String? value, double bound, String label) {
    final number = double.tryParse(value?.trim() ?? '');
    return number == null || !number.isFinite || number.abs() > bound
        ? 'Enter a valid $label'
        : null;
  }

  @override
  Widget build(BuildContext context) => AlertDialog(
    title: const Text('Enter coordinates'),
    content: SingleChildScrollView(
      child: Form(
        key: _form,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextFormField(
              controller: _latitude,
              decoration: const InputDecoration(
                labelText: 'Latitude (−90 to 90)',
              ),
              keyboardType: const TextInputType.numberWithOptions(
                decimal: true,
                signed: true,
              ),
              validator: (value) => _validate(value, 90, 'latitude'),
            ),
            TextFormField(
              controller: _longitude,
              decoration: const InputDecoration(
                labelText: 'Longitude (−180 to 180)',
              ),
              keyboardType: const TextInputType.numberWithOptions(
                decimal: true,
                signed: true,
              ),
              validator: (value) => _validate(value, 180, 'longitude'),
            ),
          ],
        ),
      ),
    ),
    actions: [
      TextButton(
        onPressed: () => Navigator.pop(context),
        child: const Text('Cancel'),
      ),
      FilledButton(
        onPressed: () {
          if (_form.currentState!.validate())
            Navigator.pop(
              context,
              LatLng(
                double.parse(_latitude.text.trim()),
                double.parse(_longitude.text.trim()),
              ),
            );
        },
        child: const Text('Use coordinates'),
      ),
    ],
  );
}
