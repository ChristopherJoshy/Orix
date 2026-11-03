"""Private safety reports and reporter/moderator follow-up conversations."""
import uuid
import hmac
from typing import Literal, Optional
from fastapi import APIRouter, HTTPException, Depends, Header, status
from pydantic import BaseModel, Field
from app.utils.timezone_utils import utc_now
from app.models.user import User
from app.database import get_db
from app.services.group_service import GroupService
from app.services.audit_service import AuditService
from app.dependencies import get_current_active_user, get_current_user, get_admin_user

router = APIRouter()

class ReportRequest(BaseModel):
    accused_user_id: str = Field(..., min_length=1, max_length=100)
    group_id: Optional[str] = Field(None, min_length=1, max_length=100)
    category: Literal['harassment', 'unsafe_driving', 'no_show', 'spam', 'other'] = 'other'
    note: str = Field('', max_length=1000)

class ReportResponse(BaseModel):
    report_id: str
    message: str

class ReportReply(BaseModel):
    client_message_id: str = Field(..., min_length=8, max_length=100)
    content: str = Field(..., min_length=1, max_length=2000)

def public_report(record):
    keys = ('report_id', 'accused_user_id', 'group_id', 'note', 'category', 'status', 'created_at', 'resolved_at', 'resolution')
    return {key: record.get(key) for key in keys}

@router.post('', response_model=ReportResponse, status_code=status.HTTP_201_CREATED)
async def create_report(request: ReportRequest, current_user: User = Depends(get_current_active_user)):
    db = get_db()
    if request.accused_user_id == current_user.user_id:
        raise HTTPException(422, 'Cannot report yourself')
    if not await db.users.find_one({'user_id': request.accused_user_id}):
        raise HTTPException(404, 'Travel contact not found')
    evidence = None
    if request.group_id:
        group = await db.ride_groups.find_one({'group_id': request.group_id, 'members.user_id': {'$all': [current_user.user_id, request.accused_user_id]}})
        if not group:
            raise HTTPException(404, 'Group not found')
        evidence = await GroupService().get_recent_chat_for_report(request.group_id)
    report_id = str(uuid.uuid4())
    created = utc_now()
    note = request.note.strip()
    record = {'report_id': report_id, 'reporter_user_id': current_user.user_id, 'accused_user_id': request.accused_user_id, 'group_id': request.group_id, 'chat_excerpt': evidence, 'note': note, 'category': request.category, 'status': 'pending', 'resolution': None, 'resolution_note': None, 'resolved_by': None, 'created_at': created, 'resolved_at': None, 'messages': []}
    if note:
        record['messages'] = [{'message_id': report_id + '-initial', 'sender_id': current_user.user_id, 'is_admin': False, 'content': note, 'created_at': created}]
    await db.reports.insert_one(record)
    await AuditService().log_user_action(user_id=current_user.user_id, action='created_report', target_type='report', target_id=report_id, metadata={'group_id': request.group_id})
    return ReportResponse(report_id=report_id, message='Report conversation created')

@router.get('')
async def own_reports(current_user: User = Depends(get_current_user)):
    records = await get_db().reports.find({'reporter_user_id': current_user.user_id}).sort('created_at', -1).limit(100).to_list(100)
    return {'reports': [public_report(record) for record in records]}

async def conversation(report_id: str, user: User, admin: bool = False, reply: Optional[ReportReply] = None):
    db = get_db()
    scope = {'report_id': report_id}
    if admin:
        if user.role != "admin":
            raise HTTPException(403, "Administrator access required")
        # Admin endpoints still require get_admin_user (server role + configured secret).
        scope.update({'reporter_user_id': {'$ne': user.user_id}, 'accused_user_id': {'$ne': user.user_id}})
    else:
        scope['reporter_user_id'] = user.user_id
    record = await db.reports.find_one(scope)
    if not record:
        raise HTTPException(404, 'Report not found')
    if reply:
        text = reply.content.strip()
        if not text:
            raise HTTPException(422, 'Write a message')
        for message in record.get('messages', []):
            if message['message_id'] == reply.client_message_id and message['sender_id'] == user.user_id:
                if message['content'] != text:
                    raise HTTPException(409, 'Message ID already used')
                return _thread(record)
        if record['status'] != 'pending':
            raise HTTPException(409, 'This report conversation is closed')
        message = {'message_id': reply.client_message_id, 'sender_id': user.user_id, 'is_admin': admin, 'content': text, 'created_at': utc_now()}
        # The status, quota and idempotency predicates are checked in the same
        # atomic update as the append, including against concurrent resolution.
        guarded = {**scope, 'status': 'pending', 'messages.199': {'$exists': False}, 'messages': {'$not': {'$elemMatch': {'message_id': reply.client_message_id, 'sender_id': user.user_id}}}}
        updated = await db.reports.update_one(guarded, {'$push': {'messages': message}, '$set': {'updated_at': utc_now()}})
        if updated.modified_count != 1:
            fresh = await db.reports.find_one(scope)
            if fresh and any(m['message_id'] == reply.client_message_id and m['sender_id'] == user.user_id and m['content'] == text for m in fresh.get('messages', [])):
                return _thread(fresh)
            raise HTTPException(409, 'Conversation changed or reached its message limit. Refresh before retrying.')
        record = await db.reports.find_one(scope)
    return _thread(record)

def _thread(record):
    messages = record.get('messages', [])
    if record.get('note') and not any(m['message_id'] == record['report_id'] + '-initial' for m in messages):
        messages = [{'message_id': record['report_id'] + '-initial', 'sender_id': record['reporter_user_id'], 'is_admin': False, 'content': record['note'], 'created_at': record['created_at']}] + messages
    return {'report': public_report(record), 'messages': [{key: message[key] for key in ('message_id', 'content', 'is_admin', 'created_at')} for message in messages]}

@router.get('/{report_id}')
async def report_thread(report_id: str, user: User = Depends(get_current_user)):
    return await conversation(report_id, user)

@router.post('/{report_id}/messages')
async def report_reply(report_id: str, reply: ReportReply, user: User = Depends(get_current_active_user)):
    return await conversation(report_id, user, reply=reply)


async def get_report_moderator(user: User = Depends(get_admin_user), x_admin_secret: Optional[str] = Header(None)):
    from app.config import settings
    secret = settings.admin_api_secret
    if user.role != 'admin' or not secret or not x_admin_secret or not hmac.compare_digest(x_admin_secret, secret):
        raise HTTPException(403, 'Administrator role and secret required')
    return user
