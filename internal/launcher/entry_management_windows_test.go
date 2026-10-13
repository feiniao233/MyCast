//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
 "errors"
 "fmt"
 "strings"
 "testing"
 "github.com/egoist/mygo/ui"
 "github.com/feiniao233/MyCast/internal/core"
 "github.com/feiniao233/MyCast/internal/settings"
)

func TestEntryHideRestoreAndStaleActions(t *testing.T) {
 for _,scale:=range []float32{1,1.5}{t.Run(fmt.Sprint(scale),func(t *testing.T){
  s:=syntheticLauncher();s.config=settings.Defaults();s.apps=s.apps[:3]
  s.persistPreferences=func(v settings.Settings)(settings.Settings,error){return v,nil}
  original:=s.apps[0];launches,locations:=0,0
  s.launchApplication=func(string)error{launches++;return nil}
  s.showLocation=func(path string)error{locations++;if path!=original.Path{t.Fatal("location replaced target")};return nil}
  tester:=newPaletteTester(s,scale)
  s.openMenu(menuActions);tester.Frame()
  for _,label:=range []string{"菜单项：查看来源","菜单项：打开所在位置","菜单项：隐藏此入口"}{box,ok:=tester.Find(label);if !ok||box.Y<palette.Header||box.Y+box.H>palette.Height-palette.Footer{t.Fatalf("menu item clipped %s %+v",label,box)}}
  savePreview(t,tester,fmt.Sprintf("entry-menu-%03d",int(scale*100)))
  tester.Click("菜单项：打开所在位置")
  if locations!=1||launches!=0{t.Fatal("location launched target")}
  s.openMenu(menuActions);tester.Frame();tester.Click("菜单项：查看来源")
  if !s.statusDetailsOpen||!strings.Contains(s.statusDetail,original.Path){t.Fatal("missing original source")}
  savePreview(t,tester,fmt.Sprintf("entry-source-%03d",int(scale*100)))
  tester.Click("关闭状态详情")
  s.openMenu(menuActions);tester.Frame();tester.Click("菜单项：隐藏此入口")
  if len(s.config.Hidden)!=1||len(s.applicationResults())!=2||!s.appNeedsSelection{t.Fatal("hide failed or selection guard missing")}
  tester.Key(0,ui.KeyEnter)
  if launches!=0{t.Fatal("old Enter launched hidden replacement")}
  // A stored closure from the old menu cannot open the new first result.
  s.runMenuTarget(original.ItemID(),false,"location")
  if locations!=1{t.Fatal("stale location action transferred")}
  s.openSettings();tester.Frame();tester.Click("管理隐藏入口")
  if !s.hiddenOpen{t.Fatal("restore manager did not open")}
  savePreview(t,tester,fmt.Sprintf("entry-hidden-%03d",int(scale*100)))
  if err:=tester.Click("恢复入口："+original.Name);err!=nil{t.Fatal(err)}
  if len(s.config.Hidden)!=0||len(s.applicationResults())!=3{t.Fatal("restore failed")}
  tester.Click("关闭隐藏入口")
  if !tester.Focused("搜索"){t.Fatal("restore lost search focus")}
 })}
}

func TestVirtualSourceLocationAndHideSaveFailure(t *testing.T){
 s:=syntheticLauncher();s.config=settings.Defaults()
 app:=core.Application{Name:"Synthetic Virtual",ShellID:core.AppsFolderParsingPrefix+"Synthetic!App"};s.apps=[]core.Application{app}
 s.showLocation=func(string)error{t.Fatal("virtual location passed to filesystem adapter");return nil}
 s.openMenu(menuActions)
 for _,item:=range s.menuItems(nil,nil){if item.text=="打开所在位置"&&!item.disabled{t.Fatal("virtual location enabled")}}
 s.runMenuTarget(app.ItemID(),false,"location")
 s.openApplicationSource(app)
 if !strings.Contains(s.statusDetail,"AppsFolder")||!strings.Contains(s.statusDetail,app.ShellID){t.Fatal("missing virtual source")}
 s.persistPreferences=func(settings.Settings)(settings.Settings,error){return settings.Settings{},errors.New("synthetic save refusal")}
 s.toggleHiddenApplication(app.ItemID())
 if len(s.config.Hidden)!=0||len(s.applicationResults())!=1{t.Fatal("failed save changed visibility")}
}
