// SPDX-License-Identifier: AGPL-3.0-or-later
package settings

import (
 "path/filepath"
 "testing"
 "github.com/feiniao233/MyCast/internal/core"
)

func TestHiddenIDsBoundedClonedAndPersistent(t *testing.T){
 id:=(core.Application{Path:"Synthetic.exe"}).ItemID()
 value:=Defaults();value.Hidden=[]string{id,id}
 value,err:=value.Validated();if err!=nil||len(value.Hidden)!=1{t.Fatal("hidden validation")}
 clone:=value.Clone();clone.Hidden[0]="bad"
 if value.Hidden[0]!=id{t.Fatal("clone shared hidden list")}
 store:=&Store{Path:filepath.Join(t.TempDir(),"settings.json")}
 if err:=store.Save(value);err!=nil{t.Fatal(err)}
 loaded,err:=(&Store{Path:store.Path}).Load();if err!=nil||len(loaded.Hidden)!=1||loaded.Hidden[0]!=id{t.Fatal("hidden round trip")}
 value.Hidden=make([]string,201)
 if _,err:=value.Validated();err==nil{t.Fatal("unbounded hidden list")}
 value.Hidden=[]string{"bad"}
 if _,err:=value.Validated();err==nil{t.Fatal("malformed hidden ID")}
}
