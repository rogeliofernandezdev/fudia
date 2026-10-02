"use client";

import {useMemo,useState} from "react";
import {Button,Input,PageHeader,Pagination,RowActionButton,Select,Status} from "@/design-system/page-header";
import {Icon,IconName} from "@/design-system/icons";

type Tone="green"|"blue"|"orange"|"gray";
type Row={name:string;detail:string;category:string;value:string;state:string;tone?:Tone};
type Props={eyebrow:string;title:string;description:string;action:string;actionIcon?:IconName;columns:[string,string,string];rows:Row[]};

export function ManagementPage({eyebrow,title,description,action,actionIcon="plus",columns,rows}:Props){
 const[search,setSearch]=useState("");const[status,setStatus]=useState("");const[page,setPage]=useState(1);const[size,setSize]=useState(10);
 const filtered=useMemo(()=>rows.filter(row=>(!search||`${row.name} ${row.detail} ${row.category}`.toLowerCase().includes(search.toLowerCase()))&&(!status||row.state===status)),[rows,search,status]);
 const pages=Math.max(1,Math.ceil(filtered.length/size));const safePage=Math.min(page,pages);const visible=filtered.slice((safePage-1)*size,safePage*size);const states=[...new Set(rows.map(row=>row.state))];
 return <><PageHeader eyebrow={eyebrow} title={title} description={description} action={<Button icon={actionIcon}>{action}</Button>}/>
 <section className="mini-kpis"><article><small>TOTAL REGISTRADOS</small><strong>{rows.length}</strong><em>En todos los locales</em></article><article><small>OPERATIVOS</small><strong>{rows.filter(r=>r.tone!=="orange"&&r.tone!=="gray").length}</strong><em>Disponibles actualmente</em></article><article><small>REQUIEREN ATENCIÓN</small><strong className="warning">{rows.filter(r=>r.tone==="orange").length}</strong><em>Revisar hoy</em></article></section>
 <section className="panel management standardized-management"><div className="toolbar"><label className="ds-input-shell"><Icon name="search" size={18}/><Input value={search} onChange={e=>{setSearch(e.target.value);setPage(1)}} placeholder={`Buscar en ${title.toLowerCase()}...`}/></label><Select aria-label="Filtrar por estado" value={status} onChange={e=>{setStatus(e.target.value);setPage(1)}}><option value="">Todos los estados</option>{states.map(value=><option key={value}>{value}</option>)}</Select><button><Icon name="filter" size={17}/>Filtros</button><button><Icon name="download" size={17}/>Exportar</button></div>
 {visible.length?<><div className="table-wrap hover-scroll"><table><thead><tr><th>{columns[0]}</th><th>{columns[1]}</th><th>{columns[2]}</th><th>ESTADO</th><th>ACCIONES</th></tr></thead><tbody>{visible.map((row,index)=><tr className={index%2?"alternate":""} key={row.name}><td><span className={`row-icon r${index%3}`}><Icon name="box" size={18}/></span><b>{row.name}</b><small>{row.detail}</small></td><td>{row.category}</td><td><b>{row.value}</b></td><td><Status tone={row.tone}>{row.state}</Status></td><td><div className="standard-actions"><RowActionButton action="view"/><RowActionButton action="edit"/></div></td></tr>)}</tbody></table></div>
 <div className="management-cards">{visible.map((row,index)=><article key={row.name}><header><span className={`row-icon r${index%3}`}><Icon name="box" size={18}/></span><div><b>{row.name}</b><small>{row.detail}</small></div><Status tone={row.tone}>{row.state}</Status></header><dl><div><dt>{columns[1]}</dt><dd>{row.category}</dd></div><div><dt>{columns[2]}</dt><dd>{row.value}</dd></div></dl><footer><RowActionButton action="view"/><RowActionButton action="edit"/></footer></article>)}</div></>:<div className="catalog-state"><span><Icon name="search"/></span><b>No encontramos resultados</b><p>Ajusta la búsqueda o el estado para consultar otros registros.</p></div>}
 <Pagination page={safePage} size={size} total={filtered.length} onPage={setPage} onSize={value=>{setSize(value);setPage(1)}}/></section></>;
}
