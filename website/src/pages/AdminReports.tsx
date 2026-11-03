import { friendlyError } from "../lib/errors";
import { useRealtimeRefresh } from '../lib/realtime';
import { useCallback, useEffect, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { adminRequestPage } from '../lib/admin';
import { useSession } from '../lib/session';
import { type SafetyRecord } from '../lib/safety';
import { PageHead, Badge, Button } from '../components/ui';
import { PixelStatus } from '../components/PixelStatus';
export default function AdminReports() {
  const identity=useSession(s=>s.account?.user_id);const revision=useRef(0);
  const [reports,setReports]=useState<SafetyRecord[]|null>(null);const [cursor,setCursor]=useState<string|null>(null);const [error,setError]=useState('');const [loading,setLoading]=useState(false);
  const load=useCallback(async(before?:string)=>{
    const version=++revision.current;setLoading(true);setError('');
    try{
      const data=await adminRequestPage<{reports:{report:SafetyRecord}[]}>('reports','GET',undefined,before?{before}:undefined);
      if(version===revision.current&&identity===useSession.getState().account?.user_id){const rows=data.data.reports.map(row=>row.report);setReports(previous=>before?[...new Map([...(previous??[]),...rows].map(report=>[report.report_id,report])).values()]:rows);setCursor(data.nextCursor);}
    }catch(problem){if(version===revision.current)setError(friendlyError(problem));}
    finally{if(version===revision.current)setLoading(false);}
  },[identity]);
  useRealtimeRefresh(load);
  useEffect(()=>{setReports(null);void load();return()=>{revision.current++;};},[load]);
  return <>
    <PageHead title="Report conversations" subtitle="Pending safety reviews"><Button variant="secondary" disabled={loading} onClick={()=>void load()}>Refresh reports</Button></PageHead>
    {error&&<p role="alert">{error}</p>}
    {loading&&reports===null ? <PixelStatus state="loading" label="Opening reports"/> : reports?.length===0 ? <section className="panel"><p>No pending reports.</p></section> : reports?.map(report=><section className="panel" key={report.report_id}>
      <Badge>{report.status}</Badge><h2>{report.category.replaceAll('_',' ')}</h2><p>{report.note}</p><p className="muted">{new Date(report.created_at).toLocaleString()}</p>
      <Link className="button secondary" to={`/admin/reports/${encodeURIComponent(report.report_id)}`}>Open conversation</Link>
    </section>)}
    {cursor&&<Button variant="ghost" disabled={loading} onClick={()=>void load(cursor)}>Earlier reports</Button>}
  </>;
}
