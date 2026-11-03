"""
ORIX Backend - FastAPI Application

Main application entry point with middleware, routers, and OpenAPI
documentation.
"""

from contextlib import asynccontextmanager
from typing import AsyncGenerator

from fastapi import FastAPI, Request, status
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse

from app.config import settings
from app.database import init_db, close_db, get_db, get_redis
from app.routers import (
    auth,
    users,
    rides,
    groups,
    reports,
    notifications,
    admin,
    websocket,
    support,
    logs,
    ratings,
    version,
    scheduler,
)
from app.middleware.rate_limit import RateLimitMiddleware
from app.services.audit_service import AuditService
from app.services.notification_service import NotificationService
from apscheduler.schedulers.asyncio import AsyncIOScheduler


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncGenerator:
    """
    Application lifespan manager.

    Handles startup and shutdown events:
    - Initialize database connections
    - Start background workers
    - Cleanup on shutdown
    """
    # Startup
    await init_db()

    # Verify connections and print logs
    print("\n" + "=" * 50)
    print("ORIX BACKEND STARTUP")
    print("=" * 50)

    # Verify MongoDB
    try:
        await get_db().client.admin.command("ping")
        print("Connected to db")
    except Exception as e:
        print(f"FAILED to connect to db: {e}")

    # Verify Firebase (trigger init via AuthService)
    try:
        from app.services.auth_service import AuthService
        import firebase_admin

        AuthService()  # Triggers _init_firebase
        firebase_admin.get_app()  # Verifies it exists
        print("Firebase connected")
    except Exception as e:
        print(f"FAILED to connect to Firebase: {e}")

    # Verify Redis
    try:
        await get_redis().ping()
        print("Redis connected")
    except Exception as e:
        print(f"FAILED to connect to Redis: {e}")

    print("=" * 50 + "\n")

    # Scheduler with industry-standard job management
    scheduler = None
    try:
        from app.scheduler import (
            health_check_job,
            ride_reminder_job,
            promotional_notification_job,
            data_cleanup_job,
        )

        scheduler = AsyncIOScheduler()

        # Register jobs with appropriate intervals
        scheduler.add_job(
            health_check_job.execute,
            "interval",
            minutes=10,
            id="health_check",
            name="Health Check Job",
            max_instances=1,
            coalesce=True,  # Skip if previous run is still executing
        )

        scheduler.add_job(
            promotional_notification_job.execute,
            "interval",
            minutes=30,
            id="promotional_notification",
            name="Promotional Notification Job",
            max_instances=1,
            coalesce=True,
        )

        scheduler.add_job(
            ride_reminder_job.execute,
            "interval",
            minutes=1,
            id="ride_reminder",
            name="Ride Reminder Job",
            max_instances=1,
            coalesce=True,
        )

        scheduler.add_job(
            data_cleanup_job.execute,
            "interval",
            minutes=5,
            id="data_cleanup",
            name="Data Cleanup Job",
            max_instances=1,
            coalesce=True,
        )

        scheduler.start()
        print(
            "✓ Scheduler started with 4 jobs: "
            "Health Check (10m) | Promotions (30m) | "
            "Reminders (1m) | Cleanup (5m)"
        )
    except Exception as e:
        print(f"✗ Failed to start scheduler: {e}")

    yield

    # Shutdown
    if scheduler:
        scheduler.shutdown()
        print("Scheduler stopped")

    await close_db()


# =============================================================================
# FastAPI Application
# =============================================================================

app = FastAPI(
    title="ORIX API",
    description="""
    ORIX - Travel Coordination Platform API

    ## Features
    - Firebase Authentication with public registration
    - Ride request creation and matchmaking
    - Real-time group coordination
    - Safety reporting and moderation

    ## Authentication
    All authenticated endpoints require a valid Firebase ID token in the
    Authorization header: `Authorization: Bearer <firebase_id_token>`
    """,
    version="1.0.0",
    docs_url="/docs",
    redoc_url="/redoc",
    openapi_url="/openapi.json",
    lifespan=lifespan,
)


# =============================================================================
# Middleware
# =============================================================================

# CORS Middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.cors_origins_list,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Rate Limiting Middleware
app.add_middleware(RateLimitMiddleware)


# =============================================================================
# Exception Handlers
# =============================================================================


@app.exception_handler(Exception)
async def global_exception_handler(request: Request, exc: Exception):
    """
    Global exception handler for unhandled errors.

    SECURITY: In production, do not leak internal error details.
    """
    # Persist an audit event without returning internal details to the client
    try:
        audit = AuditService()
        context = f"{request.method} {request.url.path}"
        await audit.log_exception(context, exc)
    except BaseException:
        pass

    # Always return simplified message to user, even in debug mode
    # if requested. But usually developers want debug info.
    # User requested: "fix showing such detailed errors... show a
    # simple plain english error message"
    # Sticking to the simplified message.

    return JSONResponse(
        status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
        content={
            "detail": (
                "Something went wrong. Our team has been notified "
                "and will fix it shortly."
            )
        },
    )


# =============================================================================
# Routers
# =============================================================================

# Routers imported at top of file

# Authentication routes
app.include_router(auth.router, prefix="/api/v1/auth", tags=["Authentication"])

# User routes
app.include_router(users.router, prefix="/api/v1/users", tags=["Users"])

# Ride routes
app.include_router(rides.router, prefix="/api/v1/rides", tags=["Rides"])

# Group routes
app.include_router(groups.router, prefix="/api/v1/groups", tags=["Groups"])

# Report routes
app.include_router(reports.router, prefix="/api/v1/reports", tags=["Reports"])

# Notification routes
app.include_router(
    notifications.router,
    prefix="/api/v1/notifications",
    tags=["Notifications"],
)

# Admin routes
app.include_router(admin.router, prefix="/api/v1/admin", tags=["Admin"])

# Support routes
app.include_router(support.router, prefix="/api/v1/support", tags=["Support"])

# WebSocket routes
app.include_router(websocket.router, prefix="/ws", tags=["WebSocket"])

# Logging routes
app.include_router(logs.router, prefix="/api/v1/logs", tags=["Logs"])

# App version routes
app.include_router(version.router, prefix="/api/v1/app", tags=["App"])

# Ratings routes
app.include_router(ratings.router, prefix="/api/v1/ratings", tags=["Ratings"])

# Scheduler monitoring routes
app.include_router(scheduler.router, prefix="/api/v1", tags=["Scheduler"])


# =============================================================================
# Health Check
# =============================================================================


@app.get("/health", tags=["Health"])
async def health_check():
    """Health check endpoint for load balancers and monitoring."""
    return {"status": "healthy", "version": "1.0.0"}


@app.get("/", tags=["Root"])
async def root():
    """Root endpoint with API information."""
    return {
        "name": "ORIX API",
        "version": "1.0.0",
        "docs": "/docs",
        "health": "/health",
    }


from app.routers import places
app.include_router(places.router, prefix="/api/v1/places", tags=["Places"])
