package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/progrium/macdriver/cocoa"
	"github.com/progrium/macdriver/core"
	"github.com/progrium/macdriver/objc"
)

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -lobjc -framework Foundation -framework ApplicationServices
#include <Foundation/Foundation.h>
#include <Carbon/Carbon.h>

bool CheckProcessIsTrusted() {
	NSDictionary *options = @{(id)kAXTrustedCheckOptionPrompt: @YES};
	return AXIsProcessTrustedWithOptions((CFDictionaryRef)options);
}

*/
import "C"

func init() {
	runtime.LockOSThread()
}

// Options represents the runtime options for keyshow.
type Options struct {
	Screen int
}

func main() {
	opt := Options{}
	flag.IntVar(&opt.Screen, "screen", 0, "Which screen to render on")
	flag.Parse()
	if err := run(opt); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(opt Options) error {
	runtime.LockOSThread()
	if !C.CheckProcessIsTrusted() {
		return fmt.Errorf("you must enable accessibility services to enable keyshow.")
	}
	screens := cocoa.NSScreen_Screens()
	if opt.Screen > len(screens)-1 {
		opt.Screen = 0
	}
	screenSize := screens[opt.Screen].Frame().Size
	fmt.Println("screen size:", screenSize)

	app := cocoa.NSApp_WithDidLaunch(func(n objc.Object) {
		fontName := "Helvetica"

		//text := fmt.Sprintf(" %s ", strings.Join(flag.Args(), " "))
		text := "keyshow started"
		tr, fontSize := func() (rect core.NSRect, size float64) {
			t := cocoa.NSTextView_Init(core.Rect(0, 0, 0, 0))
			t.SetString(text)
			for s := 30.0; s <= 550; s += 12 {
				t.SetFont(cocoa.Font(fontName, s))
				t.LayoutManager().EnsureLayoutForTextContainer(t.TextContainer())
				rect = t.LayoutManager().UsedRectForTextContainer(t.TextContainer())
				size = s
				if rect.Size.Width >= screenSize.Width*0.4 {
					break
				}
			}
			return rect, size
		}()

		height := tr.Size.Height * 1.2
		// tr.Origin.Y = height - tr.Size.Height
		tr.Origin.Y = (height / 2) - (tr.Size.Height / 2)
		//tr.Origin.Y = (height / 2) - (tr.Size.Height / 2)
		t := cocoa.NSTextView_Init(tr)
		t.SetString(text)
		t.SetFont(cocoa.Font(fontName, fontSize))
		t.SetTextColor(cocoa.Color(1, 1, 1, 0.8))
		t.SetEditable(false)
		t.SetImportsGraphics(false)
		t.SetDrawsBackground(false)

		c := cocoa.NSView_Init(core.Rect(0, 0, 0, 0))
		c.SetBackgroundColor(cocoa.Color(0, 0, 0, 0.70))
		c.SetWantsLayer(true)
		c.Layer().SetCornerRadius(32.0)
		c.AddSubviewPositionedRelativeTo(t, cocoa.NSWindowAbove, nil)

		tr.Size.Height = height
		//tr.Origin.X = (screen.Width / 2) - (tr.Size.Width / 2)
		//tr.Origin.X = 20
		//tr.Origin.Y = (screenSize.Height / 2) - (tr.Size.Height / 2)
		//tr.Origin.Y = screenSize.Height - tr.Size.Height
		// tr.Origin.Y = screenSize.Height - tr.Size.Height
		tr.Origin.Y = (screenSize.Height / 2) - (tr.Size.Height / 2)

		w := cocoa.NSWindow_Init(core.Rect(0, 0, 0, 0),
			cocoa.NSBorderlessWindowMask, cocoa.NSBackingStoreBuffered, false)
		w.SetContentView(c)
		w.SetTitlebarAppearsTransparent(true)
		w.SetTitleVisibility(cocoa.NSWindowTitleHidden)
		w.SetOpaque(false)
		w.SetBackgroundColor(cocoa.NSColor_Clear())
		w.SetLevel(cocoa.NSMainMenuWindowLevel + 2)
		w.SetFrameDisplay(tr, true)
		w.MakeKeyAndOrderFront(nil)

		/*
			go func() {
				for {
					// s := fmt.Sprint(time.Now().Nanosecond() / 1000000)
					// fmt.Println("set string to ", s)
					// t.SetString(s)
					//c.AddSubviewPositionedRelativeTo(t, cocoa.NSWindowAbove, nil)
					//w.SetContentView(c)
					time.Sleep(1000 * time.Millisecond)
				}
			}()
		*/

		events := make(chan cocoa.NSEvent, 64)
		go func() {
			text = ""
			for {
				select {
				case <-time.After(time.Second):
					text = ""
				case e := <-events:
					if e.Type() != cocoa.NSEventTypeKeyDown {
						continue
					}
					// fmt.Println("got event:", e)
					s, err := e.Characters()
					if err == nil {
						text += s
					}
					if strings.HasSuffix(text, "wq") {
						cocoa.NSApp().Terminate()
					}
					// cocoa.NSApp().Terminate()
				}
				t.SetString(text)
				c.AddSubviewPositionedRelativeTo(t, cocoa.NSWindowAbove, nil)
			}
		}()
		cocoa.NSEvent_GlobalMonitorMatchingMask(cocoa.NSEventMaskAny, events)
	})
	app.ActivateIgnoringOtherApps(true)
	app.Run()
	return nil
}
