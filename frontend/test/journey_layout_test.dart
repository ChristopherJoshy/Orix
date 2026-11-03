import 'dart:io';
import 'dart:ui' as ui;
import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:orix/core/theme/app_theme.dart';
import 'package:orix/core/widgets/journey_route.dart';
import 'package:orix/core/widgets/pixel_ui.dart';

// Isolated layout content only. No accounts, matches or service outcomes.
const pickup =
    'St. Joseph’s College of Engineering and Technology, '
    'Choondachery - 686578, KL, India';
const destination =
    'KSRTC Bus Stand, Main Road Pala, Lalam Bridge, '
    'Pala - 686575, KL, India';

void main() {
  setUpAll(() async {
    for (final entry in {
      'Pixelify Sans': 'assets/fonts/pixelify-sans-500.ttf',
      'Manrope': 'assets/fonts/manrope-600.ttf',
      'MaterialIcons': 'fonts/MaterialIcons-Regular.otf',
    }.entries) {
      final loader = FontLoader(entry.key)..addFont(rootBundle.load(entry.value));
      await loader.load();
    }
  });
  testWidgets('Long routes expand and reset when the journey changes', (
    tester,
  ) async {
    Future<void> showRoute(String from) => tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightTheme,
        home: Scaffold(
          body: SingleChildScrollView(
            child: JourneyRoute(pickup: from, destination: destination),
          ),
        ),
      ),
    );
    await showRoute(pickup);
    expect(tester.widget<Text>(find.text(pickup)).maxLines, 2);
    await tester.tap(find.text('Full addresses'));
    await tester.pump();
    expect(tester.widget<Text>(find.text(pickup)).maxLines, isNull);
    await showRoute('$pickup Gate 2');
    expect(find.text('Full addresses'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('Journey surfaces reflow at mobile widths and double text size', (
    tester,
  ) async {
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    for (final size in [
      const Size(320, 740),
      const Size(390, 844),
      const Size(740, 390),
    ]) {
      for (final scale in [1.0, 2.0]) {
        await tester.pumpWidget(const SizedBox.shrink());
        tester.view.physicalSize = size;
        tester.view.devicePixelRatio = 1;
        final capture = GlobalKey();
        await tester.pumpWidget(
          MaterialApp(
            theme: AppTheme.lightTheme,
            home: MediaQuery(
              data: MediaQueryData(
                size: size,
                textScaler: TextScaler.linear(scale),
                disableAnimations: true,
              ),
              child: RepaintBoundary(
                key: capture,
                child: OrixPage(
                  title: 'Your journeys',
                  description: 'Current requests and previous journeys.',
                  bottom: const OrixNavigation(selected: 1),
                  children: [
                    PixelSection(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.stretch,
                        children: [
                          const JourneyRoute(
                            pickup: pickup,
                            destination: destination,
                          ),
                          const SizedBox(height: 16),
                          OutlinedButton(
                            onPressed: () {},
                            child: const Text('Open journey'),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ),
        );
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull, reason: '$size at $scale');
        await tester.scrollUntilVisible(find.text('Full addresses'), 160);
        await tester.pumpAndSettle();
        await tester.tap(find.text('Full addresses'));
        await tester.pumpAndSettle();
        await tester.ensureVisible(find.text('Open journey'));
        await tester.pumpAndSettle();
        expect(
          tester.takeException(),
          isNull,
          reason: 'Expanded $size at $scale',
        );
        if (size.width == 390 && scale == 1) {
          await tester.tap(find.text('Less detail'));
          await tester.pumpAndSettle();
          final boundary =
              capture.currentContext!.findRenderObject()!
                  as RenderRepaintBoundary;
          await tester.runAsync(() async {
            final image = await boundary.toImage(pixelRatio: 2);
            final bytes = await image.toByteData(
              format: ui.ImageByteFormat.png,
            );
            File(
              '/tmp/orix-journey-layout.png',
            ).writeAsBytesSync(bytes!.buffer.asUint8List());
            image.dispose();
          });
        }
      }
    }
  });
}
