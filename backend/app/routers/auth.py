"""
Authentication Router

Handles Firebase token verification and user onboarding.
"""

from fastapi import APIRouter, HTTPException, status, Depends
from pydantic import BaseModel, Field
from typing import Optional

from app.services.auth_service import AuthService
from app.services.user_service import UserService
from app.services.audit_service import AuditService
from app.models.user import User
from app.dependencies import get_current_user


router = APIRouter()
auth_service = AuthService()
user_service = UserService()
audit_service = AuditService()


class AuthConfigResponse(BaseModel):
    """Public registration policy; empty domains remain for older clients."""
    registration_open: bool = True
    allowed_domains: list[str] = []
    version: str = "2.1.0"


@router.get("/config", response_model=AuthConfigResponse)
async def get_auth_config():
    return AuthConfigResponse()


class TokenVerifyRequest(BaseModel):
    """Request to verify Firebase token."""
    id_token: str = Field(..., description="Firebase ID token")


class TokenVerifyResponse(BaseModel):
    """Response after token verification."""
    user_id: str
    email: str
    display_name: str
    photo_url: Optional[str]
    onboarding_completed: bool
    is_new_user: bool
    occupation: str = 'other'


class OnboardingRequest(BaseModel):
    """Request to complete onboarding."""
    phone: str = Field(..., min_length=10, max_length=20)
    gender: Optional[str] = Field(None, pattern="^[MFO]$")
    safety_consent: bool = Field(...)
    privacy_consent: bool = False
    occupation: str = Field('other', pattern='^(student|teacher|other)$')
    offer_lifts: bool = False
    photo_clear: bool = False
    tour_version: int = Field(0, ge=0, le=1)


class OnboardingResponse(BaseModel):
    """Response after completing onboarding."""
    success: bool
    message: str


@router.post("/verify", response_model=TokenVerifyResponse)
async def verify_token(request: TokenVerifyRequest):
    """
    Verify Firebase ID token and get/create user.
    
    This is the main authentication endpoint. The client obtains a
    Firebase ID token after Google Sign-In and sends it here.
    
    Registration accepts verified Firebase identities from any email domain.
    """
    # Verify the token
    claims = auth_service.verify_firebase_token(request.id_token)
    
    if not claims:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid or expired token"
        )
    
    email = claims.get("email", "")
    
    # Get or create user
    user, is_new = await auth_service.get_or_create_user(claims)
    
    # Log new user registration only (prevent spamming on every app open)
    # The 'verify' endpoint is hit frequently for profile fetching
    if is_new:
        await audit_service.log_login_event(
            user_id=user.user_id,
            email=email,
            method="New Registration"
        )

    
    return TokenVerifyResponse(
        user_id=user.user_id,
        email=user.email,
        display_name=user.display_name,
        photo_url=user.photo_url,
        onboarding_completed=user.onboarding_completed,
        is_new_user=is_new,
        occupation=user.occupation,
    )


@router.post("/onboard", response_model=OnboardingResponse)
async def complete_onboarding(
    request: OnboardingRequest,
    current_user: User = Depends(get_current_user)
):
    """
    Complete user onboarding with phone and consent.
    
    Required for all new users before they can use the app.
    """
    if current_user.onboarding_completed:
        return OnboardingResponse(
            success=True,
            message="Onboarding already completed"
        )
    
    if not request.safety_consent:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="You must agree to the safety terms to continue"
        )
    
    if not request.privacy_consent:
        raise HTTPException(400, 'Read and acknowledge the privacy policy')
    if request.tour_version == 1 and (not request.photo_clear or not current_user.photo_url):
        raise HTTPException(422, 'Add a clear profile photo and confirm it shows your face')
    try:
        await user_service.complete_onboarding(
            user_id=current_user.user_id,
            phone=request.phone,
            gender=request.gender,
            safety_consent=True,
            occupation=request.occupation,
            offer_lifts=request.offer_lifts,
            photo_clear=request.photo_clear,
            tour_version=request.tour_version,
        )
        
        await audit_service.log_user_action(
            user_id=current_user.user_id,
            action="completed_onboarding",
            metadata={"gender_provided": request.gender is not None}
        )
        
        return OnboardingResponse(
            success=True,
            message="Onboarding completed successfully"
        )
        
    except ValueError as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=str(e)
        )
