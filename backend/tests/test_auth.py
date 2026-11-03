"""Public registration accepts any domain without bypassing token verification."""

from types import SimpleNamespace
from unittest.mock import AsyncMock, patch

import pytest

from fastapi import FastAPI
from httpx import ASGITransport, AsyncClient
from app.routers import auth


@pytest.fixture
def app():
    app = FastAPI()
    app.include_router(auth.router, prefix="/api/v1/auth")
    return app


@pytest.mark.asyncio
@pytest.mark.parametrize("email", ["traveler@gmail.com", "staff@company.org", "student@college.edu"])
async def test_public_registration(app, email):
    claims = {"uid": "firebase-test", "email": email, "email_verified": True}
    user = SimpleNamespace(user_id="user-test", email=email, display_name="Traveler",
                           photo_url=None, onboarding_completed=False)
    with patch.object(auth.auth_service, "verify_firebase_token", return_value=claims), \
         patch.object(auth.auth_service, "get_or_create_user", new=AsyncMock(return_value=(user, True))) as create, \
         patch.object(auth.audit_service, "log_login_event", new=AsyncMock()):
        async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
            response = await client.post("/api/v1/auth/verify", json={"id_token": "verified-token"})
        assert response.status_code == 200
        assert response.json()["email"] == email
        create.assert_awaited_once_with(claims)


@pytest.mark.asyncio
async def test_invalid_token_cannot_register(app):
    with patch.object(auth.auth_service, "verify_firebase_token", return_value=None), \
         patch.object(auth.auth_service, "get_or_create_user", new=AsyncMock()) as create:
        async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
            response = await client.post("/api/v1/auth/verify", json={"id_token": "invalid"})
        assert response.status_code == 401
        create.assert_not_awaited()


@pytest.mark.asyncio
async def test_registration_config_has_no_domain_dependency(app):
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
        response = await client.get("/api/v1/auth/config")
        assert response.status_code == 200
        assert response.json()["registration_open"] is True
        assert response.json()["allowed_domains"] == []
        assert (await client.post("/api/v1/auth/request-college", json={})).status_code == 404
