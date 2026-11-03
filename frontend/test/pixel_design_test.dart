import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:orix/core/theme/app_theme.dart';
import 'package:orix/core/widgets/pixel_ui.dart';

void main() {
  testWidgets('All lifecycle scenes render and settle with reduced motion', (
    tester,
  ) async {
    for (final scene in PixelScene.values) {
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightTheme,
          home: MediaQuery(
            data: const MediaQueryData(disableAnimations: true),
            child: Scaffold(
              body: Center(
                child: PixelAnimation(scene: scene, label: scene.name),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(tester.takeException(), isNull, reason: scene.name);
      expect(
        tester.binding.hasScheduledFrame,
        isFalse,
        reason: 'Reduced motion must stop ${scene.name}',
      );
    }
  });
  testWidgets('Loader retains brand in full and compact control constraints', (
    tester,
  ) async {
    for (final size in [const Size(200, 120), const Size(24, 24)]) {
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightTheme,
          home: MediaQuery(
            data: const MediaQueryData(disableAnimations: true),
            child: Scaffold(
              body: Center(
                child: SizedBox.fromSize(
                  size: size,
                  child: const PixelLoader(),
                ),
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(OrixLogo), findsOneWidget);
      expect(tester.takeException(), isNull);
    }
  });
  testWidgets('New page and toggle reflow on narrow display with large text', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(320, 740);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    var enabled = false;
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightTheme,
        home: MediaQuery(
          data: const MediaQueryData(
            size: Size(320, 740),
            textScaler: TextScaler.linear(2),
            disableAnimations: true,
          ),
          child: StatefulBuilder(
            builder: (context, setState) => OrixPage(
              title: 'Your report conversations',
              children: [
                PixelSection(
                  title: 'Your preferences',
                  child: PixelToggle(
                    title: const Text('Open to compatible lift offers'),
                    subtitle: const Text(
                      'Only match when your time window, route and preferences fit.',
                    ),
                    value: enabled,
                    onChanged: (value) => setState(() => enabled = value),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
    await tester.ensureVisible(find.byType(PixelToggle));
    await tester.pumpAndSettle();
    await tester.tap(find.byType(PixelToggle));
    await tester.pump();
    expect(enabled, isTrue);
    expect(tester.takeException(), isNull);
  });
}
