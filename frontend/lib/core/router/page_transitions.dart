/// Restrained navigation motion shared by the monochrome app screens.
library;

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

Widget _transition(
  BuildContext context,
  Animation<double> animation,
  Widget child,
  Offset distance,
) {
  if (MediaQuery.disableAnimationsOf(context) ||
      WidgetsBinding
          .instance
          .platformDispatcher
          .accessibilityFeatures
          .reduceMotion) {
    return child;
  }
  // Driven animations do not allocate status listeners requiring disposal.
  final progress = animation.drive(CurveTween(curve: Curves.easeOutCubic));
  if (distance == Offset.zero) {
    return FadeTransition(opacity: progress, child: child);
  }
  final ratio = MediaQuery.devicePixelRatioOf(context);
  return AnimatedBuilder(
    animation: progress,
    child: FadeTransition(opacity: progress, child: child),
    builder: (context, page) => Transform.translate(
      // Whole physical pixels preserve the logo and sprite edges during travel.
      offset: Offset(
        (distance.dx * (1 - progress.value) * ratio).round() / ratio,
        (distance.dy * (1 - progress.value) * ratio).round() / ratio,
      ),
      child: page,
    ),
  );
}

class _OrixTransitionPage<T> extends CustomTransitionPage<T> {
  _OrixTransitionPage({
    required super.child,
    super.name,
    super.arguments,
    super.restorationId,
    super.key,
    Offset distance = Offset.zero,
  }) : super(
         transitionDuration: const Duration(milliseconds: 180),
         reverseTransitionDuration: const Duration(milliseconds: 140),
         transitionsBuilder: (context, animation, secondaryAnimation, child) =>
             _transition(context, animation, child, distance),
       );
}

/// A small directional cue for opening details, independent of window width.
class SlideTransitionPage<T> extends _OrixTransitionPage<T> {
  SlideTransitionPage({
    required super.child,
    super.name,
    super.arguments,
    super.restorationId,
    super.key,
  }) : super(distance: const Offset(12, 0));
}

/// Main destinations appear without moving their text or stretching branding.
class FadeTransitionPage<T> extends _OrixTransitionPage<T> {
  FadeTransitionPage({
    required super.child,
    super.name,
    super.arguments,
    super.restorationId,
    super.key,
  });
}

/// Creation flows have the same short motion, directed from below.
class SlideUpTransitionPage<T> extends _OrixTransitionPage<T> {
  SlideUpTransitionPage({
    required super.child,
    super.name,
    super.arguments,
    super.restorationId,
    super.key,
  }) : super(distance: const Offset(0, 12));
}
