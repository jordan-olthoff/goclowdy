package main
// go build grsc.go
import (
  "fmt"
  "context"
  "os" // Args
  "google.golang.org/api/iterator"
  compute "cloud.google.com/go/compute/apiv1"
  computepb "cloud.google.com/go/compute/apiv1/computepb"
  goc "mymodule/goc"
  macv "mymodule/goc/macv"
)

var verdict = [...]string{"KEEP to be safe", "KEEP Recent (<1W)", "KEEP (one-per-week)", "DELETE (>1W)", "DELETE (> 1Y)"}

func main() {
  //ctx := context.Background()
  if len(os.Args) < 2 { fmt.Println("Pass sc: v,m"); return }
  //if () {}
  pname := os.Getenv("GCP_PROJECT")
  if pname == "" { fmt.Println("No project indicated"); return }
  if os.Args[1] == "vmlist" { vm_ls(pname) } else if os.Args[1] == "milist" { mi_ls(pname) }
  return
}

func vm_ls(pname string) {
    ctx := context.Background()
    // test overlapping sysm (old: vs)
    vmc := macv.CC{Project: pname}
    vmc.Init()
    all := vmc.GetAll()
    //fmt.Println(all)
    icnt := len(all)
    if icnt == 0 { fmt.Println("No VMs found"); return }
    fmt.Printf("Got %v Initial Instances (Filtering ...)\n", icnt)
    // Test MI overlaps
    cfg := goc.CC{Project: pname,  WD_keep: 5, KeepMinH: 168,  TZn: "Europe/London"} // tnow: tnow, tloc: loc
    cfg.Init()
    c, err := compute.NewMachineImagesRESTClient(ctx)
    if err != nil { return }
    for _, it := range all{ // Instance
      fmt.Println("vname:"+it.GetName())
      in := goc.StdName(it.GetName())
      fmt.Println("STD Name:", in)
      mi := cfg.GetOne(in, c)
      if mi != nil  {
        fmt.Println("Found image: ", mi.GetName())
        cfg.Delete(mi, c)
      } else { fmt.Println("No (std) image found for : ", it.GetName()) }
    }
    return
}
func mi_ls(pname string) {
  ctx := context.Background()
  // CredF: "~/gcpServiceAccountKey.json",
  cfg := goc.CC{Project: pname,  WD_keep: 5, KeepMinH: 168,  TZn: "Europe/London"} // tnow: tnow, tloc: loc
  cfg.Init()
  c, err := compute.NewMachineImagesRESTClient(ctx)  // , option.WithCredentialsFile(cfg.CredF)
  if err != nil {
    // NOT:fmt.Println("Error creating client "+err.msg)
    fmt.Println(err)
    //log.Fatal(err)
    return
  }
  var maxr uint32 = 20
  if cfg.Project == "" { fmt.Println("No Project passed"); return }
  req := &computepb.ListMachineImagesRequest{
    Project: cfg.Project,
    MaxResults: &maxr } // Filter: &mifilter } // 
  //fmt.Println("Search MI from: "+cfg.Project+", parse by: "+time.RFC3339)
  it := c.List(ctx, req)  
  for {
    mi, err := it.Next()
    if err == iterator.Done { fmt.Println("Iter done"); break }
    if mi == nil {  fmt.Println("No mi"); break }
    
    fmt.Println("MI:"+mi.GetName()+" "+mi.GetCreationTimestamp())
    var cl int = goc.Classify(mi, &cfg)
    fmt.Println(verdict[cl])
    if goc.To_be_deleted(cl) {
      //Delete(mi, c)
    }
  }
}