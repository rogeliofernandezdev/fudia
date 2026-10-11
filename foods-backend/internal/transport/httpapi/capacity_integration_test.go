package httpapi

import (
 "context"
 "encoding/json"
 "fmt"
 "io"
 "net/http"
 "net/http/httptest"
 "os"
 "sort"
 "sync"
 "testing"
 "time"
)

type capacityResult struct {
 Workers int `json:"workers"`
 Requests int `json:"requests"`
 Success int `json:"success"`
 Errors int `json:"errors"`
 RPS float64 `json:"requests_per_second"`
 P50MS float64 `json:"p50_ms"`
 P95MS float64 `json:"p95_ms"`
 P99MS float64 `json:"p99_ms"`
 ElapsedMS float64 `json:"elapsed_ms"`
}
type capacityReport struct {
 Scenario string `json:"scenario"`
 Commit string `json:"commit"`
 Environment string `json:"environment"`
 Stages []capacityResult `json:"stages"`
}
func capacityPercentile(sorted []time.Duration,p float64) float64 {
 if len(sorted)==0{return 0}
 index:=int(float64(len(sorted)-1)*p+0.5)
 return float64(sorted[index].Microseconds())/1000
}

// TestCapacityReadProducts benchmarks an HTTP endpoint backed by real PostgreSQL.
// A test-only scope injector replaces authentication; results are NOT a production
// Cloud Run benchmark or a proof of multi-region horizontal scaling.
func TestCapacityReadProducts(t *testing.T) {
 if os.Getenv("FUDIA_LOAD_TEST")!="1"{t.Skip("enable FUDIA_LOAD_TEST=1 to measure database-backed HTTP capacity")}
 pool:=integrationPool(t)
 scope:=seedInventoryScope(t,pool)
 api:=New(pool)
 for i:=0;i<15;i++ {
  body:=fmt.Sprintf(`{"name":"Capacity product %02d","price":"15.00"}`,i)
  rec:=postProduct(api,scope,body)
  if rec.Code!=http.StatusCreated{t.Fatalf("fixture product %d: %d %s",i,rec.Code,rec.Body.String())}
 }
 handler:=http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  r=r.WithContext(context.WithValue(r.Context(),scopeKey{},scope))
  api.listProducts(w,r)
 })
 server:=httptest.NewServer(handler)
 defer server.Close()
 client:=&http.Client{Timeout:15*time.Second,Transport:&http.Transport{MaxIdleConns:256,MaxIdleConnsPerHost:256,MaxConnsPerHost:256}}
 defer client.CloseIdleConnections()
 url:=server.URL+"/v1/admin/products?page=1&pageSize=20"
 req,err:=http.NewRequest(http.MethodGet,url,nil)
 if err!=nil{t.Fatal(err)}
 resp,err:=client.Do(req)
 if err!=nil{t.Fatal(err)}
 _,_ = io.Copy(io.Discard,resp.Body)
 _=resp.Body.Close()
 if resp.StatusCode!=http.StatusOK{t.Fatalf("warmup returned HTTP %d",resp.StatusCode)}
 report:=capacityReport{Scenario:"HTTP product list, real PostgreSQL, test-only scope injector",Commit:os.Getenv("GITHUB_SHA"),Environment:"GitHub Actions shared runner; PostgreSQL 16 container"}
 for _,stage:=range []struct{workers,requests int}{{1,80},{8,320},{32,640},{64,640}}{
  jobs:=make(chan struct{},stage.requests)
  for i:=0;i<stage.requests;i++{jobs<-struct{}{}}
  close(jobs)
  durations:=make(chan time.Duration,stage.requests)
  results:=make(chan int,stage.requests)
  var wg sync.WaitGroup
  started:=time.Now()
  for i:=0;i<stage.workers;i++{
   wg.Add(1)
   go func(){
    defer wg.Done()
    for range jobs{
     t0:=time.Now()
     response,err:=client.Get(url)
     elapsed:=time.Since(t0)
     status:=0
     if err==nil{
      status=response.StatusCode
      _,err=io.Copy(io.Discard,response.Body)
      _=response.Body.Close()
     }
     if err!=nil{status=0}
     durations<-elapsed
     results<-status
    }
   }()
  }
  wg.Wait()
  elapsed:=time.Since(started)
  close(durations)
  close(results)
  latencies:=make([]time.Duration,0,stage.requests)
  for d:=range durations{latencies=append(latencies,d)}
  sort.Slice(latencies,func(i,j int)bool{return latencies[i]<latencies[j]})
  passed:=0
  for status:=range results{if status==http.StatusOK{passed++}}
  entry:=capacityResult{Workers:stage.workers,Requests:stage.requests,Success:passed,Errors:stage.requests-passed,RPS:float64(stage.requests)/elapsed.Seconds(),P50MS:capacityPercentile(latencies,0.5),P95MS:capacityPercentile(latencies,0.95),P99MS:capacityPercentile(latencies,0.99),ElapsedMS:float64(elapsed.Milliseconds())}
  report.Stages=append(report.Stages,entry)
  t.Logf("CAPACITY workers=%d total=%d success=%d errors=%d rps=%.1f p50=%.2fms p95=%.2fms p99=%.2fms",entry.Workers,entry.Requests,entry.Success,entry.Errors,entry.RPS,entry.P50MS,entry.P95MS,entry.P99MS)
  if entry.Errors>0{t.Errorf("%d/%d product read requests failed with %d workers",entry.Errors,entry.Requests,entry.Workers)}
 }
 if data,err:=json.MarshalIndent(report,"","  ");err!=nil{t.Error(err)}else if err:=os.WriteFile("capacity-report.json",data,0600);err!=nil{t.Error(err)}
}
