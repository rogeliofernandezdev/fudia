"use client";

import {useCallback,useEffect,useRef} from "react";
import {noticeSoundSchedule,type NoticeSound} from "../domain/notice-sound";

/**
 * Reproduce el sonido de la campana con Web Audio. El navegador solo deja
 * sonar después de un gesto en la página: el audio se prepara al cargar y
 * cualquier toque o tecla lo habilita (igual que el sonido de pagos).
 */
export function useNoticeSound(){
  const audio=useRef<AudioContext|null>(null);

  useEffect(()=>{
    const Context=window.AudioContext;
    if(!Context)return;
    const context=new Context();
    audio.current=context;
    const enable=()=>{if(context.state!=="running")void context.resume().catch(()=>undefined)};
    window.addEventListener("pointerdown",enable);
    window.addEventListener("keydown",enable);
    return()=>{
      window.removeEventListener("pointerdown",enable);
      window.removeEventListener("keydown",enable);
      audio.current=null;
      void context.close().catch(()=>undefined);
    };
  },[]);

  return useCallback((sound:NoticeSound)=>{
    const context=audio.current;
    if(!context||context.state!=="running")return;
    for(const tone of noticeSoundSchedule(sound)){
      const start=context.currentTime+tone.offset;
      const oscillator=context.createOscillator();
      const gain=context.createGain();
      oscillator.type=tone.wave;
      oscillator.frequency.value=tone.frequency;
      gain.gain.setValueAtTime(.0001,start);
      gain.gain.exponentialRampToValueAtTime(.3,start+.015);
      gain.gain.exponentialRampToValueAtTime(.0001,start+tone.duration);
      oscillator.connect(gain).connect(context.destination);
      oscillator.start(start);
      oscillator.stop(start+tone.duration+.02);
    }
  },[]);
}
