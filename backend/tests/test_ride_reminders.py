from datetime import datetime, timezone, timedelta
from unittest.mock import AsyncMock

import pytest
from app.services.ride_reminder_service import departure_reminder_due, _send_reminders_to_group


def group():
    return {"group_id": "g", "date": "2026-11-03", "time_window": {"start": "00:10"},
            "timezone_offset_minutes": 330, "members": [{"user_id": "u"}, {"user_id": "u"}]}


@pytest.mark.parametrize("seconds,expected", [(901, False), (900, True), (899, True), (1, True), (0, False), (-1, False)])
def test_fifteen_minute_boundary_across_utc_midnight(seconds, expected):
    departure = datetime(2026, 11, 2, 18, 40, tzinfo=timezone.utc)
    assert departure_reminder_due(group(), departure - timedelta(seconds=seconds)) is expected


def test_missing_time_is_ignored():
    assert not departure_reminder_due({}, datetime.now(timezone.utc))


@pytest.mark.asyncio
async def test_reminder_has_fifteen_minute_payload_and_unique_members():
    service, tracker = AsyncMock(), AsyncMock()
    tracker.was_sent.return_value = False
    await _send_reminders_to_group(service, tracker, group())
    assert service.notify_catch_your_ride.await_count == 1
    assert service.notify_catch_your_ride.call_args.kwargs['minutes_remaining'] == 15
    tracker.mark_sent.assert_awaited_once()


@pytest.mark.asyncio
async def test_sent_reminder_is_not_repeated():
    service, tracker = AsyncMock(), AsyncMock()
    tracker.was_sent.return_value = True
    await _send_reminders_to_group(service, tracker, group())
    service.notify_catch_your_ride.assert_not_awaited()
