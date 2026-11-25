
    (()=>{
      const root=document.getElementById('fudia-order-preview');
      const orders=[{name:'Mesa 1',note:'Mozo: Miguel',channel:'Salón',status:'ready',time:'12:49',total:20},{name:'Lucía Ramírez',note:'',channel:'Delivery',status:'new',time:'12:43',total:62},{name:'Carlos Mendoza',note:'',channel:'Recojo',status:'new',time:'12:38',total:48},{name:'Ana Torres',note:'',channel:'Delivery',status:'ready',time:'12:32',total:36},{name:'Mesa 4',note:'Mozo: Miguel',channel:'Salón',status:'new',time:'12:26',total:22},{name:'Diego Salazar',note:'',channel:'Delivery',status:'new',time:'12:20',total:28}];
      const query=root.querySelector('input[type=search]'),status=root.querySelector('select'),tbody=root.querySelector('tbody'),modal=root.querySelector('.fp-modal-backdrop'),body=root.querySelector('.fp-modal-body');
      let channel='Todos',opener=null;
      const saved=window.openai?.widgetState;
      function normalize(text){return text.normalize('NFD').replace(/[\u0300-\u036f]/g,'').toLowerCase()}
      function money(value){return 'S/ '+value.toFixed(2)}
      function render(){
        const items=orders.filter(o=>(channel==='Todos'||o.channel===channel)&&(status.value==='all'||o.status===status.value)&&normalize(o.name).includes(normalize(query.value)));
        tbody.innerHTML=items.map(o=>`<tr><td class="fp-subject-col"><span class="fp-subject">${o.name}</span>${o.note?`<span class="fp-secondary">${o.note}</span>`:''}</td><td class="fp-channel-col"><span class="fp-channel">${o.channel}</span></td><td class="fp-status-col"><span class="fp-status ${o.status==='ready'?'ready':''}">${o.status==='ready'?'Por entregar':'Nuevo'}</span></td><td class="fp-time-col"><span class="fp-time">${o.time}</span></td><td class="fp-money">${money(o.total)}</td><td class="fp-row-action"><button class="fp-icon-button" data-detail="${orders.indexOf(o)}" aria-label="Ver detalle de ${o.name}"><i data-lucide="eye" aria-hidden="true"></i></button></td></tr>`).join('');
        root.querySelector('.fp-empty').hidden=items.length!==0;
        root.querySelector('table').hidden=items.length===0;
        root.querySelector('[data-count]').textContent=items.length+' '+(items.length===1?'pedido':'pedidos');
        root.querySelectorAll('[data-channel]').forEach(button=>button.setAttribute('aria-pressed',String(button.dataset.channel===channel)));
        if(globalThis.lucide)lucide.createIcons({attrs:{width:16,height:16}});
      }
      function persist(){window.openai?.setWidgetState?.({modelContent:{vista:'Propuesta de Pedidos FudIA',canal:channel},privateContent:{query:query.value,status:status.value}})?.catch(()=>{})}
      function applySaved(value){if(!value)return;const p=value.privateContent||{};const c=value.modelContent?.canal;channel=['Todos','Salón','Recojo','Delivery'].includes(c)?c:'Todos';query.value=typeof p.query==='string'?p.query:'';status.value=['all','new','ready'].includes(p.status)?p.status:'all';render()}
      function close(){modal.hidden=true;opener?.focus()}
      function open(title,content,trigger){opener=trigger;root.querySelector('#fp-dialog-title').textContent=title;body.innerHTML=content;modal.hidden=false;modal.querySelector('[data-close]').focus()}
      root.addEventListener('click',event=>{
        const tab=event.target.closest('[data-channel]');if(tab){channel=tab.dataset.channel;render();persist()}
        const detail=event.target.closest('[data-detail]');if(detail){const o=orders[Number(detail.dataset.detail)];open(o.name,`<dl><div><dt>Canal</dt><dd>${o.channel}</dd></div><div><dt>Estado</dt><dd><span class="fp-status ${o.status==='ready'?'ready':''}">${o.status==='ready'?'Por entregar':'Nuevo'}</span></dd></div><div><dt>Registrado</dt><dd>Hoy, ${o.time}</dd></div><div><dt>Total</dt><dd>${money(o.total)}</dd></div></dl>`,detail)}
        if(event.target.closest('[data-close]')||event.target===modal)close();
        const create=event.target.closest('[data-new]');if(create)open('Nuevo pedido','<label>Canal<select><option>Salón</option><option>Recojo en local</option><option>Delivery</option></select></label><label>Cliente o mesa<input placeholder="Nombre o mesa"></label><p class="fp-subtitle">Vista de ejemplo · No se registrarán cambios.</p>',create);
        const menu=event.target.closest('.fp-mobile-menu');if(menu){const opened=root.querySelector('.fp-shell').classList.toggle('menu-open');menu.setAttribute('aria-expanded',String(opened))}
      });
      root.addEventListener('keydown',event=>{if(modal.hidden)return;if(event.key==='Escape')close();if(event.key==='Tab'){const targets=[...modal.querySelectorAll('button,input,select')].filter(el=>!el.disabled),first=targets[0],last=targets[targets.length-1];if(event.shiftKey&&document.activeElement===first){event.preventDefault();last.focus()}else if(!event.shiftKey&&document.activeElement===last){event.preventDefault();first.focus()}}});
      query.addEventListener('input',()=>{render();persist()});status.addEventListener('change',()=>{render();persist()});
      window.addEventListener('openai:set_globals',event=>{if(event.detail?.globals?.widgetState)applySaved(event.detail.globals.widgetState)});
      render();applySaved(saved);
      if(globalThis.Tweak){const design={density:'Cómoda'};const tweak=new Tweak({container:root,onChange:()=>root.style.setProperty('--fp-row',design.density==='Compacta'?'54px':'66px')});tweak.addSelect(design,'density',{label:'Densidad de filas',options:['Cómoda','Compacta']})}
    })();
  
