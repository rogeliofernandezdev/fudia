"use client";
import {useEditor,EditorContent} from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import Placeholder from "@tiptap/extension-placeholder";
import Underline from "@tiptap/extension-underline";
import TextAlign from "@tiptap/extension-text-align";
import Highlight from "@tiptap/extension-highlight";
import {useEffect} from "react";

type RichTextEditorProps={
  value:string;
  onChange:(html:string)=>void;
  placeholder?:string;
};

export function RichTextEditor({value,onChange,placeholder="Escribe aquí..."}:RichTextEditorProps){
  const editor=useEditor({
    extensions:[
      StarterKit.configure({heading:{levels:[1,2,3]}}),
      Placeholder.configure({placeholder}),
      Underline,
      TextAlign.configure({types:["heading","paragraph"]}),
      Highlight.configure({multicolor:true}),
    ],
    content:value,
    onUpdate:(e)=>onChange(e.editor.getHTML()),
    immediatelyRender:false,
  });

  useEffect(()=>{
    if(editor&&value&&!editor.isFocused&&editor.getHTML()!==value){
      editor.commands.setContent(value,{emitUpdate:false});
    }
  },[value,editor]);

  if(!editor)return null;

  const btn=(onClick:()=>void,active:boolean,label:string,children:React.ReactNode)=>(
    <button type="button" onClick={onClick} className={active?"active":""} aria-label={label} title={label}>{children}</button>
  );

  return <div className="rte">
    <div className="rte-toolbar">
      <div className="rte-group">
        {btn(()=>editor.chain().focus().toggleBold().run(),editor.isActive("bold"),"Negrita",<b>B</b>)}
        {btn(()=>editor.chain().focus().toggleItalic().run(),editor.isActive("italic"),"Cursiva",<i>I</i>)}
        {btn(()=>editor.chain().focus().toggleUnderline().run(),editor.isActive("underline"),"Subrayado",<u>U</u>)}
        {btn(()=>editor.chain().focus().toggleStrike().run(),editor.isActive("strike"),"Tachado",<s>S</s>)}
      </div>
      <div className="rte-divider"/>
      <div className="rte-group">
        {btn(()=>editor.chain().focus().toggleHeading({level:3}).run(),editor.isActive("heading",{level:3}),"Título",<span style={{fontWeight:800}}>H</span>)}
        {btn(()=>editor.chain().focus().setParagraph().run(),editor.isActive("paragraph"),"Párrafo",<span>¶</span>)}
      </div>
      <div className="rte-divider"/>
      <div className="rte-group">
        {btn(()=>editor.chain().focus().toggleBulletList().run(),editor.isActive("bulletList"),"Lista con viñetas",<span>•</span>)}
        {btn(()=>editor.chain().focus().toggleOrderedList().run(),editor.isActive("orderedList"),"Lista numerada",<span>1.</span>)}
      </div>
      <div className="rte-divider"/>
      <div className="rte-group">
        {btn(()=>editor.chain().focus().setTextAlign("left").run(),editor.isActive({textAlign:"left"}),"Alinear izquierda",<span style={{fontFamily:"monospace"}}>≡</span>)}
        {btn(()=>editor.chain().focus().setTextAlign("center").run(),editor.isActive({textAlign:"center"}),"Centrar",<span style={{fontFamily:"monospace"}}>≡</span>)}
      </div>
      <div className="rte-divider"/>
      <div className="rte-group">
        {btn(()=>editor.chain().focus().toggleHighlight().run(),editor.isActive("highlight"),"Resaltar",<span style={{background:"#fef08a",padding:"0 2px"}}>A</span>)}
      </div>
    </div>
    <EditorContent editor={editor}/>
  </div>;
}
