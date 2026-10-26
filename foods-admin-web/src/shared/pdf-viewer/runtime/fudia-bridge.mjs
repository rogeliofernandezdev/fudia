/* Owner: shared/pdf-viewer — adapter around Mozilla's unchanged full viewer. */
const protocol="fudia-pdf-viewer";
const controller=new AbortController(),signal=controller.signal;
let app,session,disposed=false;
// PDF.js creates URLs for downloads, attachments, images and print pages.
// Track only this isolated viewer realm, including URLs its download manager keeps.
const blobUrls=new Set();let released=false;
const createBlobUrl=URL.createObjectURL.bind(URL),revokeBlobUrl=URL.revokeObjectURL.bind(URL);
URL.createObjectURL=blob=>{const url=createBlobUrl(blob);if(released)revokeBlobUrl(url);else blobUrls.add(url);return url;};
URL.revokeObjectURL=url=>{blobUrls.delete(url);revokeBlobUrl(url);};
let initialize;
const initialized=new Promise(resolve=>{initialize=resolve;});
const post=(type,extra={})=>{if(session&&parent!==window)parent.postMessage({channel:protocol,session,type,...extra},location.origin);};
const button=id=>document.getElementById(id);
const actions=document.createElement("footer");actions.id="fudiaDocumentActions";
const controls=document.createElement("div");controls.className="fudia-document-buttons";
for(const [id,label] of [["printButton","Imprimir"],["downloadButton","Descargar PDF"]]){
 const control=button(id);
 control.removeAttribute("data-l10n-id");control.setAttribute("aria-label",label);control.title=label;
 control.classList.add("fudia-document-action");control.querySelector("span").removeAttribute("data-l10n-id");control.querySelector("span").textContent=label;
 controls.append(control);
}
const fullscreen=document.createElement("button");fullscreen.id="fudiaFullscreen";
fullscreen.type="button";fullscreen.className="fudia-fullscreen";fullscreen.textContent="Pantalla completa";fullscreen.setAttribute("aria-label","Pantalla completa");fullscreen.title="Pantalla completa";
controls.append(fullscreen);
const fitWidth=document.createElement("button");fitWidth.id="fudiaFitWidth";
fitWidth.type="button";fitWidth.className="toolbarButton labeled";
const fitWidthLabel=document.createElement("span");fitWidthLabel.textContent="Ajustar al ancho";
fitWidth.append(fitWidthLabel);button("secondaryToolbarButtonContainer").append(fitWidth);
fitWidth.addEventListener("click",()=>{
 if(!app?.pdfDocument)return;
 app.pdfViewer.currentScaleValue="page-width";app.secondaryToolbar.close();
},{signal});
const hint=document.createElement("p");hint.id="fudiaPrintHelp";hint.setAttribute("role","status");hint.setAttribute("aria-live","polite");
const defaultHint="Si aparece, elige «Todas las impresoras». Si no puedes imprimir, usa «Descargar PDF». Depende del dispositivo y su servicio de impresión.";
hint.textContent=defaultHint;
actions.append(controls,hint);document.body.append(actions);
// Reserve the actual action/help height, including wrapping and safe areas.
const actionObserver=typeof ResizeObserver==="undefined"?null:new ResizeObserver(()=>{
 document.documentElement.style.setProperty("--pdf-document-actions-height",`${actions.getBoundingClientRect().height}px`);
 app?.eventBus.dispatch("resize",{source:window});
});
actionObserver?.observe(actions);

function announce(message){hint.textContent=message;hint.classList.add("is-visible");}
fullscreen.addEventListener("click",async()=>{
 try{
  if(document.fullscreenElement)await document.exitFullscreen();
  else if(document.documentElement.requestFullscreen)await document.documentElement.requestFullscreen();
  else announce("Este navegador no permite pantalla completa. Puedes ampliar el documento con los controles de zoom.");
 }catch{announce("No se pudo activar pantalla completa. Usa los controles de zoom del visor.");}
},{signal});
document.addEventListener("fullscreenchange",()=>{const label=document.fullscreenElement?"Salir de pantalla completa":"Pantalla completa";fullscreen.textContent=label;fullscreen.setAttribute("aria-label",label);fullscreen.title=label;},{signal});

// Mozilla dispatches this synchronously to the embedding document BEFORE run().
// Set options here, not after initialization or via persisted browser preferences.
const onViewerLoaded=event=>{
 if(event.detail?.source!==window)return;
 const options=window.PDFViewerApplicationOptions;
 options.setAll({
  defaultUrl:"",disablePreferences:true,disableHistory:true,
  localeProperties:{lang:"es-ES"},defaultZoomValue:"page-fit",
  enableScripting:false,enablePermissions:true,enableAltTextModelDownload:false,enableGuessAltText:false,
  disableAutoFetch:false,enableHWA:false,enableDetailCanvas:false,
  maxCanvasPixels:matchMedia("(width <= 600px)").matches?2097152:8388608,
  maxCanvasDim:4096,printResolution:150,
 });
 app=window.PDFViewerApplication;
 app.initializedPromise.then(()=>{
  if(disposed)return;
  // Keep native previous/next controls available at narrow widths too.
  button("previous").parentElement.classList.remove("hiddenSmallView");
  // A temporary viewer URL is not a shareable document or an external-open action.
  button("viewBookmark").hidden=true;
  app.eventBus.on("documenterror",()=>post("error"),{signal});
  window.addEventListener("beforeprint",()=>{
   const service=app.printService;
   if(!service){announce("No se pudo preparar la impresión. Usa «Descargar PDF» para imprimir desde una aplicación compatible.");return;}
   // Observe preparation failures without replacing Mozilla's sequential print renderer.
   const renderPages=service.renderPages.bind(service);
   service.renderPages=()=>renderPages().catch(error=>{
    if(service.active!==false)announce("No se pudo preparar la impresión. Descarga el PDF e inténtalo desde una aplicación compatible.");
    throw error;
   });
  },{signal});
  button("printButton").addEventListener("click",async event=>{
   event.stopImmediatePropagation();
   if(!app.pdfDocument)return;
   announce(defaultHint);
   if(!app.supportsPrinting||!app.pdfViewer.printingAllowed){announce("La impresión no está disponible para este documento o navegador. Usa «Descargar PDF».");return;}
   try{
    // Page metadata is not page rendering. Mozilla needs every page's size to print;
    // huge documents may defer that metadata even after the first canvas is ready.
    if(app.pdfViewer.pageViewsReady===false){
     for(let page=1;page<=app.pdfDocument.numPages;page++){
      if(disposed)return;
      const view=app.pdfViewer.getPageView(page-1);
      if(!view.pdfPage)view.setPdfPage(await app.pdfDocument.getPage(page));
     }
    }
    if(!disposed)await app.triggerPrinting();
   }catch{app.afterPrint();announce("No se pudo abrir la impresión. Usa «Descargar PDF».");}
  },{capture:true,signal});
  initialize();
 });
};
const eventDocument=parent===window?document:parent.document;
eventDocument.addEventListener("webviewerloaded",onViewerLoaded,{signal});

window.addEventListener("message",async event=>{
 if(disposed||event.source!==parent||event.origin!==location.origin||event.data?.channel!==protocol)return;
 const data=event.data;
 if(data.type==="connect"&&typeof data.session==="string"){
  session=data.session;await initialized;if(!disposed)post("connected");return;
 }
 if(data.session!==session)return;
 if(data.type==="dispose"){await dispose();return;}
 if(data.type!=="open"||!(data.data instanceof ArrayBuffer))return;
 try{
  await initialized;
  if(disposed)return;
  await app.open({data:new Uint8Array(data.data),filename:String(data.filename||"documento.pdf").replace(/[\\/\r\n]/g,"-")});
  if(!disposed)post("loaded",{pages:app.pdfDocument?.numPages});
 }catch{if(!disposed)post("error");}
},{signal});

async function dispose(){
 if(disposed)return;
 disposed=true;controller.abort();actionObserver?.disconnect();
 if(document.fullscreenElement)void document.exitFullscreen().catch(()=>{});
 app?.printService?.destroy();
 app?.unbindEvents();app?.unbindWindowEvents();
 try{await app?.close();}
 finally{released=true;for(const url of blobUrls)URL.revokeObjectURL(url);}
}
window.FudiaPdfViewer={dispose};
window.addEventListener("pagehide",()=>void dispose(),{once:true});
