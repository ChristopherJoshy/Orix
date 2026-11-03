import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:orix/core/theme/app_theme.dart';
import 'package:orix/core/widgets/orix_tour.dart';

void main() {
  testWidgets('Tour advances and closes at narrow width with enlarged text', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(320, 640);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightTheme,
        builder: (context, child) => MediaQuery(
          data: MediaQuery.of(context).copyWith(
            disableAnimations: true,
            textScaler: const TextScaler.linear(2),
          ),
          child: child!,
        ),
        home: Scaffold(
          body: Builder(
            builder: (context) => TextButton(
              onPressed: () => showOrixTour(context),
              child: const Text('Open tour'),
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.text('Open tour'));
    await tester.pumpAndSettle();
    expect(find.text('Set your route'), findsOneWidget);
    await tester.tap(find.text('Next'));
    await tester.pumpAndSettle();
    expect(find.text('Meet your group'), findsOneWidget);
    await tester.tap(find.text('Next'));
    await tester.pumpAndSettle();
    expect(find.text('Travel on your terms'), findsOneWidget);
    await tester.tap(find.text('Got it'));
    await tester.pumpAndSettle();
    expect(find.byType(AlertDialog), findsNothing);
    expect(tester.takeException(), isNull);
  });
}
