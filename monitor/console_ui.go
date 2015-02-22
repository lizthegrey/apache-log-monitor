package monitor

import (
	"fmt"
	"github.com/jroimartin/gocui"
	"github.com/nsf/termbox-go"
)

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("status", 0, 0, maxX-1, 2); err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
		fmt.Fprintln(v, "Waiting for initial status.")
	}
	if v, err := g.SetView("log", 0, 2, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
		v.Autoscroll = true
	}
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.Quit
}

type Console struct {
	g      *gocui.Gui
	status chan string
	log    chan string
}

func NewConsole(status chan string, logEntry chan string) (*Console, error) {
	g := gocui.NewGui()
	if err := g.Init(); err != nil {
		return nil, err
	}
	g.SetLayout(layout)
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		g.Close()
		return nil, err
	}
	g.Flush()
	return &Console{
		g:      g,
		status: status,
		log:    logEntry,
	}, nil
}

func (c *Console) Loop() error {
	go func() {
		v, err := c.g.View("status")
		if err != nil {
			return
		}
		for status := range c.status {
			v.Clear()
			fmt.Fprintln(v, status)
			c.g.Flush()
		}
		fmt.Fprintln(v, "Status channel closed.")
		c.g.Flush()
	}()

	go func() {
		v, err := c.g.View("log")
		if err != nil {
			return
		}
		for line := range c.log {
			fmt.Fprintln(v, line)
			c.g.Flush()
		}
		fmt.Fprintln(v, "No more log entries.")
		c.g.Flush()
	}()

	if err := c.g.SetCurrentView("log"); err != nil {
		return err
	}

	if err := c.g.MainLoop(); err != gocui.Quit {
		return err
	}
	return nil
}

func (c *Console) Interrupt() {
	termbox.Interrupt()
}

func (c *Console) Cleanup() {
	c.g.Close()
	close(c.status)
	close(c.log)
}
