// Deletion Policy parametes
//                     (1 Y+1 w)            (1 w)
// Abs.Age            KeepMaxH             KeepMinH         Now
// <---------------------|--------------------|---------------|
//  keep none/del all    |<-- keep 1/week  -->|  keep all     |
package main

// go.formatOnSave
// editor.formatOnSave
// go build grsc.go
import (
  "context"
  "fmt"
  "os" // Args  
  "google.golang.org/api/iterator"  
  //compute "cloud.google.com/go/compute/apiv1" // Used only in lower levels
  computepb "cloud.google.com/go/compute/apiv1/computepb"
  //goc "VMs"
  //macv "MIs"
  VMs "github.com/ohollmen/goclowdy/VMs"
  MIs "github.com/ohollmen/goclowdy/MIs"
  //"github.com/ohollmen/goclowdy"
  //"goclowdy/vm/VMs"
  //"goclowdy/mi/MIs"
  //MIs "goclowdy/mi"
  //VMs "goclowdy/vm" 
  //"goclowdy/VMs"
  //"goclowdy/MIs"
  "path" // Base  
  "google.golang.org/api/iam/v1"
  // NEW
  "regexp"
)

var verdict = [...]string{"KEEP to be safe", "KEEP NEW (< KeepMinH)", "KEEP (MID-TERM, WEEKLY)", "DELETE (MID-TERM)", "DELETE OLD (> KeepMaxH)"}

func main() {
  //ctx := context.Background()
  if len(os.Args) < 2 { fmt.Println("Pass one of subcommands: vmlist,midel,keylist"); return }
  //if () {}
  pname := os.Getenv("GCP_PROJECT")
  if pname == "" { fmt.Println("No project indicated (by GCP_PROJECT)"); return }
  if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" { fmt.Println("No creds given by (by GOOGLE_APPLICATION_CREDENTIALS)"); return }
  if os.Args[1] == "vmlist" {
    vm_ls(pname)
  } else if os.Args[1] == "midel" {
    mi_del(pname)
  } else if os.Args[1] == "keylist" {
    key_list(pname)
  } else { fmt.Println("Pass one of subcommands: vmlist,milist,keylist"); return }
  return
}

func vm_ls(pname string) {
    //ctx := context.Background()
    // test overlapping sysm (old: vs)
    vmc := VMs.CC{Project: pname}
    vmc.Init()
    all := vmc.GetAll()
    //fmt.Println(all)
    icnt := len(all)
    if icnt == 0 { fmt.Println("No VMs found"); return }
    fmt.Printf("Got %v Initial Instances (Filtering ...)\n", icnt)
    // Test for daily MI. This is now embedded to mic.CreateFrom() logic.
    mic := MIs.CC{Project: pname,  WD_keep: 5, KeepMinH: 168,  TZn: "Europe/London", Debug: true} // tnow: tnow, tloc: loc
    mic.Init()
    for _, it := range all{ // Instance
      fmt.Println("vname:"+it.GetName())
      in := MIs.StdName(it.GetName())
      fmt.Println("STD Name:", in)
      mi := mic.GetOne(in)
      if mi != nil  {
        fmt.Println("Found image: ", mi.GetName())
        //mic.Delete(mi)
      } else { fmt.Println("No (std) image found for : ", it.GetName()) }
    }
    return
}

// Delet machine images per given config policy
func mi_del(pname string) {
  ctx := context.Background()
  midel := os.Getenv("MI_DELETE_EXEC")
  // https://pkg.go.dev/regexp
  var stdnamere * regexp.Regexp // Regexp
  var err error
  // E.g. "^\\w+-\\d{1,3}-\\d{1,3}-\\d{1,3}-\\d{1,3}-\\d{4}-\\d{2}-\\d{2}" (in Go runtime)
  // E.g. "^[a-z0-9-]+?-\\d{1,3}-\\d{1,3}-\\d{1,3}-\\d{1,3}-\\d{4}-\\d{2}-\\d{2}" // No need for 1) \\ before [..-] 2) \ before [ / ]
  if (os.Getenv("MI_STDNAME") != "") {
    stdnamere, err = regexp.Compile(os.Getenv("MI_STDNAME")) // (*Regexp, error) // Also MustCompile
    if err != nil { fmt.Println("Cannot compile STD name RegExp"); return }
    //stdm := stdnamere.MatchString( "myhost-00-00-00-00-1900-01-01" ); // Also reg.MatchString() reg.FindString() []byte()
    //if !stdm { fmt.Println("STD Name re not matching "); return }
  }
  
  //return
  // 168 h = 1 = week, (24 * (365 + 7)) hours = 1 year,  weekday 5 = Friday (wdays: 0=Sun... 6=Sat)
  mic := MIs.CC{Project: pname,  WD_keep: 5, KeepMinH: 168,  KeepMaxH: (24 * (365 + 7)), TZn: "Europe/London"} // tnow: tnow, tloc: loc
  mic.Init()
  if midel != "" { mic.DelOK = true; } // Non-empty => DELETE
  var maxr uint32 = 20
  if mic.Project == "" { fmt.Println("No Project passed"); return }
  req := &computepb.ListMachineImagesRequest{
    Project: mic.Project,
    MaxResults: &maxr } // Filter: &mifilter } // 
  //fmt.Println("Search MI from: "+cfg.Project+", parse by: "+time.RFC3339)
  it := mic.Client().List(ctx, req)
  if it == nil { fmt.Println("No mis from "+mic.Project); }
  // Iterate MIs, check for need to del
  for {
    //fmt.Println("Next ...");
    mi, err := it.Next()
    if err == iterator.Done { fmt.Println("Iter done"); break }
    if mi == nil {  fmt.Println("No mi. check (actual) creds, project etc."); break }
    
    fmt.Println("MI:"+mi.GetName()+" (Created: "+mi.GetCreationTimestamp()+")")
    if (stdnamere != nil) && (!stdnamere.MatchString( mi.GetName() )) {
      fmt.Printf("  - NON-STD-NAME: %s\n", mi.GetName())
      continue;
    }
    //var cl int = MIs.Classify(mi, &mic)
    var cl int = mic.Classify(mi)
    fmt.Println(verdict[cl])
    if MIs.ToBeDeleted(cl) {
      fmt.Printf("DELETE %s\n", mi.GetName()) // Also in DRYRUN
      if mic.DelOK {
        
        err := mic.Delete(mi)
         if err != nil {
          fmt.Printf("Error Deleting MI: %s", mi.GetName())
        } else {
          fmt.Printf("Deleted %s\n", mi.GetName())
        }
        //fmt.Printf("Should have deleted %s. Set DelOK (MI_DELETE_EXEC) to actually delete.\n", mi.GetName())
      } 
    } else {
      fmt.Printf("KEEP %s\n", mi.GetName())
    }
    fmt.Printf("============\n")
  }
}

func key_list(pname string) {
  ctx := context.Background()
  iamService, err := iam.NewService(ctx)
  if err != nil { fmt.Println("No Service"); return }
  acct := os.Getenv("GCP_SA")
  pname = os.Getenv("GCP_SA_PROJECT") // override
  if acct == "" { fmt.Println("No GCP_SA"); return }
  if pname == "" { fmt.Println("No GCP_SA_PROJECT"); return }
  sapath := fmt.Sprintf( "projects/%s/serviceAccounts/%s", pname,  acct)
  resp, err := iamService.Projects.ServiceAccounts.Keys.List(sapath).Context(ctx).Do()
  if err != nil { fmt.Println("No Keys %v", err); return }
  //fmt.Println("Got:", resp) // iam.ListServiceAccountKeysResponse
  fmt.Printf("%T\n", resp) // import "reflect" fmt.Println(reflect.TypeOf(tst))
  for _, key := range resp.Keys {
    fmt.Printf("%T\n", key) // iam.ServiceAccountKey
    fmt.Printf("%v Exp.: %s\n", path.Base(key.Name), key.ValidBeforeTime)
  }
  // Also: SignJwtRequest, but: https://cloud.google.com/iam/docs/migrating-to-credentials-api
}
