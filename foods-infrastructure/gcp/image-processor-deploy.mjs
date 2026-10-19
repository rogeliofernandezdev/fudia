import {spawnSync} from "node:child_process";
import {fileURLToPath} from "node:url";
import {resolve} from "node:path";
import {buildImage,configuration,databaseArgs,runMigrations,secretReferences} from "./deploy.mjs";
import {candidateRevision,promotionArgs,releaseLabels} from "./release-state.mjs";

function required(env,name,pattern){const value=env[name]??"";if(!pattern.test(value))throw new Error(`Configura ${name}.`);return value}
function triggerMatchesSource(filters,bucket){
 // Eventarc returns an unordered array of {attribute,value,operator}, not a map.
 // Keep map compatibility while requiring unique, exact type/bucket filters.
 const entries=Array.isArray(filters)?filters:Object.entries(filters??{}).map(([attribute,value])=>({attribute,value}));
 return Object.entries({type:"google.cloud.storage.object.v1.finalized",bucket}).every(([attribute,value])=>{
  const matches=entries.filter(entry=>entry?.attribute===attribute);
  return matches.length===1&&matches[0].value===value&&!matches[0].operator;
 });
}
export function processorConfiguration(env){
 const base=configuration("backend",env);
 const account=new RegExp(`^[a-z][a-z0-9-]+@${base.project}\\.iam\\.gserviceaccount\\.com$`);
 const identity=required(env,"GCP_IMAGE_PROCESSOR_SERVICE_ACCOUNT",account);
 const eventIdentity=required(env,"GCP_EVENTARC_SERVICE_ACCOUNT",account);
 if([base.identity,base.migrationIdentity].includes(identity)||[base.identity,identity].includes(eventIdentity))throw new Error("Procesador, API y Eventarc requieren identidades distintas.");
 const databaseSecret=secretReferences(`DATABASE_URL=${required(env,"GCP_IMAGE_PROCESSOR_DATABASE_SECRET",/^[a-zA-Z0-9_-]+:[1-9][0-9]*$/)}`,["DATABASE_URL"]).get("DATABASE_URL");
 return {...base,service:base.processorService,identity,eventIdentity,databaseSecret,migrationImage:base.image,migrationBuild:base,
  trigger:required(env,"GCP_IMAGE_TRIGGER",/^[a-z][a-z0-9-]{0,61}[a-z0-9]$/),
  image:base.image.replace("/foods-backend:","/fudia-image-processor:")};
}
function run(binary,args,capture=false,missing=false){
 const result=spawnSync(binary,args,{encoding:"utf8",stdio:capture?"pipe":"inherit"});
 if(result.error)throw result.error;
 if(result.status!==0){if(missing&&/NOT_FOUND|Cannot find/i.test(result.stderr??""))return null;throw new Error(`${binary} falló (${result.status}).`)}
 return capture?result.stdout.trim():"";
}
export function buildProcessor(config,command=run){buildImage(config.migrationBuild,command);command("docker",["build","--platform","linux/amd64","-t",config.image,"fudia-image-processor"])}
export function processorArgs(config,digest,exists=false){return ["run","deploy",config.service,"--image",digest,"--service-account",config.identity,
 "--tag","candidate","--update-labels",releaseLabels(config.sha,"processor"),...(exists?["--no-traffic"]:[]),
 "--execution-environment","gen2","--port","8080","--cpu","1","--memory","1Gi","--concurrency","1","--max-instances","3","--min-instances","0","--timeout","150s",
 "--ingress","all","--invoker-iam-check","--no-allow-unauthenticated",...databaseArgs(config),
 "--set-secrets",`DATABASE_URL=${config.databaseSecret}`,"--set-env-vars",`DATABASE_MAX_CONNS=2,PRODUCT_IMAGES_BUCKET=${config.bucket}`,
 "--startup-probe","httpGet.path=/health,httpGet.port=8080,periodSeconds=10,timeoutSeconds=3,failureThreshold=24"]}
export async function deployProcessor(config,env,command=run){
 const cloud=(args,capture=false,missing=false)=>command("gcloud",[...args,"--project",config.project,"--region",config.region,"--quiet"],capture,missing);
 const previous=cloud(["run","services","describe",config.service,"--format=json"],true,true);
 command("gcloud",["auth","configure-docker",config.registry,"--quiet"]);
 if(env.GCP_IMAGE_BUILT!=="true")buildProcessor(config,command);
 command("docker",["push",config.image]);
 const digest=command("docker",["image","inspect",config.image,"--format","{{index .RepoDigests 0}}"],true);
 if(!digest.startsWith(config.image.split(":")[0]+"@sha256:")||!/@sha256:[a-f0-9]{64}$/.test(digest))throw new Error("Digest del procesador inválido.");
 command("docker",["push",config.migrationImage]);
 const migrationDigest=command("docker",["image","inspect",config.migrationImage,"--format","{{index .RepoDigests 0}}"],true);
 if(!migrationDigest.startsWith(config.migrationImage.split(":")[0]+"@sha256:")||!/@sha256:[a-f0-9]{64}$/.test(migrationDigest))throw new Error("Digest del migrador inválido.");
 runMigrations(config,migrationDigest,cloud);
 cloud(processorArgs(config,digest,Boolean(previous)));
 // Grant invocation only on this private processor, before creating its trigger.
 cloud(["run","services","add-iam-policy-binding",config.service,
  "--member",`serviceAccount:${config.eventIdentity}`,"--role","roles/run.invoker"]);
 const triggerScope=["--project",config.project,"--location",config.region,"--quiet"];
 const existing=command("gcloud",["eventarc","triggers","describe",config.trigger,"--format=json",...triggerScope],true,true);
 const filters=["type=google.cloud.storage.object.v1.finalized",`bucket=${config.bucket}`];
 if(existing){
  const actual=JSON.parse(existing).eventFilters;
  if(!triggerMatchesSource(actual,config.bucket))throw new Error("El trigger existente usa otro origen; no se reemplazará.");
 }
 const triggerArgs=["eventarc","triggers",existing?"update":"create",config.trigger,"--destination-run-service",config.service,"--destination-run-region",config.region,"--destination-run-path","/","--service-account",config.eventIdentity];
 if(!existing)for(const filter of filters)triggerArgs.push("--event-filters",filter);
 command("gcloud",[...triggerArgs,...triggerScope]);
 // Only a ready revision with successfully configured IAM/trigger is completed.
 const current=JSON.parse(cloud(["run","services","describe",config.service,"--format=json"],true));
 cloud(promotionArgs(config.service,candidateRevision(current).revisionName));
}
if(process.argv[1]&&resolve(process.argv[1])===fileURLToPath(import.meta.url)){
 try{const config=processorConfiguration(process.env);if(process.argv[2]==="validate")console.log("Configuración del procesador validada.");else if(process.argv[2]==="build")buildProcessor(config);else if(!process.argv[2])await deployProcessor(config,process.env);else throw new Error("Argumento desconocido.")}
 catch(error){console.error(error instanceof Error?error.message:"Despliegue fallido.");process.exitCode=1}
}
