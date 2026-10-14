import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {processorConfiguration,processorArgs,deployProcessor} from "../gcp/image-processor-deploy.mjs";
const env={GCP_DATABASE_READY:"true",GCP_PROJECT_ID:"foods-example",GCP_REGION:"southamerica-west1",GCP_ARTIFACT_REPOSITORY:"foods",GCP_WORKLOAD_IDENTITY_PROVIDER:"projects/123/locations/global/workloadIdentityPools/github/providers/github",GCP_DEPLOY_SERVICE_ACCOUNT:"deploy@foods-example.iam.gserviceaccount.com",GCP_BACKEND_SERVICE:"foods-api",GCP_BACKEND_SERVICE_ACCOUNT:"foods-api@foods-example.iam.gserviceaccount.com",GCP_MIGRATION_SERVICE_ACCOUNT:"foods-migrate@foods-example.iam.gserviceaccount.com",GCP_MIGRATION_JOB:"foods-migrate",GCP_MIGRATION_DATABASE_SECRET:"migration-db:1",GCP_BACKEND_SECRETS:"DATABASE_URL=runtime-db:1",GCP_VPC_CONNECTOR:"foods-connector",GCP_UPLOADS_BUCKET:"foods-images",GCP_IMAGE_PROCESSOR_SERVICE:"fudia-image-processor",GCP_IMAGE_PROCESSOR_SERVICE_ACCOUNT:"image-processor@foods-example.iam.gserviceaccount.com",GCP_EVENTARC_SERVICE_ACCOUNT:"image-events@foods-example.iam.gserviceaccount.com",GCP_IMAGE_PROCESSOR_DATABASE_SECRET:"processor-db:1",GCP_IMAGE_TRIGGER:"product-images",GITHUB_SHA:"a".repeat(40),GITHUB_RUN_ID:"1",GITHUB_RUN_ATTEMPT:"1"};
test("processor is private, isolated, bounded and uses its own runtime identity",()=>{
 const config=processorConfiguration(env),args=processorArgs(config,"image@sha256:digest");
 for(const flag of ["--invoker-iam-check","--no-allow-unauthenticated",env.GCP_IMAGE_PROCESSOR_SERVICE_ACCOUNT,"DATABASE_MAX_CONNS=2,PRODUCT_IMAGES_BUCKET=foods-images"])assert.ok(args.includes(flag));
 assert.equal(args[args.indexOf("--concurrency")+1],"1");assert.ok(!args.includes("--no-invoker-iam-check"));assert.ok(!args.includes("--add-volume"));
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
 await deployProcessor(config,{GCP_IMAGE_BUILT:"true"},(binary,args)=>{calls.push([binary,...args]);if(binary==="docker"&&args[0]==="image")return args[2].split(":")[0]+"@sha256:"+"b".repeat(64);if(args.includes("describe"))return null;return ""});
 const migration=calls.findIndex(call=>call.includes("execute")),service=calls.findIndex(call=>call[1]==="run"&&call[2]==="deploy"),binding=calls.findIndex(call=>call.includes("add-iam-policy-binding")),trigger=calls.findIndex(call=>call[1]==="eventarc"&&call[3]==="create");assert.ok(migration<service&&service<binding&&binding<trigger);
 assert.ok(calls[binding].includes(config.service));assert.ok(calls[binding].includes(`serviceAccount:${env.GCP_EVENTARC_SERVICE_ACCOUNT}`));assert.ok(calls[binding].includes("roles/run.invoker"));
 assert.ok(!calls[binding].includes("allUsers"));
 assert.ok(calls[trigger].includes("type=google.cloud.storage.object.v1.finalized"));assert.ok(calls[trigger].includes("bucket=foods-images"));assert.ok(calls[trigger].includes(env.GCP_EVENTARC_SERVICE_ACCOUNT));
});
test("a failed invocation permission stops trigger creation",async()=>{
 const calls=[];
 await assert.rejects(deployProcessor(processorConfiguration(env),{GCP_IMAGE_BUILT:"true"},(binary,args)=>{
  calls.push([binary,...args]);if(binary==="docker"&&args[0]==="image")return args[2].split(":")[0]+"@sha256:"+"b".repeat(64);
  if(args.includes("add-iam-policy-binding"))throw new Error("permission denied");if(args.includes("describe"))return null;return "";
 }),/permission denied/);
 assert.ok(!calls.some(call=>call[1]==="eventarc"&&call[3]==="create"));
});
test("processor workflow keeps short-lived OIDC and serializes schema changes with API",()=>{
 const source=readFileSync(new URL("../../.github/workflows/foods-image-processor-gcp.yml",import.meta.url),"utf8");
 assert.ok(source.includes("workflow_dispatch:"));assert.ok(!source.includes("push:"));assert.ok(!source.includes("credentials_json"));assert.ok(source.includes("create_credentials_file: false"));assert.ok(source.includes("-gcp-schema-deploy"));
 const api=readFileSync(new URL("../../.github/workflows/foods-backend-gcp.yml",import.meta.url),"utf8");assert.ok(api.includes("-gcp-schema-deploy"));
});
test("lifecycle only deletes objects explicitly scheduled after READY",()=>{
 const rule=JSON.parse(readFileSync(new URL("../gcp/product-images-lifecycle.json",import.meta.url))).rule[0];assert.equal(rule.condition.daysSinceCustomTime,2);assert.equal(rule.condition.age,undefined);
 const cors=JSON.parse(readFileSync(new URL("../gcp/product-images-cors.example.json",import.meta.url)));assert.ok(cors[0].method.includes("POST"));assert.ok(!cors[0].origin.includes("*"));
});
