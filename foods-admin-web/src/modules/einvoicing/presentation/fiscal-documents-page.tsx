"use client";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,DetailFacts,DetailHero,EmptyState,ErrorState,Icon,ModalFooter,ModalHeader,PageHeader,Pagination,RowActionButton,SearchField,Select,Skeleton,Status,TableSkeleton} from "@/design-system";
import {DataTable} from "@/design-system/data-table";
import {Dialog} from "@/design-system/dialog";
import {useFeedback,useMoney,useSession} from "@/providers";
import {useDebouncedValue} from "@/shared/hooks/use-debounced-value";
import {formatRegionalCalendarDate,formatRegionalNumber} from "@/shared/i18n/regional-format";
import {fiscalStatusView} from "../domain/fiscal";
import type {FiscalDocumentSummary,FiscalFileKind} from "../domain/types";
import {fiscalDocumentFileUrl,getFiscalDocument,listFiscalDocuments,retryFiscalDocument} from "../infrastructure/einvoicing-api";
import "./einvoicing.css";

const fileLabels:Record<FiscalFileKind,string>={xml:"XML firmado",cdr:"CDR de SUNAT"};
const fileKinds:FiscalFileKind[]=["xml","cdr"];
const columns=["Comprobante","Fecha","Cliente","Total","Estado","Acciones"];

export function FiscalStatus({status}:{status:FiscalDocumentSummary["status"]}){
 const view=fiscalStatusView[status]??fiscalStatusView.PENDING;
 return <Status tone={view.tone}>{view.label}</Status>;
}

export function FiscalDocumentsPage(){
 const{organization,location,can}=useSession();
 const money=useMoney();
 const[q,setQ]=useState("");const query=useDebouncedValue(q);const[status,setStatus]=useState("");
 const[page,setPage]=useState(1);const[size,setSize]=useState(10);const[detailId,setDetailId]=useState<string|null>(null);
 const list=useQuery({queryKey:["fiscal-documents",organization?.id,location?.id,query,status,page,size],queryFn:()=>listFiscalDocuments({q:query,status,page,pageSize:size})});
 const day=(value:string)=>formatRegionalCalendarDate(value,location?.country,{dateStyle:"medium"});
 const items=list.data?.items??[];
 const filtered=Boolean(q||status);
 return <>
  <PageHeader title="Comprobantes" description="Boletas y facturas electrónicas emitidas en este local."/>
  <section className="panel standardized-management">
   <div className="toolbar">
    <SearchField value={q} onChange={event=>{setQ(event.target.value);setPage(1)}} placeholder="Buscar número, cliente o pedido…" label="Buscar comprobantes"/>
    <Select chip aria-label="Filtrar por estado" value={status} onChange={event=>{setStatus(event.target.value);setPage(1)}}><option value="">Todos los estados</option>{(list.data?.statusOptions??[]).map(option=><option key={option.value} value={option.value}>{option.label}</option>)}</Select>
   </div>
   {list.isLoading?<TableSkeleton columns={columns} label="Cargando comprobantes"/>:list.isError?<ErrorState title="No pudimos cargar los comprobantes" text={list.error.message} onRetry={()=>void list.refetch()} retrying={list.isFetching}/>:!items.length?
    <EmptyState icon="receipt" title={filtered?"Sin coincidencias":"Aún no hay comprobantes"} text={filtered?"Prueba con otro número, cliente o estado.":"Se registran al cobrar eligiendo boleta o factura."}/>:
    <DataTable label="Comprobantes electrónicos" containerClassName="hover-scroll"><thead><tr>{columns.map(column=><th key={column}>{column}</th>)}</tr></thead><tbody>{items.map((item,i)=><tr className={i%2?"alternate":""} key={item.id}>
     <td><span className={`row-icon r${i%3}`}><Icon name={item.documentType==="invoice"?"invoice":"receipt"} size={18}/></span><b>{item.number}</b><small>{item.label} · {item.orderCode}</small></td>
     <td>{day(item.issueDate)}</td>
     <td>{item.customerName||"Cliente sin identificar"}<small>{item.customerDocument}</small></td>
     <td><b>{money(Number(item.total))}</b></td>
     <td><FiscalStatus status={item.status}/></td>
     <td><div className="table-actions"><RowActionButton action="view" onClick={()=>setDetailId(item.id)}/></div></td>
    </tr>)}</tbody></DataTable>}
   <Pagination page={page} size={size} total={list.data?.total??0} onPage={setPage} onSize={value=>{setSize(value);setPage(1)}}/>
  </section>
  {detailId&&<FiscalDocumentDialog id={detailId} canRetry={can("orders.manage")} close={()=>setDetailId(null)}/>}
 </>;
}

export function FiscalDocumentDialog({id,canRetry,close}:{id:string;canRetry:boolean;close:()=>void}){
 const{location}=useSession();const{notify}=useFeedback();const money=useMoney();const client=useQueryClient();
 const detail=useQuery({queryKey:["fiscal-document",id],queryFn:()=>getFiscalDocument(id)});
 const retry=useMutation({mutationFn:()=>retryFiscalDocument(id),onSuccess:document=>{client.setQueryData(["fiscal-document",id],document);void client.invalidateQueries({queryKey:["fiscal-documents"]});notify({tone:"success",title:"Comprobante reenviado",message:`${document.label} ${document.number} · estado: ${fiscalStatusView[document.status].label.toLowerCase()}.`})},onError:error=>notify({tone:"danger",title:"No se pudo reenviar",message:error.message})});
 const data=detail.data;
 const percent=(rate:string)=>`${formatRegionalNumber(Math.round(Number(rate)*1_000_000)/10_000,location?.country,{maximumFractionDigits:4})} %`;
 return <div className="modal-backdrop modal-overlay-in"><Dialog onResponseClose={()=>{if(!retry.isPending)close()}} className="crud-modal modal-panel-in" role="dialog" aria-modal="true" aria-labelledby="fiscal-document-title">
  <ModalHeader title={data?`${data.label} ${data.number}`:"Comprobante"} titleId="fiscal-document-title" icon="receipt" onClose={close} busy={retry.isPending} closeLabel="Cerrar"/>
  <div className="modal-body">{detail.isLoading?<div className="organization-field-skeleton" role="status" aria-busy="true" aria-label="Cargando comprobante"><Skeleton height="var(--size-44)"/><Skeleton height="var(--size-120)"/></div>:detail.isError||!data?<ErrorState title="No pudimos cargar el comprobante" text={detail.error?.message??""} onRetry={()=>void detail.refetch()}/>:<>
   <DetailHero title={money(Number(data.total))} status={<FiscalStatus status={data.status}/>} meta={<><Icon name="calendar" size={14}/>{formatRegionalCalendarDate(data.issueDate,location?.country,{dateStyle:"medium"})} · {data.orderCode}</>}/>
   {data.statusMessage&&data.status!=="ACCEPTED"&&<p className="field-error" role="status">{data.statusMessage}</p>}
   <DetailFacts label="Datos del comprobante" items={[
    {label:"Cliente",value:data.customer.name||"Cliente sin identificar"},
    {label:"Documento",value:data.customerDocument||"Sin documento"},
    ...(data.customer.address?[{label:"Dirección",value:data.customer.address,wide:true}]:[]),
    {label:"Proveedor",value:`${data.providerLabel} · ${data.environment==="test"?"Pruebas":"Producción"}`},
    {label:"Intentos de envío",value:String(data.attempts)},
    ...(data.providerCode&&data.status!=="ACCEPTED"?[{label:"Código de respuesta",value:data.providerCode}]:[]),
   ]}/>
   {data.files.length>0&&<div className="fiscal-files">{fileKinds.filter(kind=>data.files.includes(kind)).map(kind=><a key={kind} className="button secondary" href={fiscalDocumentFileUrl(data.id,kind)} download><Icon name="download" size={16}/>{fileLabels[kind]}</a>)}</div>}
   <h3 className="modal-section-title fiscal-section-title">Detalle</h3>
   <table className="fiscal-lines"><thead><tr><th>Descripción</th><th className="num">Cant.</th><th className="num">Importe</th></tr></thead><tbody>{data.breakdown.lines.map(line=><tr key={line.no}><td>{line.description}</td><td className="num">{Number(line.quantity)}</td><td className="num">{money(Number(line.total))}</td></tr>)}</tbody></table>
   <dl className="fiscal-totals">
    <div><dt>Valor de venta</dt><dd>{money(Number(data.breakdown.netTotal))}</dd></div>
    {data.breakdown.taxes.map(tax=><div key={tax.code}><dt>{tax.name} {percent(tax.rate)}</dt><dd>{money(Number(tax.amount))}</dd></div>)}
    <div className="grand"><dt>Total</dt><dd>{money(Number(data.breakdown.total))}</dd></div>
   </dl>
  </>}</div>
  {data?.canRetry&&canRetry&&<ModalFooter><Button kind="secondary" icon="refresh" onClick={()=>retry.mutate()} disabled={retry.isPending} loading={retry.isPending}>{retry.isPending?"Reenviando…":"Reenviar"}</Button></ModalFooter>}
 </Dialog></div>;
}
