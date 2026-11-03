import 'package:orix/core/widgets/orix_illustration.dart';
import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:provider/provider.dart';
import 'package:orix/core/providers/auth_provider.dart';
import 'package:orix/core/config/api_config.dart';
import 'package:orix/core/theme/app_theme.dart';
import 'package:orix/core/widgets/pixel_ui.dart';

class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key});
  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  Future<void> _showInternalLogin() async {
    final email = TextEditingController(text: 'traveler-a@orix.example');
    final password = TextEditingController();
    final provider = context.read<AuthProvider>();
    try {
      final accepted = await showDialog<bool>(
        context: context,
        builder: (dialogContext) => AlertDialog(
          title: const Text('Local test account'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: email,
                keyboardType: TextInputType.emailAddress,
                decoration: const InputDecoration(labelText: 'Test email'),
              ),
              TextField(
                controller: password,
                obscureText: true,
                autocorrect: false,
                enableSuggestions: false,
                decoration: const InputDecoration(labelText: 'Test password'),
              ),
            ],
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(dialogContext, false),
              child: const Text('Cancel'),
            ),
            FilledButton(
              onPressed: () => Navigator.pop(dialogContext, true),
              child: const Text('Sign in'),
            ),
          ],
        ),
      );
      if (accepted == true && mounted) {
        await provider.signInInternal(email.text.trim(), password.text);
      }
    } finally {
      email.dispose();
      password.dispose();
    }
  }

  @override
  Widget build(BuildContext context) {
    final auth = context.watch<AuthProvider>();
    return Scaffold(
      body: SafeArea(
        child: OrixWelcomeLayout(
          header: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const OrixLogo(width: 132),
              const SizedBox(height: 28),
              Text(
                'Go together.',
                style: Theme.of(context).textTheme.headlineLarge,
              ),
              const SizedBox(height: 12),
              const Text(
                'A familiar route. A little company.\nSign in to find your people.',
                style: TextStyle(color: AppColors.textSecondary),
              ),
            ],
          ),
          controls: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              if (auth.error != null)
                Padding(
                  padding: const EdgeInsets.only(bottom: 20),
                  child: Semantics(liveRegion: true, child: Text(auth.error!)),
                ),
              FilledButton(
                onPressed: auth.isLoading
                    ? null
                    : () => InternalTestConfig.enabled
                          ? _showInternalLogin()
                          : context.read<AuthProvider>().signInWithGoogle(),
                child: auth.isLoading
                    ? const PixelButtonLoader()
                    : Row(
                        mainAxisAlignment: MainAxisAlignment.center,
                        children: [
                          if (!InternalTestConfig.enabled) ...[
                            SvgPicture.asset(
                              'assets/images/google_logo.svg',
                              width: 20,
                              height: 20,
                              colorFilter: const ColorFilter.mode(
                                Colors.white,
                                BlendMode.srcIn,
                              ),
                            ),
                            const SizedBox(width: 12),
                          ],
                          Flexible(
                            child: Text(
                              InternalTestConfig.enabled
                                  ? 'Internal test account'
                                  : 'Continue with Google',
                            ),
                          ),
                        ],
                      ),
              ),
              const SizedBox(height: 20),
              const Text(
                'Review the community terms and privacy notice during setup.',
                style: TextStyle(
                  color: AppColors.textSecondary,
                  fontSize: 12,
                  height: 1.7,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
