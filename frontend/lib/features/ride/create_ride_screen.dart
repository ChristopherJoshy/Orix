/// Create Ride Screen
///
/// Map-based ride request creation.
library;

import 'package:orix/core/widgets/pixel_ui.dart';
import 'package:orix/core/widgets/place_autocomplete.dart';
import 'package:orix/core/services/place_service.dart';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import 'package:intl/intl.dart';
import 'package:latlong2/latlong.dart';

import 'package:orix/core/providers/ride_provider.dart';
import 'package:orix/core/providers/auth_provider.dart';
import 'package:orix/core/providers/group_provider.dart';
import 'package:orix/core/router/app_router.dart';
import 'package:orix/core/theme/app_theme.dart';
import 'package:orix/core/utils/error_helper.dart';

import 'package:orix/core/widgets/custom_alert_dialog.dart';
import 'package:orix/core/widgets/trip_route_map.dart';
import 'location_picker_screen.dart';

class CreateRideScreen extends StatefulWidget {
  final String initialMode;
  final bool offerLift;
  final RideRequest? editRequest;
  const CreateRideScreen({
    super.key,
    this.initialMode = 'pool',
    this.offerLift = false,
    this.editRequest,
  });

  @override
  State<CreateRideScreen> createState() => _CreateRideScreenState();
}

class _CreateRideScreenState extends State<CreateRideScreen> {
  final _formKey = GlobalKey<FormState>();

  // Location data
  String? _pickupLabel; // Matches backend field name, but UI will say "Start"
  double? _pickupLat;
  double? _pickupLng;

  String? _dropLabel;
  double? _dropLat;
  double? _dropLng;

  // Date and time
  DateTime _selectedDate = DateTime.now();
  late TimeOfDay _startTime;
  late TimeOfDay _endTime;

  // Preferences
  String? _pickupPlaceId, _dropPlaceId;
  String _mode = 'pool';
  bool _allowLift = false;
  bool _offeringLift = false;
  int _seats = 1;
  bool _femaleOnly = false;

  bool _isLoading = false;
  bool _isCheckingGroups = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _offeringLift = widget.offerLift;
    _mode = _offeringLift ? 'lift' : 'pool';
    var start = DateTime.now().add(const Duration(minutes: 30));
    if (start.hour == 23 && start.minute > 29)
      start = DateTime(start.year, start.month, start.day + 1, 0, 15);
    var end = start.add(const Duration(minutes: 30));
    final edit = widget.editRequest;
    if (edit != null) {
      start =
          edit.departureStart ??
          DateTime.parse('${edit.date}T${edit.timeWindow.start}');
      end =
          edit.departureEnd ??
          DateTime.parse('${edit.date}T${edit.timeWindow.end}');
      _offeringLift = edit.driver;
      _mode = edit.driver ? 'lift' : 'pool';
      _allowLift = edit.allowRiders;
      _seats = edit.seats.clamp(1, 3).toInt();
      _femaleOnly = edit.femaleOnly;
      _pickupPlaceId = edit.pickupPlaceId;
      _dropPlaceId = edit.dropPlaceId;
      _pickupLat = edit.pickupLat;
      _pickupLng = edit.pickupLng;
      _pickupLabel = edit.pickupLabel;
      _dropLat = edit.dropLat;
      _dropLng = edit.dropLng;
      _dropLabel = edit.dropLabel;
      _isCheckingGroups = false;
    } else {
      _checkActiveGroups();
    }
    _selectedDate = DateTime(start.year, start.month, start.day);
    _startTime = TimeOfDay.fromDateTime(start);
    _endTime = TimeOfDay.fromDateTime(end);
  }

  Future<void> _checkActiveGroups() async {
    final groupProvider = context.read<GroupProvider>();
    await groupProvider.fetchActiveGroups();

    if (!mounted) return;

    setState(() => _isCheckingGroups = false);

    if (groupProvider.hasActiveGroup) {
      final activeGroup = groupProvider.activeGroups.first;
      _showActiveGroupDialog(activeGroup);
    }
  }

  void _showActiveGroupDialog(RideGroup group) {
    CustomAlertDialog.show(
      context: context,
      barrierDismissible: false,
      title: 'Active Ride Exists',
      message:
          'You already have an active ride group:\n\n'
          '${group.routeSummary}\n'
          '${group.timeWindowStart != null ? "Time: ${group.timeWindowStart}" : ""}\n\n'
          'Please complete or cancel your current ride before creating a new one.',
      primaryButtonText: 'View Ride',
      onPrimaryPressed: () {
        Navigator.of(context).pop();
        context.push('/groups/${group.groupId}'); // Go to group details
      },
      secondaryButtonText: 'Go Home',
      onSecondaryPressed: () {
        Navigator.of(context).pop();
        context.go('/'); // Go to home
      },
      icon: Icons.directions_car_filled_outlined,
      iconColor: AppColors.primary,
    );
  }

  Future<void> _selectDate() async {
    final picked = await showDatePicker(
      context: context,
      initialDate: _selectedDate.isBefore(DateUtils.dateOnly(DateTime.now()))
          ? DateTime.now()
          : _selectedDate,
      firstDate: DateTime.now(),
      lastDate: DateTime.now().add(const Duration(days: 90)),
    );

    if (picked != null) {
      setState(() => _selectedDate = picked);
    }
  }

  Future<void> _selectStartTime() async {
    final picked = await showTimePicker(
      initialEntryMode: TimePickerEntryMode.input,
      context: context,
      initialTime: _startTime,
    );

    if (picked != null) {
      setState(() => _startTime = picked);
    }
  }

  Future<void> _pickLocation(bool isPickup) async {
    final result = await context.push<LocationResult>(
      RoutePaths.locationPicker,
      extra: isPickup
          ? (_pickupLat != null && _pickupLng != null
                ? LocationResult(
                    LatLng(_pickupLat!, _pickupLng!),
                    _pickupLabel,
                    placeId: _pickupPlaceId,
                  )
                : null)
          : (_dropLat != null && _dropLng != null
                ? LocationResult(
                    LatLng(_dropLat!, _dropLng!),
                    _dropLabel,
                    placeId: _dropPlaceId,
                  )
                : null),
    );

    if (result != null) {
      setState(() {
        if (isPickup) {
          _pickupPlaceId = result.placeId;
          _pickupLabel = result.address ?? 'Selected Location';
          _pickupLat = result.point.latitude;
          _pickupLng = result.point.longitude;
        } else {
          _dropPlaceId = result.placeId;
          _dropLabel = result.address ?? 'Selected Location';
          _dropLat = result.point.latitude;
          _dropLng = result.point.longitude;
        }
      });
    }
  }

  Future<void> _createRequest() async {
    if (_pickupLabel == null ||
        _dropLabel == null ||
        _pickupLat == null ||
        _pickupLng == null ||
        _dropLat == null ||
        _dropLng == null) {
      setState(() => _error = 'Please select start and drop locations');
      return;
    }

    // Coordinates identify locations; different places may share a label.
    if ((_pickupLat != null &&
        _dropLat != null &&
        _pickupLat == _dropLat &&
        _pickupLng == _dropLng)) {
      setState(() => _error = 'Start and drop locations cannot be the same');
      return;
    }

    setState(() {
      _isLoading = true;
      _error = null;
    });

    // Keep enough time to submit the selected pickup window.
    final selectedDateTime = DateTime(
      _selectedDate.year,
      _selectedDate.month,
      _selectedDate.day,
      _startTime.hour,
      _startTime.minute,
    );

    final now = DateTime.now();
    final difference = selectedDateTime.difference(now);

    if (difference.inMinutes < 1) {
      setState(() {
        _isLoading = false;
        _error = 'Ride time must be in the future';
      });
      return;
    }

    final dateStr = DateFormat('yyyy-MM-dd').format(_selectedDate);
    final startStr =
        '${_startTime.hour.toString().padLeft(2, '0')}:${_startTime.minute.toString().padLeft(2, '0')}';
    final endStr =
        '${_endTime.hour.toString().padLeft(2, '0')}:${_endTime.minute.toString().padLeft(2, '0')}';
    if (_endTime.hour * 60 + _endTime.minute <=
        _startTime.hour * 60 + _startTime.minute) {
      setState(() {
        _isLoading = false;
        _error = 'Choose an end time after the start on the selected day.';
      });
      return;
    }

    final endDateTime = DateTime(
      _selectedDate.year,
      _selectedDate.month,
      _selectedDate.day,
      _endTime.hour,
      _endTime.minute,
    );
    if (selectedDateTime.hour != _startTime.hour ||
        selectedDateTime.minute != _startTime.minute ||
        endDateTime.hour != _endTime.hour ||
        endDateTime.minute != _endTime.minute ||
        selectedDateTime.timeZoneOffset != endDateTime.timeZoneOffset) {
      setState(() {
        _isLoading = false;
        _error = 'Choose a pickup window within one local time offset.';
      });
      return;
    }

    final rideProvider = context.read<RideProvider>();

    try {
      final request = await rideProvider.createRideRequest(
        pickupPlaceId: _pickupPlaceId,
        dropPlaceId: _dropPlaceId,
        pickupLat: _pickupLat!,
        pickupLng: _pickupLng!,
        pickupLabel: _pickupLabel!,
        dropLat: _dropLat!,
        dropLng: _dropLng!,
        dropLabel: _dropLabel!,
        date: dateStr,
        timeWindowStart: startStr,
        timeWindowEnd: endStr,
        allowRiders: _allowLift || _offeringLift,
        mode: _mode,
        driver: _mode == 'lift' && _offeringLift,
        seats: _seats,
        femaleOnly: _femaleOnly,
        requestId: widget.editRequest?.requestId,
      );

      if (!mounted) return;

      if (request != null) {
        if (_offeringLift)
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(
              content: Row(
                children: [
                  PixelAnimation(
                    scene: PixelScene.liftOffered,
                    size: 80,
                    color: Colors.white,
                    label: 'Lift offered',
                  ),
                  SizedBox(width: 12),
                  Expanded(child: Text('Lift route registered.')),
                ],
              ),
            ),
          );
        context.go(RoutePaths.matching);
      } else {
        // Use error helper for user-friendly messages
        final errorMsg = rideProvider.error ?? ErrorMessages.unknownError;
        setState(() => _error = errorMsg);
      }
    } catch (e) {
      if (!mounted) return;
      setState(
        () => _error = 'Failed to create ride request. Please try again.',
      );
    } finally {
      if (mounted) {
        setState(() => _isLoading = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final gender = context.watch<AuthProvider>().user?.gender?.toLowerCase();
    final female = gender == 'f' || gender == 'female';
    Widget location(String title, String? value, bool pickup) =>
        PlaceAutocomplete(
          label: title,
          pickup: pickup,
          enabled: !_isLoading,
          selection: pickup
              ? (_pickupLat != null && _pickupLng != null
                    ? PlaceSelection(
                        LatLng(_pickupLat!, _pickupLng!),
                        _pickupLabel ?? '',
                        id: _pickupPlaceId,
                      )
                    : null)
              : (_dropLat != null && _dropLng != null
                    ? PlaceSelection(
                        LatLng(_dropLat!, _dropLng!),
                        _dropLabel ?? '',
                        id: _dropPlaceId,
                      )
                    : null),
          onMap: () => _pickLocation(pickup),
          onChanged: (place) => setState(() {
            if (pickup) {
              _pickupLat = place?.point.latitude;
              _pickupLng = place?.point.longitude;
              _pickupLabel = place?.label;
              _pickupPlaceId = place?.id;
            } else {
              _dropLat = place?.point.latitude;
              _dropLng = place?.point.longitude;
              _dropLabel = place?.label;
              _dropPlaceId = place?.id;
            }
          }),
        );
    Widget time(String title, TimeOfDay value, VoidCallback tap) =>
        OutlinedButton(
          onPressed: _isLoading ? null : tap,
          child: Row(
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      title,
                      style: const TextStyle(
                        fontSize: 12,
                        color: AppColors.textSecondary,
                      ),
                    ),
                    const SizedBox(height: 6),
                    Text(value.format(context)),
                  ],
                ),
              ),
              const Icon(Icons.schedule, size: 18),
            ],
          ),
        );
    return OrixPage(
      title: widget.editRequest != null
          ? 'Edit journey'
          : _offeringLift
          ? 'Offer a lift'
          : 'Where to?',
      description: _offeringLift
          ? 'Register the route and time you can share.'
          : 'Set your route, departure window and preferences.',
      maxWidth: 720,
      children: [
        if (_isCheckingGroups)
          const PixelLoader()
        else
          Form(
            key: _formKey,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                PixelSection(
                  title: 'Your route',
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      location('Starting from', _pickupLabel, true),
                      const SizedBox(height: 16),
                      location('Going to', _dropLabel, false),
                      if (_pickupLat != null && _dropLat != null) ...[
                        const SizedBox(height: 20),
                        TripRouteMap(
                          pickupLat: _pickupLat!,
                          pickupLng: _pickupLng!,
                          dropLat: _dropLat!,
                          dropLng: _dropLng!,
                        ),
                      ],
                    ],
                  ),
                ),
                PixelSection(
                  title: 'Departure',
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      OutlinedButton.icon(
                        onPressed: _isLoading ? null : _selectDate,
                        icon: const Icon(
                          Icons.calendar_today_outlined,
                          size: 18,
                        ),
                        label: Text(
                          DateFormat('EEE, d MMM yyyy').format(_selectedDate),
                        ),
                      ),
                      const SizedBox(height: 16),
                      time('Earliest pickup', _startTime, _selectStartTime),
                      const SizedBox(height: 16),
                      time('Latest pickup', _endTime, () async {
                        final value = await showTimePicker(
                          initialEntryMode: TimePickerEntryMode.input,
                          context: context,
                          initialTime: _endTime,
                        );
                        if (value != null && mounted)
                          setState(() => _endTime = value);
                      }),
                    ],
                  ),
                ),
                PixelSection(
                  title: 'Your preferences',
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      if (!_offeringLift)
                        PixelToggle(
                          contentPadding: EdgeInsets.zero,
                          title: const Text('Open to lifts'),
                          subtitle: const Text(
                            'Include compatible lift offers in automatic matching.',
                          ),
                          value: _allowLift,
                          onChanged: _isLoading
                              ? null
                              : (value) => setState(() => _allowLift = value),
                        ),
                      PixelToggle(
                        contentPadding: EdgeInsets.zero,
                        title: const Text('Women-only group'),
                        subtitle: Text(
                          female
                              ? 'Match with eligible women.'
                              : 'Available to eligible women.',
                        ),
                        value: female && _femaleOnly,
                        onChanged: !female || _isLoading
                            ? null
                            : (value) => setState(() => _femaleOnly = value),
                      ),
                      if (_offeringLift)
                        DropdownButtonFormField<int>(
                          initialValue: _seats,
                          decoration: const InputDecoration(
                            labelText: 'Passenger seats',
                          ),
                          items: List.generate(
                            3,
                            (index) => DropdownMenuItem(
                              value: index + 1,
                              child: Text(
                                '${index + 1} ${index == 0 ? 'seat' : 'seats'}',
                              ),
                            ),
                          ),
                          onChanged: _isLoading
                              ? null
                              : (value) => setState(() => _seats = value ?? 1),
                        ),
                    ],
                  ),
                ),
                if (_error != null)
                  Padding(
                    padding: const EdgeInsets.only(bottom: 20),
                    child: Text(
                      _error!,
                      style: const TextStyle(color: AppColors.textPrimary),
                    ),
                  ),
                FilledButton(
                  onPressed: _isLoading ? null : _createRequest,
                  child: _isLoading
                      ? const PixelButtonLoader()
                      : Text(
                          widget.editRequest != null
                              ? 'Save changes'
                              : _offeringLift
                              ? 'Register lift'
                              : 'Find my pool',
                        ),
                ),
              ],
            ),
          ),
      ],
    );
  }
}
