import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
import {createFilter} from "react-select";

const require=createRequire(import.meta.url);
function load(path,globals={}){
  const source=readFileSync(new URL(path,import.meta.url),"utf8");
  const compiled=ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,target:ts.ScriptTarget.ES2022}}).outputText;
  const context={exports:{},require,...globals};
  vm.runInNewContext(compiled,context);
  return context.exports;
}
const {countryLocationDefaults}=load("../src/modules/platform/domain/onboarding-defaults.ts");
const {timezoneOptions}=load("../src/shared/format/timezones.ts");
const {companySchema,locationSchema,onboardingStepFields}=load("../src/modules/platform/domain/onboarding-schemas.ts");
const peru={code:"PE",defaultTimezone:"America/Lima",callingCode:"+51"};
const colombia={code:"CO",defaultTimezone:"America/Bogota",callingCode:"+57"};

test("el país precarga la zona; otro país actualiza la sugerencia",()=>{
  const initial=countryLocationDefaults(peru,{country:"",timezone:"",locationPhone:""});
  assert.equal(initial.timezone,"America/Lima");
  const next=countryLocationDefaults(colombia,{country:"PE",timezone:"America/Lima",locationPhone:"+51999888777"},peru);
  assert.equal(next.timezone,"America/Bogota");
});

test("refrescar catálogos conserva la zona manual del local",()=>{
  const value=countryLocationDefaults(peru,{country:"PE",timezone:"America/New_York",locationPhone:"+1999888777"});
  assert.equal(value.timezone,"America/New_York");
});

test("un país sin sugerencia no hereda silenciosamente la zona anterior",()=>{
  const value=countryLocationDefaults({...colombia,defaultTimezone:""},{country:"PE",timezone:"America/Lima",locationPhone:""},peru);
  assert.equal(value.timezone,"");
});

test("zona obligatoria en Primer local, no en Empresa",()=>{
  assert.equal(companySchema.safeParse({legalName:"Empresa SAC",tradeName:"Empresa",taxId:"12345678"}).success,true);
  assert.equal(onboardingStepFields[0].includes("timezone"),false);
  assert.equal(onboardingStepFields[3].includes("timezone"),true);
  const local={locationName:"Principal",address:"Dirección de prueba",timezone:""};
  assert.equal(locationSchema.safeParse(local).success,false);
  assert.equal(locationSchema.safeParse({...local,timezone:"America/New_York"}).success,true);
});

test("búsqueda por ciudad o IANA mantiene el identificador persistido",()=>{
  const options=timezoneOptions("America/Argentina/Buenos_Aires");
  const matches=(search)=>options.filter(option=>createFilter()({...option,data:option},search));
  assert.ok(matches("Lima").some(option=>option.value==="America/Lima"));
  assert.ok(matches("Bogotá").some(option=>option.value==="America/Bogota"));
  assert.ok(matches("New York").some(option=>option.value==="America/New_York"));
  assert.ok(matches("America/").length>1);
  assert.ok(matches("Buenos Aires").some(option=>option.value==="America/Argentina/Buenos_Aires"));
  assert.equal(new Set(options.map(option=>option.value)).size,options.length);
  assert.ok(options.every(option=>option.value));
});

test("sin supportedValuesOf conserva la sugerencia del API y UTC",()=>{
  const fallback=load("../src/shared/format/timezones.ts",{Intl:{}});
  assert.deepEqual(Array.from(fallback.timezoneOptions("America/Lima"),option=>option.value),["America/Lima","UTC"]);
});
