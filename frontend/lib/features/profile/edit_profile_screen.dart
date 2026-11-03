import 'package:orix/core/widgets/pixel_ui.dart';
import 'dart:io';

import 'package:orix/core/widgets/account_photo.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:image_picker/image_picker.dart';
import 'package:provider/provider.dart';

import 'package:orix/core/providers/auth_provider.dart';
import 'package:orix/core/providers/user_provider.dart';
import 'package:orix/core/services/api_service.dart';
import 'package:orix/core/theme/app_theme.dart';
import 'package:orix/core/utils/error_helper.dart';

class EditProfileScreen extends StatefulWidget {
  const EditProfileScreen({super.key});

  @override
  State<EditProfileScreen> createState() => _EditProfileScreenState();
}

class _EditProfileScreenState extends State<EditProfileScreen> {
  final _formKey = GlobalKey<FormState>();
  late TextEditingController _nameController;
  final ImagePicker _picker = ImagePicker();

  File? _imageFile;
  String? _imageUrl;
  bool _isLoading = false;

  @override
  void initState() {
    super.initState();
    final user = context.read<AuthProvider>().user;
    _nameController = TextEditingController(text: user?.displayName ?? '');
    _imageUrl = user?.photoUrl;
  }

  @override
  void dispose() {
    _nameController.dispose();
    super.dispose();
  }

  Future<void> _pickImage() async {
    try {
      final XFile? pickedFile = await _picker.pickImage(
        source: ImageSource.gallery,
        maxWidth: 1024,
        maxHeight: 1024,
        imageQuality: 85,
      );

      if (pickedFile != null && mounted) {
        setState(() {
          _imageFile = File(pickedFile.path);
        });
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(ErrorHelper.getUserFriendlyMessage(e))),
        );
      }
    }
  }

  Future<String?> _uploadImage(File image, String userId) async {
    try {
      final apiService = context.read<ApiService>();

      // Determine mime type
      String mediaType = 'image/jpeg';
      final path = image.path.toLowerCase();
      if (path.endsWith('.png')) {
        mediaType = 'image/png';
      } else if (path.endsWith('.webp')) {
        mediaType = 'image/webp';
      }

      final response = await apiService.postMultipart<Map<String, dynamic>>(
        '/users/me/avatar',
        file: image,
        fileField: 'file',
        mediaType: mediaType,
        parser: (json) => json as Map<String, dynamic>,
      );

      if (response.success && response.data != null) {
        final photoUrl = response.data!['photo_url'];

        return photoUrl;
      } else {
        throw Exception(response.error ?? 'Upload failed');
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(ErrorHelper.getUserFriendlyMessage(e))),
        );
      }
      return null;
    }
  }

  Future<void> _saveProfile() async {
    if (!_formKey.currentState!.validate()) return;

    setState(() => _isLoading = true);

    try {
      final userProvider = context.read<UserProvider>();
      final authProvider = context.read<AuthProvider>();
      final api = context.read<ApiService>();
      final identity = api.authIdentity;
      if (identity == null || authProvider.user == null)
        throw Exception('Sign in again to edit your profile');

      String? photoUrl = _imageUrl;

      if (_imageFile != null) {
        photoUrl = await _uploadImage(_imageFile!, authProvider.user!.userId);
        if (!mounted || api.authIdentity != identity) return;
        if (photoUrl == null) {
          throw Exception('Failed to upload image');
        }
        // Avatar upload is already committed; keep its result if the name save fails.
        setState(() {
          _imageUrl = photoUrl;
          _imageFile = null;
        });
      }

      final saved = await userProvider.updateProfile(
        displayName: _nameController.text.trim(),
        photoUrl: photoUrl,
      );

      if (!mounted || api.authIdentity != identity) return;
      if (!saved)
        throw Exception(userProvider.error ?? 'Could not update profile');

      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Profile updated successfully')),
        );
        context.pop();
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(ErrorHelper.getUserFriendlyMessage(e))),
        );
      }
    } finally {
      if (mounted) {
        setState(() => _isLoading = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) => OrixPage(
    title: 'Edit profile',
    description: 'Your name and photo help your group recognize you.',
    maxWidth: 600,
    children: [
      Form(
        key: _formKey,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            PixelSection(
              title: 'Photo',
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  SizedBox(
                    width: 96,
                    height: 96,
                    child: ClipPath(
                      clipper: const ShapeBorderClipper(shape: PixelBorder()),
                      child: _imageFile != null
                          ? ColorFiltered(
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
                              child: Image.file(_imageFile!, fit: BoxFit.cover),
                            )
                          : _imageUrl != null
                          ? AccountPhoto(url: _imageUrl, size: 96)
                          : const OrixLogo(symbolOnly: true),
                    ),
                  ),
                  const SizedBox(height: 20),
                  OutlinedButton.icon(
                    onPressed: _isLoading ? null : _pickImage,
                    icon: const Icon(
                      Icons.add_photo_alternate_outlined,
                      size: 18,
                    ),
                    label: const Text('Choose photo'),
                  ),
                ],
              ),
            ),
            PixelSection(
              title: 'Name',
              child: TextFormField(
                controller: _nameController,
                enabled: !_isLoading,
                maxLength: 100,
                decoration: const InputDecoration(labelText: 'Display name'),
                validator: (value) => value == null || value.trim().length < 2
                    ? 'Enter at least two characters.'
                    : null,
              ),
            ),
            FilledButton(
              onPressed: _isLoading ? null : _saveProfile,
              child: _isLoading
                  ? const PixelButtonLoader()
                  : const Text('Save changes'),
            ),
          ],
        ),
      ),
    ],
  );
}
