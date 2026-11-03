/// Bounded realtime notifications with fresh-token reconnect and REST resync hooks.
library;

import 'dart:async';
import 'dart:convert';
import 'dart:math' as math;

import 'package:web_socket_channel/web_socket_channel.dart';
import 'package:web_socket_channel/status.dart' as status;

import '../config/api_config.dart';
import 'websocket_transport_stub.dart'
    if (dart.library.io) 'websocket_transport_io.dart'
    if (dart.library.js_interop) 'websocket_transport_web.dart';

class WebSocketService {
  WebSocketChannel? _channel;
  StreamSubscription<dynamic>? _subscription;
  Timer? _pingTimer;
  Timer? _reconnectTimer;
  Future<String?> Function()? _tokenProvider;
  DateTime? _lastFrameAt;
  bool _connected = false;
  bool _connecting = false;
  bool _intentionalDisconnect = true;
  int _generation = 0;
  int _reconnectAttempts = 0;
  final _random = math.Random();
  final Map<String, Completer<Map<String, dynamic>>> _chatAcks = {};
  final Map<String, List<void Function(Map<String, dynamic>)>> _listeners = {};

  void Function(bool isConnected)? onConnectionChange;
  bool get isConnected => _connected;

  Future<void> connect(
    String token, {
    Future<String?> Function()? tokenProvider,
  }) async {
    _tokenProvider = tokenProvider ?? _tokenProvider;
    _intentionalDisconnect = false;
    if (_connected || _connecting) return;
    await _connect(token);
  }

  Future<void> _connect(String? initialToken) async {
    if (_intentionalDisconnect || _connected || _connecting) return;
    _connecting = true;
    final generation = ++_generation;
    WebSocketChannel? candidate;
    try {
      final token = initialToken ?? await _tokenProvider?.call();
      if (generation != _generation || _intentionalDisconnect) return;
      if (token == null || token.isEmpty) {
        _intentionalDisconnect = true;
        return;
      }
      final origin = ApiConfig.baseUri;
      final uri = origin.replace(
        scheme: origin.scheme == 'https' ? 'wss' : 'ws',
        path: '/ws',
      );
      candidate = openAuthenticatedSocket(uri, token);
      _channel = candidate;
      // Subscribe before awaiting readiness, so handshake/stream errors have an
      // owner and cannot surface as unhandled asynchronous exceptions.
      _subscription = candidate.stream.listen(
        (raw) {
          if (generation != _generation || raw is! String) return;
          _lastFrameAt = DateTime.now();
          try {
            final decoded = jsonDecode(raw);
            if (decoded is Map<String, dynamic>) _dispatch(decoded);
          } on FormatException {
            /* Ignore malformed remote frames. */
          }
        },
        onDone: () => _lost(generation),
        onError: (Object _) => _lost(generation),
        cancelOnError: true,
      );
      await candidate.ready.timeout(const Duration(seconds: 10));
      if (generation != _generation || _intentionalDisconnect) {
        unawaited(_closeChannel(candidate));
        return;
      }
      _lastFrameAt = DateTime.now();
      _connected = true;
      _connecting = false;
      _reconnectAttempts = 0;
      onConnectionChange?.call(true);
      _dispatch({'type': 'connection', 'connected': true});
      _pingTimer?.cancel();
      _pingTimer = Timer.periodic(const Duration(seconds: 30), (_) {
        if (_lastFrameAt != null &&
            DateTime.now().difference(_lastFrameAt!) >
                const Duration(seconds: 75)) {
          _lost(_generation);
        } else {
          send({'type': 'ping'});
        }
      });
    } catch (_) {
      _lost(generation);
    } finally {
      if (generation == _generation) _connecting = false;
    }
  }

  void disconnect() {
    _intentionalDisconnect = true;
    _tokenProvider = null;
    _reconnectAttempts = 0;
    _closeCurrent();
  }

  /// Renew the socket after app resume/network recovery without retaining an old
  /// Firebase token. Consumers refresh REST state on the next connection event.
  void refreshConnection() {
    if (_intentionalDisconnect || _tokenProvider == null) return;
    _closeCurrent();
    unawaited(_connect(null));
  }

  void _lost(int generation) {
    if (generation != _generation) return;
    _closeCurrent();
    if (!_intentionalDisconnect) _scheduleReconnect();
  }

  void _closeCurrent() {
    _generation++;
    final waiting = _chatAcks.values.toList();
    _chatAcks.clear();
    for (final ack in waiting) {
      if (!ack.isCompleted)
        ack.completeError(
          StateError('Chat connection closed before acknowledgement.'),
        );
    }
    _connecting = false;
    _connected = false;
    _pingTimer?.cancel();
    _pingTimer = null;
    _reconnectTimer?.cancel();
    _reconnectTimer = null;
    final subscription = _subscription;
    _subscription = null;
    if (subscription != null)
      unawaited(subscription.cancel().catchError((Object _) {}));
    final channel = _channel;
    _channel = null;
    if (channel != null) unawaited(_closeChannel(channel));
    onConnectionChange?.call(false);
    _dispatch({'type': 'connection', 'connected': false});
  }

  Future<void> _closeChannel(WebSocketChannel channel) async {
    try {
      await channel.sink.close(status.goingAway);
    } catch (_) {
      /* Already lost. */
    }
  }

  void _scheduleReconnect() {
    if (_intentionalDisconnect ||
        _tokenProvider == null ||
        _reconnectTimer?.isActive == true)
      return;
    final exponent = _reconnectAttempts > 5 ? 5 : _reconnectAttempts;
    final seconds = math.min(30, 1 << exponent);
    _reconnectAttempts++;
    final generation = _generation;
    _reconnectTimer = Timer(
      Duration(milliseconds: seconds * 1000 + _random.nextInt(500)),
      () {
        _reconnectTimer = null;
        if (generation == _generation && !_intentionalDisconnect) {
          unawaited(_connect(null));
        }
      },
    );
  }

  void resetReconnectAttempts() {
    _reconnectAttempts = 0;
    if (!_connected && !_connecting && !_intentionalDisconnect) {
      _reconnectTimer?.cancel();
      _reconnectTimer = null;
      _scheduleReconnect();
    }
  }

  /// A transport write is not delivery. Complete only with the persisted
  /// chat_sent response; callers reuse this ID for any uncertain retry.
  Future<Map<String, dynamic>> sendChat({
    required String groupId,
    required String content,
    required String clientMessageId,
  }) async {
    if (!_connected) throw StateError('Chat is reconnecting.');
    final existing = _chatAcks[clientMessageId];
    if (existing != null) {
      return existing.future.timeout(const Duration(seconds: 8));
    }
    if (_chatAcks.length >= 16) throw StateError('Chat send queue is full.');
    final ack = Completer<Map<String, dynamic>>();
    _chatAcks[clientMessageId] = ack;
    final result = ack.future.timeout(const Duration(seconds: 8));
    try {
      if (!send({
            'type': 'chat_message',
            'group_id': groupId,
            'content': content,
            'client_message_id': clientMessageId,
          }) &&
          !ack.isCompleted) {
        ack.completeError(StateError('Chat message was not written.'));
      }
      return await result;
    } finally {
      if (identical(_chatAcks[clientMessageId], ack))
        _chatAcks.remove(clientMessageId);
    }
  }

  bool send(Map<String, dynamic> data) {
    if (!_connected || _channel == null) return false;
    try {
      _channel!.sink.add(jsonEncode(data));
      return true;
    } catch (_) {
      _lost(_generation);
      return false;
    }
  }

  void _dispatch(Map<String, dynamic> data) {
    final type = data['type'];
    if (type is! String) return;
    final clientId = data['client_message_id'];
    final ack = clientId is String ? _chatAcks[clientId] : null;
    if (ack != null && !ack.isCompleted) {
      if (type == 'chat_sent' && data['message'] is Map<String, dynamic>) {
        ack.complete(data['message'] as Map<String, dynamic>);
      } else if (type == 'error') {
        ack.completeError(
          StateError('The server did not confirm this message.'),
        );
      }
    }
    for (final callback in List<void Function(Map<String, dynamic>)>.of(
      _listeners[type] ?? [],
    )) {
      try {
        callback(data);
      } catch (_) {
        /* One listener must not drop others. */
      }
    }
  }

  void on(String type, void Function(Map<String, dynamic>) callback) {
    final callbacks = _listeners.putIfAbsent(type, () => []);
    if (!callbacks.contains(callback)) callbacks.add(callback);
  }

  void off(String type, void Function(Map<String, dynamic>) callback) {
    _listeners[type]?.remove(callback);
  }

  void dispose() {
    disconnect();
    _listeners.clear();
    onConnectionChange = null;
  }
}
