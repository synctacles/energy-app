"""Authentication Endpoints"""

import os

from fastapi import APIRouter, Depends, Header, HTTPException
from pydantic import BaseModel, EmailStr
from sqlalchemy.orm import Session

from synctacles_db import auth_service
from synctacles_db.api.dependencies import get_db

router = APIRouter()


class SignupRequest(BaseModel):
    email: EmailStr


class SignupResponse(BaseModel):
    user_id: str
    email: str
    license_key: str
    api_key: str
    message: str


class UserStatsResponse(BaseModel):
    user_id: str
    email: str
    tier: str
    rate_limit_daily: int
    usage_today: int
    remaining_today: int


class RegenerateKeyResponse(BaseModel):
    user_id: str
    email: str
    new_api_key: str
    message: str


class UserListResponse(BaseModel):
    total: int
    users: list


@router.post("/signup", response_model=SignupResponse)
async def signup(
    request: SignupRequest,
    db: Session = Depends(get_db)
):
    """
    Create new user account
    
    Returns license key and API key (save both!)
    """
    try:
        user, api_key_plain = auth_service.create_user(db, request.email)

        return SignupResponse(
            user_id=str(user.id),
            email=user.email,
            license_key=str(user.license_key),
            api_key=api_key_plain,
            message="Account created successfully. Save your API key - it won't be shown again!"
        )

    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))


@router.get("/stats", response_model=UserStatsResponse)
async def get_stats(
    x_api_key: str = Header(..., alias="X-API-Key"),
    db: Session = Depends(get_db)
):
    """Get current user statistics and rate limit info"""

    user = auth_service.validate_api_key(db, x_api_key)
    if not user:
        raise HTTPException(status_code=401, detail="Invalid API key")

    stats = auth_service.get_user_stats(db, user)
    return UserStatsResponse(**stats)


@router.post("/regenerate-key", response_model=RegenerateKeyResponse)
async def regenerate_api_key(
    x_api_key: str = Header(..., alias="X-API-Key"),
    db: Session = Depends(get_db)
):
    """
    Regenerate API key for current user
    
    Old key will be invalidated immediately
    """
    user = auth_service.validate_api_key(db, x_api_key)
    if not user:
        raise HTTPException(status_code=401, detail="Invalid API key")

    new_api_key = auth_service.regenerate_api_key(db, user)

    return RegenerateKeyResponse(
        user_id=str(user.id),
        email=user.email,
        new_api_key=new_api_key,
        message="API key regenerated. Update your applications immediately!"
    )


@router.post("/deactivate")
async def deactivate_account(
    x_api_key: str = Header(..., alias="X-API-Key"),
    db: Session = Depends(get_db)
):
    """Deactivate current user account (can be reactivated)"""
    user = auth_service.validate_api_key(db, x_api_key)
    if not user:
        raise HTTPException(status_code=401, detail="Invalid API key")

    auth_service.deactivate_user(db, user)

    return {"message": "Account deactivated successfully"}


@router.get("/admin/users", response_model=UserListResponse)
async def list_users(
    admin_key: str = Header(..., alias="X-Admin-Key"),
    db: Session = Depends(get_db)
):
    """
    List all users (admin only)
    
    Requires X-Admin-Key header
    """
    if admin_key != os.getenv("ADMIN_API_KEY", "change-me-in-production"):
        raise HTTPException(status_code=403, detail="Admin access required")

    users = db.query(auth_service.User).all()

    user_list = [
        {
            "user_id": str(u.id),
            "email": u.email,
            "tier": u.tier,
            "is_active": u.is_active,
            "created_at": u.created_at.isoformat(),
            "rate_limit": u.rate_limit_daily
        }
        for u in users
    ]

    return UserListResponse(total=len(users), users=user_list)
