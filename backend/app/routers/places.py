"""Authenticated autocomplete through the operator's Geoapify free-tier account."""
import asyncio
import math
import time
from urllib.parse import urlsplit
import httpx
from fastapi import APIRouter, Depends, HTTPException, Query
from app.config import settings
from app.dependencies import get_current_active_user
from app.models.user import User

router = APIRouter()
_lock = asyncio.Lock()
_cache = {}
_last_request = 0.0

@router.get('/search')
async def search_places(query: str = Query(..., min_length=3, max_length=200), user: User = Depends(get_current_active_user)):
    global _last_request
    if not user.onboarding_completed:
        raise HTTPException(403, 'Complete onboarding first')
    origin = settings.geocoding_base_url.rstrip('/')
    configured = urlsplit(origin)
    if configured.scheme != 'https' or configured.hostname != 'api.geoapify.com' or configured.username or configured.query or configured.fragment or not settings.geocoding_api_token:
        raise HTTPException(503, 'Place search is not configured. Select a map point or try later.')
    query = query.strip()
    if len(query) < 3:
        raise HTTPException(422, 'Enter at least three characters')
    key = ' '.join(query.lower().split())
    now = time.monotonic()
    if key in _cache and _cache[key][0] > now:
        return _cache[key][1]
    if _lock.locked() or now - _last_request < .25:
        raise HTTPException(429, 'Place search is busy. Try again.', headers={'Retry-After': '1'})
    async with _lock:
        _last_request = time.monotonic()
        try:
            async with httpx.AsyncClient(timeout=6, follow_redirects=False) as client:
                async with client.stream('GET', origin + '/autocomplete', params={'text': query, 'format': 'json', 'limit': 5, 'apiKey': settings.geocoding_api_token}) as response:
                    if response.status_code != 200:
                        raise HTTPException(503, 'Place search is unavailable. Try again.')
                    payload = bytearray()
                    async for chunk in response.aiter_bytes():
                        payload.extend(chunk)
                        if len(payload) > 256 * 1024:
                            raise HTTPException(503, 'Place search returned an invalid response')
            import json
            data = json.loads(payload)
        except (httpx.HTTPError, ValueError):
            raise HTTPException(503, 'Could not search places. Try again.') from None
    places = []
    for item in data.get('results', [])[:5]:
        lat, lng, label, place_id = item.get('lat'), item.get('lon'), item.get('formatted'), item.get('place_id')
        if not isinstance(lat, (float, int)) or not isinstance(lng, (float, int)) or not math.isfinite(lat) or not math.isfinite(lng) or abs(lat)>90 or abs(lng)>180 or not isinstance(label,str) or not label.strip() or len(label)>1000 or not isinstance(place_id,str) or len(place_id)>512:
            continue
        places.append({'place_id': 'geoapify:' + place_id, 'label': label.strip(), 'latitude': lat, 'longitude': lng})
    result = {'places': places, 'attribution': 'Powered by Geoapify | OpenStreetMap contributors', 'attribution_url': 'https://www.geoapify.com/'}
    if len(_cache) >= 256:
        _cache.pop(next(iter(_cache)))
    _cache[key] = (time.monotonic() + 3600, result)
    return result
