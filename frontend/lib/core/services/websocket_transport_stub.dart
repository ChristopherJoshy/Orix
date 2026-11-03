import 'package:web_socket_channel/web_socket_channel.dart';

WebSocketChannel openAuthenticatedSocket(Uri uri, String token) =>
    throw UnsupportedError(
      'Realtime transport is unavailable on this platform.',
    );
