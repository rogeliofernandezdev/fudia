"use client";

import {useEffect} from "react";
import {playTones,prepareAudio} from "@/shared/audio";
import {noticeSoundSchedule,type NoticeSound} from "../domain/notice-sound";

/**
 * Sonido de la campana por el audio compartido de la página: se prepara al
 * montar y cualquier toque o tecla lo habilita (el mismo que usan los pagos).
 */
export function useNoticeSound(){
  useEffect(()=>prepareAudio(),[]);
  return playNoticeSound;
}

function playNoticeSound(sound:NoticeSound){
  playTones(noticeSoundSchedule(sound));
}
