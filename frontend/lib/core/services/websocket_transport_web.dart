import 'package:web_socket_channel/html.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

// Browsers cannot attach an Authorization header to a WebSocket handshake. The
// server reads this offered credential but negotiates only the fixed orix.v1
// protocol. Tokens stay out of URLs and are never reflected as the selected protocol.
WebSocketChannel openAuthenticatedSocket(Uri uri, String token) =>
    HtmlWebSocketChannel.connect(
      uri,
      protocols: ['orix.v1', 'firebase.$token'],
    );
