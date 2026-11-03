"""One departure reminder per member, due fifteen minutes before the ride."""

import logging
from datetime import timedelta

from app.database import get_db
from app.services.notification_service import NotificationService
from app.services.notification_tracker import get_notification_tracker
from app.utils.timezone_utils import utc_now, parse_local_to_utc

logger = logging.getLogger(__name__)


def departure_reminder_due(group: dict, now) -> bool:
    """Catch up a delayed scheduler tick, but never alert after departure."""
    try:
        departure = parse_local_to_utc(
            group["date"], group["time_window"]["start"],
            group["timezone_offset_minutes"],
        )
    except (ValueError, KeyError, TypeError, OverflowError):
        return False
    return timedelta(0) < departure - now <= timedelta(minutes=15)


async def check_and_send_ride_reminders():
    db = get_db()
    service = NotificationService()
    tracker = get_notification_tracker()
    now = utc_now()
    # Local ride dates can be yesterday or tomorrow relative to UTC.
    dates = [(now + timedelta(days=offset)).strftime("%Y-%m-%d")
             for offset in (-1, 0, 1)]
    cursor = db.ride_groups.find({
        "status": {"$in": ["pending", "active", "confirming"]},
        "date": {"$in": dates},
    })
    async for group in cursor:
        if not departure_reminder_due(group, now):
            continue
        await _send_reminders_to_group(service, tracker, group)


async def _send_reminders_to_group(service, tracker, group):
    # Include the schedule so a rescheduled ride receives its own reminder.
    target = f"{group['group_id']}:{group['date']}:{group['time_window']['start']}"
    for user_id in {m.get("user_id") for m in group.get("members", [])} - {None, ""}:
        notification_type = "ride_departure_15min"
        if await tracker.was_sent(notification_type, target, user_id):
            continue
        try:
            await service.notify_catch_your_ride(
                user_id=user_id, group_id=group["group_id"],
                ride_time=group["time_window"]["start"],
                pickup_summary=group.get("pickup_summary", ""),
                minutes_remaining=15,
            )
            await tracker.mark_sent(notification_type, target, user_id, ttl_hours=48)
        except Exception:
            logger.exception("Departure reminder failed for group %s", group["group_id"])
