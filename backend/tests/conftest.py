"""Isolated defaults for service tests; no production credentials."""
import os

# These tests use mocked services and never connect to production infrastructure.
for key, value in {
    "FIREBASE_PROJECT_ID": "orix-test",
    "MONGODB_URI": "mongodb://localhost:27017",
    "REDIS_URL": "redis://localhost:6379",
    "TELEGRAM_BOT_TOKEN": "test",
    "TELEGRAM_SECRET_ADMIN_PASSPHRASE": "test-passphrase",
    "ADMIN_API_SECRET": "test-admin-secret",
    "JWT_SECRET": "test-jwt-secret",
    "CORS_ORIGINS": "http://localhost",
}.items():
    os.environ.setdefault(key, value)

