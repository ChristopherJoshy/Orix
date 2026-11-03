import 'package:flutter/material.dart';
import 'package:orix/core/widgets/pixel_ui.dart';

class FAQScreen extends StatelessWidget {
  const FAQScreen({super.key});
  @override
  Widget build(BuildContext context) => OrixPage(
    title: 'Common questions',
    children: [
      for (final item in const [
        (
          'How does pooling work?',
          'Choose your starting point, destination and time. Pickups must be within 300 metres, the destination must be the same, and everyone’s time and safety preferences must fit.',
        ),
        (
          'How do I offer a lift?',
          'Open Offer a lift and register your route, time and available passenger seats.',
        ),
        (
          'How are costs shared?',
          'Your group agrees on costs and settles them directly. Orix does not process payments.',
        ),
        (
          'Can I cancel?',
          'You can cancel a waiting request or leave an eligible group from the journey screen. If you are the driver, leaving cancels the lift for the group.',
        ),
        (
          'How do I report a concern?',
          'Choose the contact’s email and a reason, then describe what happened. This starts a private conversation with moderation so you can answer follow-up questions.',
        ),
        (
          'Can I block someone?',
          'Yes. Search by email in Blocked contacts. Blocking excludes future matches and cancels shared active groups.',
        ),
        (
          'Does sign-in guarantee safety?',
          'No. Confirm who you are meeting and any vehicle details. For immediate danger, contact local emergency services.',
        ),
      ])
        Card(
          child: ExpansionTile(
            title: Text(item.$1),
            childrenPadding: const EdgeInsets.fromLTRB(20, 0, 20, 20),
            expandedCrossAxisAlignment: CrossAxisAlignment.start,
            children: [Text(item.$2)],
          ),
        ),
    ],
  );
}
