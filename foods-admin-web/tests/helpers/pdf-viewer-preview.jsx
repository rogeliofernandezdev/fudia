import React,{useState} from "react";
import {createRoot} from "react-dom/client";
import {jsPDF} from "jspdf";
import {PdfViewerModal} from "../../src/shared/pdf-viewer/pdf-viewer";
import {buildPaymentTicketPdf,layoutPaymentTicket} from "../../src/modules/sales/infrastructure/payment-ticket-pdf";
import {buildCashReportPdf,loadFudiaReportLogo} from "../../src/modules/operations/cash/infrastructure/cash-report-pdf";
import {cashReportFixture} from "./cash-report-fixture.mjs";
import "../../src/styles/globals.css";
import "../../src/design-system/styles/modal.css";
import "../../src/design-system/styles/pdf-viewer.css";
import "../../src/design-system/styles/skeleton.css";
import "../../src/design-system/styles/remote-modal-skeleton.css";

const ticket={receiptContext:{organizationName:"Restaurante de prueba",legalName:"Restaurante de prueba",taxId:"20123456789",locationName:"Local de prueba",address:"Dirección de prueba",country:"PE",timezone:"America/Lima",currency:"PEN",currencySymbol:"S/",currencyPosition:"before",currencyDecimals:2},order:{code:"PED-PRUEBA",channel:"salon",tableName:"03",customerName:"Cliente de prueba",createdAt:"2026-10-26T12:00:00Z",subtotal:"30",deliveryFee:"0",total:"30",items:[{name:"Ají de gallina",qty:"2",unitPrice:"15",selections:[]}]},payments:[{id:"test-payment",netAmount:"30",methodName:"Efectivo",createdAt:"2026-10-26T12:00:00Z",createdByName:"Personal de prueba"}],paidAmount:"30",remainingAmount:"0",paymentStatus:"paid"};
function App(){
 const[kind,setKind]=useState(null),[pdf,setPdf]=useState(null),[error,setError]=useState(null);
 async function preview(next){
  setKind(next);setPdf(null);setError(null);
  try{
   if(next==="ticket")setPdf(buildPaymentTicketPdf(layoutPaymentTicket(ticket)));
   else if(next==="cash")setPdf(buildCashReportPdf(cashReportFixture(80),await loadFudiaReportLogo(),{primary:"#4654cd",ink:"#11152b",muted:"#59617c",surface:"#ffffff",soft:"#f7f8fc",line:"#e2e6ef",brand:"#63dcae"}));
   else if(next==="error")setPdf(new Blob(["not a pdf"],{type:"application/pdf"}));
   else{
    const doc=new jsPDF();for(let page=1;page<=80;page++){if(page>1)doc.addPage();doc.text(`FudIA - Documento de prueba - Pagina ${page}`,20,25);doc.text("Navegacion, busqueda y zoom del visor oficial.",20,45);}setPdf(doc.output("blob"));
   }
  }catch(cause){setError(String(cause));}
 }
 return <main><h1>Prueba aislada de PdfViewer</h1><p>Datos ficticios. No usa sesión, no llama al API y no registra ventas ni cobros.</p>
  <button className="button secondary" onClick={()=>preview("ticket")}>Previsualizar ticket 80 mm</button>
  <button className="button secondary" onClick={()=>preview("cash")}>Previsualizar cierre de caja</button>
  <button className="button secondary" onClick={()=>preview("long")}>Previsualizar 80 páginas</button>
  <button className="button secondary" onClick={()=>preview("error")}>Probar PDF dañado</button>
  {kind&&<PdfViewerModal title="Previsualización de prueba" size={kind==="ticket"||kind==="error"?"sm":"lg"} filename={`${kind}-prueba.pdf`} document={pdf} loading={!pdf&&!error} error={error} onClose={()=>{setKind(null);setPdf(null);}} onRetry={()=>preview(kind==="error"?"ticket":kind)} notice={<><span>Documento de prueba · sin datos reales</span><button className="button secondary" onClick={()=>preview(kind==="ticket"?"long":"ticket")}>Cambiar PDF</button></>}/>}
 </main>;
}
createRoot(document.getElementById("root")).render(<React.StrictMode><App/></React.StrictMode>);
