import { apiRequest, apiRequestPage } from './session';
import { adminRequest } from './admin';
export interface TravelContact { userId: string; name: string; email: string; photoUrl?: string }
export interface SafetyRecord {
  report_id: string; accused_user_id: string; group_id?: string;
  category: string; note: string; status: string; created_at: string;
  resolution?: string;
}
export interface ReportMessage {
  message_id: string; content: string; is_admin: boolean; created_at: string;
}
export interface ReportThread { report: SafetyRecord; messages: ReportMessage[] }
interface ContactResponse { user_id: string; email?: string; display_name?: string; photo_url?: string }
export function travelContact(row: ContactResponse): TravelContact {
  return { userId: row.user_id, name: row.display_name ?? '', email: row.email ?? '', photoUrl: row.photo_url };
}
export async function searchContacts(query: string, signal: AbortSignal) {
  const rows = await apiRequest<ContactResponse[]>(`/users/search?${new URLSearchParams({query})}`, {signal});
  return rows.filter((row) => row.email?.includes('@')).map(travelContact);
}
export async function blockedContacts(signal?: AbortSignal) {
  const data = await apiRequest<{ blocked_users: ContactResponse[] }>('/users/me/blocks', {signal});
  return data.blocked_users.map(travelContact);
}
export async function setContactBlocked(person: TravelContact, blocked: boolean) {
  await apiRequest(`/users/${encodeURIComponent(person.userId)}/block`, {method: blocked ? 'PUT' : 'DELETE', ...(blocked ? {body:'{}'} : {})});
}
export async function ownReports(signal?: AbortSignal,before?:string) {
  const page = await apiRequestPage<{reports: SafetyRecord[]}>(`/reports?${new URLSearchParams({limit:'100',...(before?{before}:{})})}`, {signal});
  return {reports:page.data.reports,nextCursor:page.nextCursor};
}
export async function createReport(person: TravelContact, category: string, note: string) {
  return apiRequest<{report_id: string}>('/reports', {method:'POST',body:JSON.stringify({accused_user_id:person.userId,category,note})});
}
export async function reportThread(id: string, admin: boolean, signal?: AbortSignal) {
  return admin ? adminRequest<ReportThread>('report-thread','GET',undefined,{report_id:id})
    : apiRequest<ReportThread>(`/reports/${encodeURIComponent(id)}`,{signal});
}
export async function replyToReport(id: string, content: string, clientID: string, admin: boolean) {
  const body = {client_message_id:clientID,content};
  return admin ? adminRequest<ReportThread>('report-reply','POST',body,{report_id:id})
    : apiRequest<ReportThread>(`/reports/${encodeURIComponent(id)}/messages`,{method:'POST',body:JSON.stringify(body)});
}
