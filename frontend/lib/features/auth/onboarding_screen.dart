import 'package:orix/core/widgets/orix_illustration.dart';
import 'package:orix/core/widgets/account_photo.dart';
import 'dart:io';
import 'dart:typed_data';
import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:geolocator/geolocator.dart';
import 'package:provider/provider.dart';
import 'package:go_router/go_router.dart';
import 'package:orix/core/providers/auth_provider.dart';
import 'package:orix/core/services/api_service.dart';
import 'package:orix/core/services/notification_service.dart';
import 'package:orix/core/config/privacy_config.dart';
import 'package:orix/core/router/app_router.dart';
import 'package:orix/core/widgets/pixel_ui.dart';
import 'package:orix/core/widgets/orix_tour.dart';

class OnboardingScreen extends StatefulWidget {
  const OnboardingScreen({super.key});
  @override
  State<OnboardingScreen> createState() => _OnboardingScreenState();
}

class _OnboardingScreenState extends State<OnboardingScreen> {
  final _phone = TextEditingController();
  final _form = GlobalKey<FormState>();
  String? _gender, _error, _permissionNote;
  bool _safety = false, _privacy = false, _busy = false;
  bool _photoClear = false, _offerLifts = false, _uploaded = false;
  bool _permissionBusy = false;
  Uint8List? _preview;
  int _step = 0;
  String? _occupation;
  @override
  void dispose() {
    _phone.dispose();
    super.dispose();
  }

  Future<void> _photo() async {
    if (_busy) return;
    final api = context.read<ApiService>();
    final identity = api.authIdentity;
    final picked = await ImagePicker().pickImage(
      source: ImageSource.gallery,
      maxWidth: 1024,
      maxHeight: 1024,
      imageQuality: 85,
    );
    if (picked == null || !mounted || identity != api.authIdentity) return;
    final bytes = await picked.readAsBytes();
    if (!mounted || identity != api.authIdentity) return;
    if (bytes.length > 2 * 1024 * 1024) {
      setState(() => _error = 'Choose a photo smaller than 2 MB.');
      return;
    }
    setState(() {
      _busy = true;
      _error = null;
      _photoClear = false;
    });
    try {
      final response = await api.postMultipart<Map<String, dynamic>>(
        '/users/me/avatar',
        file: File(picked.path),
        fileField: 'file',
        mediaType: picked.path.toLowerCase().endsWith('.png')
            ? 'image/png'
            : 'image/jpeg',
        parser: (value) => value as Map<String, dynamic>,
      );
      if (!mounted || identity != api.authIdentity) return;
      if (!response.success) {
        setState(
          () => _error = response.error ?? 'Photo upload failed. Try again.',
        );
        return;
      }
      setState(() {
        _preview = bytes;
        _uploaded = true;
      });
    } catch (_) {
      if (mounted && identity == api.authIdentity)
        setState(() => _error = 'Photo upload was not confirmed. Try again.');
    } finally {
      if (mounted && identity == api.authIdentity)
        setState(() => _busy = false);
    }
  }

  Future<void> _permission(bool notifications) async {
    setState(() {
      _permissionBusy = true;
      _permissionNote = null;
    });
    try {
      final bool granted;
      if (notifications) {
        granted = await context.read<NotificationService>().requestPermission();
      } else {
        final permission = await Geolocator.requestPermission();
        granted =
            permission == LocationPermission.whileInUse ||
            permission == LocationPermission.always;
      }
      if (mounted)
        setState(
          () => _permissionNote = granted
              ? '${notifications ? 'Notifications' : 'Location'} enabled.'
              : 'Permission is off. You can continue and change it in device settings.',
        );
    } catch (_) {
      if (mounted)
        setState(
          () => _permissionNote =
              'Permission could not be enabled. You can continue without it.',
        );
    } finally {
      if (mounted) setState(() => _permissionBusy = false);
    }
  }

  Future<void> _finish() async {
    if (_busy ||
        !_safety ||
        !_privacy ||
        !_photoClear ||
        PrivacyConfig.policy == null)
      return;
    setState(() {
      _busy = true;
      _error = null;
    });
    final auth = context.read<AuthProvider>();
    try {
      final success = await auth.completeOnboarding(
        phone: _phone.text.trim().replaceAll(RegExp(r'[^+0-9]'), ''),
        gender: _gender,
        safetyConsent: true,
        offerLifts: _offerLifts,
        photoClear: true,
        tourVersion: 1,
        occupation: _occupation ?? 'other',
      );
      if (!mounted) return;
      if (success) {
        context.go(RoutePaths.home);
      } else {
        setState(
          () => _error = auth.error ?? 'Could not save setup. Try again.',
        );
      }
    } catch (_) {
      if (mounted)
        setState(
          () =>
              _error = 'Could not save setup. Check your connection and retry.',
        );
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final user = context.watch<AuthProvider>().user;
    final hasPhoto = _uploaded || user?.photoUrl?.isNotEmpty == true;
    return OrixPage(
      title: [
        'Your details',
        'Your photo',
        'Travel preferences',
        'Ready to travel',
      ][_step],
      description: 'Step ${_step + 1} of 4',
      maxWidth: 640,
      canPop: false,
      children: [
        if (_step == 0)
          const OrixIllustration(
            scene: OrixIllustrationScene.companions,
            height: 180,
          ),
        if (_step == 0)
          Form(
            key: _form,
            child: PixelSection(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(
                    user?.displayName ?? 'Your profile',
                    style: Theme.of(context).textTheme.titleLarge,
                  ),
                  const SizedBox(height: 8),
                  Text(user?.email ?? ''),
                  const SizedBox(height: 24),
                  DropdownButtonFormField<String>(
                    initialValue: _occupation,
                    decoration: const InputDecoration(labelText: 'I am a'),
                    validator: (value) => value == null
                        ? 'Choose student, teacher or other.'
                        : null,
                    items: const [
                      DropdownMenuItem(
                        value: 'student',
                        child: Text('Student'),
                      ),
                      DropdownMenuItem(
                        value: 'teacher',
                        child: Text('Teacher'),
                      ),
                      DropdownMenuItem(value: 'other', child: Text('Other')),
                    ],
                    onChanged: (value) =>
                        setState(() => _occupation = value ?? 'other'),
                  ),
                  const SizedBox(height: 24),
                  TextFormField(
                    controller: _phone,
                    keyboardType: TextInputType.phone,
                    decoration: const InputDecoration(
                      labelText: 'Phone number',
                      hintText: '+91 98765 43210',
                      helperText: 'Include your country code',
                    ),
                    validator: (value) =>
                        RegExp(r'^\+[1-9][0-9]{9,14}$').hasMatch(
                          (value ?? '').replaceAll(RegExp(r'[^+0-9]'), ''),
                        )
                        ? null
                        : 'Enter a phone number with country code.',
                  ),
                  const SizedBox(height: 24),
                  DropdownButtonFormField<String>(
                    initialValue: _gender,
                    isExpanded: true,
                    decoration: const InputDecoration(
                      labelText: 'Gender (optional)',
                    ),
                    items: const [
                      DropdownMenuItem(value: 'F', child: Text('Woman')),
                      DropdownMenuItem(value: 'M', child: Text('Man')),
                      DropdownMenuItem(
                        value: 'O',
                        child: Text('Another identity / prefer not to say'),
                      ),
                    ],
                    onChanged: (value) => setState(() => _gender = value),
                  ),
                  const SizedBox(height: 12),
                  const Text(
                    'Women-only matching depends on verified eligibility.',
                  ),
                ],
              ),
            ),
          ),
        if (_step == 1)
          PixelSection(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                const Text(
                  'Use a recent photo with your face clearly visible. Travel partners use it to recognise you and may decline a match if they cannot identify you.',
                ),
                const SizedBox(height: 24),
                if (_preview != null)
                  Align(
                    alignment: Alignment.centerLeft,
                    child: ColorFiltered(
                      colorFilter: const ColorFilter.matrix([
                        .2126,
                        .7152,
                        .0722,
                        0,
                        0,
                        .2126,
                        .7152,
                        .0722,
                        0,
                        0,
                        .2126,
                        .7152,
                        .0722,
                        0,
                        0,
                        0,
                        0,
                        0,
                        1,
                        0,
                      ]),
                      child: Image.memory(
                        _preview!,
                        width: 128,
                        height: 128,
                        fit: BoxFit.cover,
                      ),
                    ),
                  ),
                if (_preview == null && user?.photoUrl != null) ...[
                  Align(
                    alignment: Alignment.centerLeft,
                    child: AccountPhoto(url: user!.photoUrl, size: 128),
                  ),
                  const SizedBox(height: 16),
                  const Text('Replace this photo if your face is not clear.'),
                ],
                PixelAnimation(
                  scene: _busy
                      ? PixelScene.uploading
                      : _uploaded
                      ? PixelScene.uploaded
                      : PixelScene.found,
                  size: 160,
                ),
                OutlinedButton(
                  onPressed: _busy ? null : _photo,
                  child: const Text('Choose a clear photo'),
                ),
                CheckboxListTile(
                  contentPadding: EdgeInsets.zero,
                  controlAffinity: ListTileControlAffinity.leading,
                  title: const Text('My profile photo clearly shows my face'),
                  value: _photoClear,
                  onChanged: hasPhoto && !_busy
                      ? (value) => setState(() => _photoClear = value ?? false)
                      : null,
                ),
              ],
            ),
          ),
        if (_step == 2) ...[
          PixelSection(
            title: 'Lift offers',
            child: PixelToggle(
              title: const Text('I would like to offer lifts'),
              subtitle: const Text(
                'You will still register a route, time and seats before being matched as a driver.',
              ),
              value: _offerLifts,
              onChanged: (value) => setState(() => _offerLifts = value),
            ),
          ),
          PixelSection(
            title: 'Optional permissions',
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                const Text(
                  'Notifications tell you about matches and pickup updates. Location can fill your starting point while using the app. You can also select places manually.',
                ),
                const SizedBox(height: 16),
                OutlinedButton(
                  onPressed: _permissionBusy ? null : () => _permission(true),
                  child: const Text('Enable notifications'),
                ),
                const SizedBox(height: 12),
                OutlinedButton(
                  onPressed: _permissionBusy ? null : () => _permission(false),
                  child: const Text('Enable location'),
                ),
                if (_permissionNote != null)
                  Padding(
                    padding: const EdgeInsets.only(top: 12),
                    child: Semantics(
                      liveRegion: true,
                      child: Text(_permissionNote!),
                    ),
                  ),
              ],
            ),
          ),
        ],
        if (_step == 3) ...[
          OutlinedButton(
            onPressed: () => showOrixTour(context),
            child: const Text('Take the app tour'),
          ),
          const SizedBox(height: 24),
          PixelSection(
            title: 'Before your first journey',
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                const Text(
                  'Check the person and pickup plan before travelling. Agree on costs directly. Block contacts by email or start a private report conversation if you need help.',
                ),
                CheckboxListTile(
                  contentPadding: EdgeInsets.zero,
                  controlAffinity: ListTileControlAffinity.leading,
                  title: const Text('I agree to the community safety rules'),
                  value: _safety,
                  onChanged: (v) => setState(() => _safety = v ?? false),
                ),
                TextButton(
                  onPressed: PrivacyConfig.policy == null
                      ? null
                      : PrivacyConfig.openPolicy,
                  child: const Text('Read the privacy policy'),
                ),
                CheckboxListTile(
                  contentPadding: EdgeInsets.zero,
                  controlAffinity: ListTileControlAffinity.leading,
                  title: const Text('I have read the privacy policy'),
                  value: _privacy,
                  onChanged: PrivacyConfig.policy == null
                      ? null
                      : (v) => setState(() => _privacy = v ?? false),
                ),
              ],
            ),
          ),
        ],
        if (_error != null)
          Padding(
            padding: const EdgeInsets.only(bottom: 16),
            child: Semantics(liveRegion: true, child: Text(_error!)),
          ),
        FilledButton(
          onPressed:
              _busy ||
                  _permissionBusy ||
                  (_step == 3 &&
                      (!_safety || !_privacy || PrivacyConfig.policy == null))
              ? null
              : () {
                  if (_step == 0 && _form.currentState?.validate() != true)
                    return;
                  if (_step == 1 && (!hasPhoto || !_photoClear)) {
                    setState(
                      () => _error =
                          'Add or confirm a clear photo before continuing.',
                    );
                    return;
                  }
                  if (_step == 3) {
                    _finish();
                    return;
                  }
                  setState(() {
                    _step++;
                    _error = null;
                  });
                  if (_step == 3) {
                    WidgetsBinding.instance.addPostFrameCallback((_) {
                      if (mounted) showOrixTour(context);
                    });
                  }
                },
          child: _busy
              ? const PixelButtonLoader()
              : Text(_step == 3 ? 'Finish setup' : 'Continue'),
        ),
        if (_step > 0)
          TextButton(
            onPressed: _busy
                ? null
                : () => setState(() {
                    _step--;
                    _error = null;
                  }),
            child: const Text('Back'),
          ),
      ],
    );
  }
}
