//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
 "os"
 "path/filepath"
 "runtime"
 "syscall"
 "testing"
 "unsafe"
 "github.com/feiniao233/MyCast/internal/core"
)

func TestNativeSourcePIDLOwnershipAndVirtualRefusal(t *testing.T){
 if ShowApplicationLocation(core.AppsFolderParsingPrefix+"Synthetic!App")==nil{t.Fatal("namespace sent to filesystem Explorer adapter")}
 runtime.LockOSThread();defer runtime.UnlockOSThread()
 hr,_,_:=coInitializeEx.Call(0,2)
 if err:=comError("fixture apartment",hr);err!=nil{t.Fatal(err)};defer coUninitialize.Call()
 path:=filepath.Join(t.TempDir(),"Synthetic.exe")
 if err:=os.WriteFile(path,[]byte("inert"),0600);err!=nil{t.Fatal(err)}
 wide,_:=syscall.UTF16PtrFromString(path)
 for n:=0;n<32;n++{
  var pidl uintptr
  hr,_,_=parseShellDisplayName.Call(uintptr(unsafe.Pointer(wide)),0,uintptr(unsafe.Pointer(&pidl)),0,0)
  if err:=comError("fixture PIDL",hr);err!=nil{t.Fatal(err)}
  if pidl==0{t.Fatal("missing source PIDL")}
  freeCOMMemory.Call(pidl)
 }
 // Explorer navigation itself is an actual-user check, not performed by tests.
}
