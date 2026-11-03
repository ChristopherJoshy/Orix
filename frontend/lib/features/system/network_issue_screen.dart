/// Network Issue Screen
///
/// Displayed when network connectivity is lost.
library;

import 'package:orix/core/widgets/pixel_ui.dart';

import 'dart:async';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import 'package:orix/core/providers/auth_provider.dart';
import 'package:orix/core/providers/connectivity_provider.dart';

class NetworkIssueScreen extends StatefulWidget {
  final VoidCallback? onRetry;

  const NetworkIssueScreen({super.key, this.onRetry});

  @override
  State<NetworkIssueScreen> createState() => _NetworkIssueScreenState();
}

class _NetworkIssueScreenState extends State<NetworkIssueScreen> {
  bool _isRetrying = false;
  ConnectivityProvider? _connectivityProvider;
  Timer? _retryTimer;

  @override
  void initState() {
    super.initState();
    // Listen for connectivity changes to auto-recover
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _listenToConnectivity();
      _startPeriodicRetry();
    });
  }

  void _listenToConnectivity() {
    _connectivityProvider = context.read<ConnectivityProvider>();
    _connectivityProvider?.addListener(_onConnectivityChange);
  }

  /// Start periodic retry every 5 seconds
  void _startPeriodicRetry() {
    _retryTimer = Timer.periodic(const Duration(seconds: 5), (_) {
      if (mounted && !_isRetrying) {
        _retryConnection();
      }
    });
  }

  void _onConnectivityChange() {
    if (!mounted) return;

    final isOnline = _connectivityProvider?.isOnline ?? false;
    debugPrint('Connectivity changed: isOnline=$isOnline');

    if (isOnline && !_isRetrying) {
      // Connection restored - try to refresh auth immediately
      debugPrint('Connection restored - attempting to reconnect');
      _retryConnection();
    }
  }

  Future<void> _retryConnection() async {
    if (_isRetrying) return;

    setState(() => _isRetrying = true);

    try {
      // First, check if we actually have connectivity
      if (!mounted) return;
      final connectivityProvider = context.read<ConnectivityProvider>();
      await connectivityProvider.refresh();

      if (!connectivityProvider.isOnline) {
        // Still offline - don't try to refresh auth
        if (mounted) {
          setState(() => _isRetrying = false);
        }
        return;
      }

      // We have connectivity - try to refresh auth
      if (!mounted) return;
      final authProvider = context.read<AuthProvider>();
      await authProvider.refreshUser();

      if (mounted) {
        // Success - clean up and navigate
        _retryTimer?.cancel();
        _connectivityProvider?.removeListener(_onConnectivityChange);
        _connectivityProvider?.clearOfflineFlag();

        // Navigate based on auth state
        if (authProvider.isAuthenticated) {
          context.go('/home');
        } else {
          context.go('/');
        }
      }
    } catch (e) {
      // Failed - will retry again in 5 seconds
      debugPrint('Connection retry failed: $e');
      if (mounted) {
        setState(() => _isRetrying = false);
      }
    }
  }

  @override
  void dispose() {
    _retryTimer?.cancel();
    _connectivityProvider?.removeListener(_onConnectivityChange);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => OrixPage(
    title: 'You’re offline.',
    description:
        'The network is on a chaya break. Check your connection and try again.',
    maxWidth: 600,
    children: [
      const PixelAnimation(
        scene: PixelScene.connectionLost,
        label: 'The network is on a chaya break',
      ),
      const SizedBox(height: 24),
      FilledButton(
        onPressed: _isRetrying ? null : _retryConnection,
        child: _isRetrying ? const PixelLoader() : const Text('Try again'),
      ),
      const SizedBox(height: 16),
      const Text(
        'We’ll also check for a connection while this screen is open.',
      ),
    ],
  );
}
