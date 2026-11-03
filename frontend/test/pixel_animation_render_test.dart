import 'dart:io';
import 'dart:ui' as ui;
import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:orix/core/theme/app_theme.dart';
import 'package:orix/core/widgets/pixel_ui.dart';

void main() {
  testWidgets('Render lifecycle frames for visual review', (tester) async {
    final output = Directory('/tmp/orix-pixel-frames')
      ..createSync(recursive: true);
    for (final scene in PixelScene.values) {
      final key = GlobalKey();
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightTheme,
          home: Scaffold(
            body: Center(
              child: RepaintBoundary(
                key: key,
                child: ColoredBox(
                  color: AppColors.background,
                  child: PixelAnimation(
                    key: ValueKey(scene),
                    scene: scene,
                    size: 240,
                    label: scene.name,
                  ),
                ),
              ),
            ),
          ),
        ),
      );
      for (final frame in [0, 450, 1800]) {
        await tester.pump(Duration(milliseconds: frame == 1800 ? 1350 : frame));
        final boundary =
            key.currentContext!.findRenderObject()! as RenderRepaintBoundary;
        await tester.runAsync(() async {
          final image = await boundary.toImage(pixelRatio: 2);
          final data = await image.toByteData(format: ui.ImageByteFormat.png);
          File(
            '${output.path}/${scene.name}-$frame.png',
          ).writeAsBytesSync(data!.buffer.asUint8List());
          image.dispose();
        });
        expect(tester.takeException(), isNull);
      }
    }
    await tester.pumpWidget(const SizedBox.shrink());
  });
}
