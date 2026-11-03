import 'dart:async';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import 'package:orix/core/providers/auth_provider.dart';
import 'package:orix/core/router/app_router.dart';
import 'package:orix/core/widgets/pixel_ui.dart';

class SplashScreen extends StatefulWidget {
  const SplashScreen({super.key});
  @override
  State<SplashScreen> createState() => _SplashScreenState();
}

class _SplashScreenState extends State<SplashScreen> {
  AuthProvider? _auth;
  Timer? _timeout;
  bool _navigated = false;
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      _auth = context.read<AuthProvider>();
      _auth!.addListener(_check);
      _timeout = Timer(const Duration(seconds: 8), () {
        if (mounted && !_navigated) {
          _navigated = true;
          context.go(RoutePaths.login);
        }
      });
      _check();
    });
  }

  void _check() {
    final auth = _auth;
    if (!mounted || auth == null || auth.isLoading || _navigated) return;
    _navigated = true;
    _timeout?.cancel();
    context.go(
      auth.isBanned
          ? RoutePaths.banned
          : auth.isSuspended
          ? RoutePaths.suspended
          : !auth.isAuthenticated
          ? RoutePaths.login
          : !auth.isOnboarded
          ? RoutePaths.onboarding
          : RoutePaths.home,
    );
  }

  @override
  void dispose() {
    _auth?.removeListener(_check);
    _timeout?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => const Scaffold(
    body: SafeArea(
      child: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            OrixLogo(symbolOnly: true, width: 72),
            SizedBox(height: 24),
            PixelAnimation(
              scene: PixelScene.loading,
              size: 48,
              label: 'Loading',
            ),
          ],
        ),
      ),
    ),
  );
}
