import test from "node:test";
import assert from "node:assert/strict";
import {spawnSync, execFileSync} from "node:child_process";
import {mkdtempSync, mkdirSync, writeFileSync, rmSync} from "node:fs";
import {tmpdir} from "node:os";
import {join, dirname} from "node:path";
import {activeWorkerRelease, pendingComponents} from "../gcp/changed-components.mjs";
import {activeRelease, candidateRevision, releaseLabels} from "../gcp/release-state.mjs";

const base="a".repeat(40), head="b".repeat(40);
const env={GCP_PROJECT_ID:"foods-example",GCP_REGION:"us-central1",HEAD_SHA:head,
  GCP_IMAGE_PROCESSOR_SERVICE:"foods-processor",GCP_BACKEND_SERVICE:"foods-backend",GCP_FRONTEND_SERVICE:"foods-frontend",
  GCP_CONCIERGE_HTTP_SERVICE:"foods-concierge",GCP_CONCIERGE_WORKER_POOL:"concierge-worker"};
const services={processor:env.GCP_IMAGE_PROCESSOR_SERVICE,backend:env.GCP_BACKEND_SERVICE,frontend:env.GCP_FRONTEND_SERVICE,concierge:env.GCP_CONCIERGE_HTTP_SERVICE};
const state=(revision="active-001")=>({metadata:{labels:{"fudia-commit":head}},status:{latestReadyRevisionName:"candidate-002",traffic:[
  {revisionName:revision,percent:100},{revisionName:revision,tag:"deployed"},{revisionName:"candidate-002",tag:"candidate"},
]}});
const revision=(name,component,sha=base)=>({metadata:{name,labels:{"fudia-commit":sha,"fudia-component":component}},status:{conditions:[{type:"Ready",status:"True"}]}});
const json=value=>({status:0,stdout:JSON.stringify(value)});

function fake({snapshots={},commits={},diffs={},nativeGit,serviceError,revisionError,ancestry=0,workerPool,workerCommit,workerError}={}){
  const calls=[];
  const run=(binary,args,options)=>{
    calls.push([binary,...args]);
    assert.ok(!options.shell);
    if(binary==="git"){
      if(nativeGit)return nativeGit(binary,args,options);
      if(args[0]==="merge-base")return {status:ancestry,stdout:""};
      assert.equal(args[0],"diff");
      return {status:0,stdout:(diffs[args.at(-3)]??diffs.default??[]).join("\0")};
    }
    assert.equal(binary,"gcloud");
    assert.equal(args[0],"run");
    if(args[1]==="worker-pools"){
      assert.ok(args[2]==="describe"||(args[2]==="revisions"&&args[3]==="describe"));
      if(workerError)return workerError;
      if(args[2]==="describe"){
        if(workerPool===null)return {status:1,stderr:`Cannot find worker pool [${args[3]}]`};
        return json(workerPool??{status:{conditions:[{type:"Ready",status:"True"}],instanceSplits:[{revisionName:"concierge-worker-001",percent:100}]}});
      }
      return json(revision(args[4],"concierge",workerCommit??commits.concierge??base));
    }
    assert.equal(args[2],"describe"); // Detection must never mutate cloud state.
    if(args[1]==="services"){
      if(serviceError)return serviceError;
      const component=Object.keys(services).find(key=>services[key]===args[3]);
      if(snapshots[component]===null)return {status:1,stderr:`ERROR: (gcloud.run.services.describe) Cannot find service [${args[3]}]`};
      return json(snapshots[component]??state(`${component}-001`));
    }
    if(revisionError)return revisionError;
    assert.equal(args[1],"revisions");
    const component=args[3].split("-")[0];
    return json(revision(args[3],component,commits[component]??base));
  };
  return {run,calls};
}

test("baselines come from completed active revisions, never latest candidate or service labels",()=>{
  const mock=fake({diffs:{default:["foods-admin-web/new.tsx"]}});
  const result=pendingComponents(env,mock.run);
  assert.deepEqual(result.selected,{processor:false,backend:false,frontend:true,concierge:false});
  assert.equal(result.details[2].deployedSha,base);
  assert.ok(mock.calls.filter(call=>call[2]==="revisions").every(call=>call[4].endsWith("-001")));
  assert.ok(mock.calls.filter(call=>call[0]==="git"&&call[1]==="diff").every(call=>call.includes(base)&&call.includes(head)));
});

test("first installation and legacy revisions bootstrap once instead of losing pending changes",()=>{
  const noMarker=state("frontend-001");noMarker.status.traffic=noMarker.status.traffic.filter(target=>target.tag!=="deployed");
  const mock=fake({snapshots:{processor:null,frontend:noMarker}});
  assert.deepEqual(pendingComponents(env,mock.run).selected,{processor:true,backend:false,frontend:true,concierge:false});
  const missingLabel=fake();const run=(binary,args,options)=>{
    if(binary==="gcloud"&&args[1]==="revisions"){
      const result=revision(args[3],args[3].split("-")[0]);delete result.metadata.labels["fudia-commit"];return json(result);
    }
    return missingLabel.run(binary,args,options);
  };
  assert.deepEqual(pendingComponents(env,run).selected,{processor:true,backend:true,frontend:true,concierge:true});
});

test("IAM, network, timeout, invalid JSON and missing revision errors fail closed",()=>{
  for(const result of [{status:1,stderr:"PERMISSION_DENIED"},{status:1,stderr:"NOT_FOUND: project inaccessible"},
    {status:1,stderr:"network unavailable"},{status:1,stderr:"(NOT_FOUND)",error:new Error("timeout")},
    {status:0,stdout:"not-json"},{status:0,stdout:"null"},{status:0,stdout:"[]"}]){
    assert.throws(()=>pendingComponents(env,fake({serviceError:result}).run),/Cloud Run/);
  }
  assert.throws(()=>pendingComponents(env,fake({revisionError:{status:1,stderr:"(NOT_FOUND)"}}).run),/Cloud Run/);
});

test("release markers not serving all traffic are never treated as a deployed baseline",()=>{
  const rollback=state("frontend-001");rollback.status.traffic[0].revisionName="frontend-old";
  assert.equal(pendingComponents(env,fake({snapshots:{frontend:rollback}}).run).selected.frontend,true);
  const split=state("frontend-001");split.status.traffic[0].percent=50;split.status.traffic.push({revisionName:"frontend-002",percent:50});
  assert.equal(pendingComponents(env,fake({snapshots:{frontend:split}}).run).selected.frontend,true);
  assert.equal(activeRelease({status:{traffic:[{tag:"candidate",revisionName:"first-001",percent:100}]}}),null);
});

test("invalid identity, unavailable history and malformed traffic never silently skip deployment",()=>{
  for(const invalid of [{...env,GCP_FRONTEND_SERVICE:env.GCP_BACKEND_SERVICE},{...env,HEAD_SHA:"HEAD"},{...env,GCP_REGION:"us;command"}]){
    assert.throws(()=>pendingComponents(invalid,()=>assert.fail("must validate before cloud calls")));
  }
  assert.throws(()=>pendingComponents(env,fake({ancestry:1}).run),/ancestro/);
  const mock=fake();
  for(const labels of [{"fudia-commit":"not-a-sha","fudia-component":"backend"},{"fudia-commit":base,"fudia-component":"wrong"}]){
    assert.throws(()=>pendingComponents(env,(binary,args,options)=>binary==="gcloud"&&args[1]==="revisions"?
      json({...revision(args[3],"backend"),metadata:{name:args[3],labels}}):mock.run(binary,args,options)),/identidad/);
  }
  assert.throws(()=>activeRelease({status:{traffic:[{tag:"deployed",revisionName:"active-001",percent:80}]}}),/tráfico/);
  assert.throws(()=>candidateRevision({status:{latestReadyRevisionName:"other",traffic:[{tag:"candidate",revisionName:"old"}]}}),/candidata/);
  assert.throws(()=>releaseLabels("HEAD","backend"));
  assert.throws(()=>pendingComponents(env,(binary,args,options)=>binary==="gcloud"&&args[1]==="revisions"?
    json({...revision(args[3],"backend"),status:{conditions:[{type:"Ready",status:"False"}]}}):mock.run(binary,args,options)),/Ready/);
});

test("shared deployment changes are evaluated independently from each component baseline",()=>{
  const mock=fake({commits:{backend:head},diffs:{[base]:["foods-infrastructure/gcp/deploy.mjs"],[head]:[]}});
  assert.deepEqual(pendingComponents(env,mock.run).selected,{processor:true,backend:false,frontend:true,concierge:true});
});

test("regression: backend-only fix still deploys frontend changes blocked by the previous failed push",t=>{
  const directory=mkdtempSync(join(tmpdir(),"fudia-pending-deploy-"));
  t.after(()=>{assert.ok(directory.startsWith(join(tmpdir(),"fudia-pending-deploy-")));rmSync(directory,{recursive:true,force:true})});
  const git=(...args)=>execFileSync("git",["-C",directory,"-c","commit.gpgsign=false","-c","core.hooksPath=disabled-hooks",...args],{encoding:"utf8"}).trim();
  git("init","--quiet");git("config","user.name","Deployment Test");git("config","user.email","test@example.invalid");
  const save=(file,content)=>{mkdirSync(dirname(join(directory,file)),{recursive:true});writeFileSync(join(directory,file),content)};
  for(const component of ["foods-backend","foods-admin-web","fudia-image-processor","fudia-concierge"])save(`${component}/source.txt`,"baseline\n");
  const commit=message=>{git("add",".");git("commit","--quiet","-m",message);return git("rev-parse","HEAD")};
  const deployed=commit("deployed baseline");
  save("foods-admin-web/source.txt","new interface\n");save("foods-backend/source.txt","failing test\n");
  const failed=commit("frontend changes blocked by backend failure");
  save("foods-backend/source.txt","fixed test\n");const fixed=commit("backend-only fix");
  assert.deepEqual(git("diff","--name-only",failed,fixed).split("\n"),["foods-backend/source.txt"]);
  const nativeGit=(binary,args,options)=>spawnSync(binary,args,{...options,cwd:directory});
  const first=fake({commits:{processor:deployed,backend:deployed,frontend:deployed,concierge:deployed},nativeGit});
  assert.deepEqual(pendingComponents({...env,HEAD_SHA:fixed},first.run).selected,{processor:false,backend:true,frontend:true,concierge:false});
  const backendRecovered=fake({commits:{processor:deployed,backend:fixed,frontend:deployed,concierge:deployed},nativeGit});
  assert.deepEqual(pendingComponents({...env,HEAD_SHA:fixed},backendRecovered.run).selected,{processor:false,backend:false,frontend:true,concierge:false});
  const completed=fake({commits:{processor:deployed,backend:fixed,frontend:fixed,concierge:deployed},nativeGit});
  assert.deepEqual(pendingComponents({...env,HEAD_SHA:fixed},completed.run).selected,{processor:false,backend:false,frontend:false,concierge:false});
  save("fudia-concierge/source.txt","pending bot release\n");
  const botFailed=commit("Concierge changes not yet deployed");
  save("foods-admin-web/source.txt","unrelated frontend update\n");
  const unrelated=commit("frontend-only update after failed bot release");
  assert.deepEqual(git("diff","--name-only",botFailed,unrelated).split("\n"),["foods-admin-web/source.txt"]);
  const botPending=fake({commits:{processor:deployed,backend:fixed,frontend:unrelated,concierge:deployed},nativeGit});
  assert.deepEqual(pendingComponents({...env,HEAD_SHA:unrelated},botPending.run).selected,{processor:false,backend:false,frontend:false,concierge:true});
});

test("Concierge-only changes read the actual worker revision and incomplete releases remain pending",()=>{
  const mock=fake({diffs:{default:["foods-infrastructure/gcp/concierge-deploy.mjs","fudia-concierge/src/worker.py"]}});
  assert.deepEqual(pendingComponents(env,mock.run).selected,{processor:false,backend:false,frontend:false,concierge:true});
  assert.ok(mock.calls.some(call=>call.slice(0,5).join(" ")==="gcloud run worker-pools revisions describe"));
  const missing=fake({workerPool:null});
  assert.equal(pendingComponents(env,missing.run).selected.concierge,true);
  const ahead=fake({workerCommit:head});
  assert.equal(pendingComponents(env,ahead.run).selected.concierge,true);
  assert.throws(()=>pendingComponents(env,fake({workerError:{status:1,stderr:"PERMISSION_DENIED"}}).run),/Cloud Run/);
});

test("worker completion requires ready, reconciled, single-revision distribution, never latestReady",()=>{
  const pool={metadata:{generation:2},status:{observedGeneration:2,conditions:[{type:"Ready",status:"True"}],
    latestReadyRevisionName:"candidate-worker",instanceSplits:[{revisionName:"active-worker",percent:100}]}};
  assert.equal(activeWorkerRelease(pool),"active-worker");
  assert.equal(activeWorkerRelease({...pool,status:{...pool.status,observedGeneration:1}}),null);
  assert.equal(activeWorkerRelease({...pool,status:{...pool.status,conditions:[]}}),null);
  assert.equal(activeWorkerRelease({...pool,status:{...pool.status,instanceSplits:[{revisionName:"old-worker",percent:50},{revisionName:"new-worker",percent:50}]}}),null);
  assert.throws(()=>activeWorkerRelease({...pool,status:{...pool.status,instanceSplits:[{latestRevision:true,percent:100}]}}),/worker verificable/);
});
