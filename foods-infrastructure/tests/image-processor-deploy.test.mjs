import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {processorConfiguration,processorArgs,deployProcessor} from "../gcp/image-processor-deploy.mjs";
const env={GCP_DATABASE_READY:"true",GCP_PROJECT_ID:"foods-example",GCP_REGION:"southamerica-west1",GCP_ARTIFACT_REPOSITORY:"foods",GCP_WORKLOAD_IDENTITY_PROVIDER:"projects/123/locations/global/workloadIdentityPools/github/providers/github",GCP_DEPLOY_SERVICE_ACCOUNT:"deploy@foods-example.iam.gserviceaccount.com",GCP_BACKEND_SERVICE:"foods-api",GCP_BACKEND_SERVICE_ACCOUNT:"foods-api@foods-example.iam.gserviceaccount.com",GCP_MIGRATION_SERVICE_ACCOUNT:"foods-migrate@foods-example.iam.gserviceaccount.com",GCP_MIGRATION_JOB:"foods-migrate",GCP_MIGRATION_DATABASE_SECRET:"migration-db:1",GCP_BACKEND_SECRETS:"DATABASE_URL=runtime-db:1",GCP_VPC_CONNECTOR:"foods-connector",GCP_UPLOADS_BUCKET:"foods-images",GCP_IMAGE_PROCESSOR_SERVICE:"fudia-image-processor",GCP_IMAGE_PROCESSOR_SERVICE_ACCOUNT:"image-processor@foods-example.iam.gserviceaccount.com",GCP_EVENTARC_SERVICE_ACCOUNT:"image-events@foods-example.iam.gserviceaccount.com",GCP_IMAGE_PROCESSOR_DATABASE_SECRET:"processor-db:1",GCP_IMAGE_TRIGGER:"product-images",GITHUB_SHA:"a".repeat(40),GITHUB_RUN_ID:"1",GITHUB_RUN_ATTEMPT:"1"};
const processorState=()=>JSON.stringify({status:{latestReadyRevisionName:"processor-002",traffic:[{tag:"candidate",revisionName:"processor-002"}]}});
test("processor is private, isolated, bounded and uses its own runtime identity",()=>{
 const config=processorConfiguration(env),args=processorArgs(config,"image@sha256:digest");
 for(const flag of ["--invoker-iam-check","--no-allow-unauthenticated",env.GCP_IMAGE_PROCESSOR_SERVICE_ACCOUNT,"DATABASE_MAX_CONNS=2,PRODUCT_IMAGES_BUCKET=foods-images"])assert.ok(args.includes(flag));
 assert.equal(args[args.indexOf("--concurrency")+1],"1");assert.ok(!args.includes("--no-invoker-iam-check"));assert.ok(!args.includes("--add-volume"));
 assert.ok(args.includes(`fudia-commit=${config.sha},fudia-component=processor`));
 assert.ok(processorArgs(config,"digest",true).includes("--no-traffic"));
 assert.ok(!args.includes("--no-traffic"));
 assert.throws(()=>processorConfiguration({...env,GCP_IMAGE_PROCESSOR_SERVICE_ACCOUNT:env.GCP_BACKEND_SERVICE_ACCOUNT}),/distintas/);
});
test("processor uses the Cloud SQL socket without a VPC connector",()=>{
 const instance="foods-example:southamerica-west1:foods-postgres";
 const config=processorConfiguration({...env,GCP_VPC_CONNECTOR:"",GCP_CLOUD_SQL_INSTANCE:instance}),args=processorArgs(config,"image@sha256:digest");
 assert.equal(args[args.indexOf("--set-cloudsql-instances")+1],instance);
 assert.ok(args.includes("--clear-vpc-connector"));assert.ok(!args.includes("--vpc-connector"));assert.ok(!args.includes("--vpc-egress"));
});
test("migrates first, deploys processor then creates finalized-only Eventarc trigger",async()=>{
 const config=processorConfiguration(env),calls=[];
 await deployProcessor(config,{GCP_IMAGE_BUILT:"true"},(binary,args)=>{calls.push([binary,...args]);if(binary==="docker"&&args[0]==="image")return args[2].split(":")[0]+"@sha256:"+"b".repeat(64);if(args.slice(0,3).join(" ")==="run services describe")return processorState();if(args.includes("describe"))return null;return ""});
 const migration=calls.findIndex(call=>call.includes("execute")),service=calls.findIndex(call=>call[1]==="run"&&call[2]==="deploy"),binding=calls.findIndex(call=>call.includes("add-iam-policy-binding")),trigger=calls.findIndex(call=>call[1]==="eventarc"&&call[3]==="create");assert.ok(migration<service&&service<binding&&binding<trigger);
 assert.ok(calls[binding].includes(config.service));assert.ok(calls[binding].includes(`serviceAccount:${env.GCP_EVENTARC_SERVICE_ACCOUNT}`));assert.ok(calls[binding].includes("roles/run.invoker"));
 assert.ok(!calls[binding].includes("allUsers"));
 assert.ok(calls[trigger].includes("type=google.cloud.storage.object.v1.finalized"));assert.ok(calls[trigger].includes("bucket=foods-images"));assert.ok(calls[trigger].includes(env.GCP_EVENTARC_SERVICE_ACCOUNT));
 const promotion=calls.findIndex(call=>call.includes("update-traffic"));assert.ok(trigger<promotion);assert.ok(calls[promotion].includes("deployed=processor-002"));
});
test("a failed invocation permission stops trigger creation",async()=>{
 const calls=[];
 await assert.rejects(deployProcessor(processorConfiguration(env),{GCP_IMAGE_BUILT:"true"},(binary,args)=>{
  calls.push([binary,...args]);if(binary==="docker"&&args[0]==="image")return args[2].split(":")[0]+"@sha256:"+"b".repeat(64);
  if(args.includes("add-iam-policy-binding"))throw new Error("permission denied");if(args.includes("describe"))return null;return "";
 }),/permission denied/);
 assert.ok(!calls.some(call=>call[1]==="eventarc"&&call[3]==="create"));
});

function existingTriggerRunner(eventFilters){
 const calls=[];
 return {calls,run:(binary,args)=>{
  calls.push([binary,...args]);
  if(binary==="docker"&&args[0]==="image")return args[2].split(":")[0]+"@sha256:"+"b".repeat(64);
  if(args.slice(0,3).join(" ")==="run services describe")return processorState();
  if(args.slice(0,3).join(" ")==="eventarc triggers describe")return JSON.stringify({eventFilters});
  return "";
 }};
}

test("redeployment reads the actual unordered Eventarc filter array and updates the same trigger",async()=>{
 const mock=existingTriggerRunner([
  {attribute:"bucket",value:env.GCP_UPLOADS_BUCKET},
  {attribute:"type",value:"google.cloud.storage.object.v1.finalized"},
 ]);
 await deployProcessor(processorConfiguration(env),{GCP_IMAGE_BUILT:"true"},mock.run);
 const updates=mock.calls.filter(call=>call.slice(0,4).join(" ")==="gcloud eventarc triggers update");
 assert.equal(updates.length,1);
 assert.ok(updates[0].includes(env.GCP_IMAGE_TRIGGER));
 assert.ok(updates[0].includes(env.GCP_EVENTARC_SERVICE_ACCOUNT));
 assert.ok(!updates[0].includes("--event-filters"));
 assert.ok(!mock.calls.some(call=>call[1]==="eventarc"&&["create","delete"].includes(call[3])));
});

test("redeployment still supports map-shaped filters without replacing their source",async()=>{
 const mock=existingTriggerRunner({type:"google.cloud.storage.object.v1.finalized",bucket:env.GCP_UPLOADS_BUCKET});
 await deployProcessor(processorConfiguration(env),{GCP_IMAGE_BUILT:"true"},mock.run);
 assert.ok(mock.calls.some(call=>call.slice(0,4).join(" ")==="gcloud eventarc triggers update"));
});

test("wrong, missing, duplicate or pattern source filters block trigger updates and deletion",async()=>{
 const type={attribute:"type",value:"google.cloud.storage.object.v1.finalized"};
 const bucket={attribute:"bucket",value:env.GCP_UPLOADS_BUCKET};
 for(const filters of [
  [type,{...bucket,value:"other-bucket"}],
  [{...type,value:"google.cloud.storage.object.v1.deleted"},bucket],
  [type],[],undefined,[null,type],
  [type,bucket,bucket],
  [type,bucket,{...bucket,value:"other-bucket"}],
  [type,{...bucket,operator:"match-path-pattern"}],
  {type:type.value,bucket:"other-bucket"},
 ]){
  const mock=existingTriggerRunner(filters);
  await assert.rejects(deployProcessor(processorConfiguration(env),{GCP_IMAGE_BUILT:"true"},mock.run),/otro origen/);
  assert.ok(!mock.calls.some(call=>call[1]==="eventarc"&&["update","create","delete"].includes(call[3])));
  assert.ok(!mock.calls.some(call=>call.includes("update-traffic")));
 }
});
test("failed Eventarc configuration never promotes the processor or its completed marker",async()=>{
 const mock=existingTriggerRunner({type:"google.cloud.storage.object.v1.finalized",bucket:env.GCP_UPLOADS_BUCKET});
 await assert.rejects(deployProcessor(processorConfiguration(env),{GCP_IMAGE_BUILT:"true"},(binary,args)=>{
  if(args.slice(0,3).join(" ")==="eventarc triggers update")throw new Error("trigger failed");
  return mock.run(binary,args);
 }),/trigger failed/);
 assert.ok(!mock.calls.some(call=>call.includes("update-traffic")));
});
test("processor workflow keeps short-lived OIDC and serializes schema changes with API",()=>{
 const source=readFileSync(new URL("../../.github/workflows/foods-image-processor-gcp.yml",import.meta.url),"utf8");
 assert.ok(!source.includes("push:"));assert.ok(source.includes("workflow_dispatch:"));assert.ok(source.includes("workflow_call:"));assert.ok(!source.includes("credentials_json"));assert.ok(source.includes("create_credentials_file: false"));assert.ok(source.includes("-gcp-schema-deploy"));
 assert.ok(source.includes("environment: ${{ inputs.environment }}"));assert.ok(source.includes("needs: verify"));assert.ok(source.includes("GCP_DATABASE_READY: ${{ inputs.database_ready }}"));
 const api=readFileSync(new URL("../../.github/workflows/foods-backend-gcp.yml",import.meta.url),"utf8");assert.ok(api.includes("-gcp-schema-deploy"));
});
test("lifecycle only deletes objects explicitly scheduled after READY",()=>{
 const rule=JSON.parse(readFileSync(new URL("../gcp/product-images-lifecycle.json",import.meta.url))).rule[0];assert.equal(rule.condition.daysSinceCustomTime,2);assert.equal(rule.condition.age,undefined);
 const cors=JSON.parse(readFileSync(new URL("../gcp/product-images-cors.example.json",import.meta.url)));assert.ok(cors[0].method.includes("POST"));assert.ok(!cors[0].origin.includes("*"));
});
