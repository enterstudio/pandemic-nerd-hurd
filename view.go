package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/anthonybishopric/pandemic-nerd-hurd/pandemic"
	"github.com/fatih/color"
	"github.com/jroimartin/gocui"
)

type PandemicView struct {
	logger       *logrus.Logger
	colorAllGood func(...interface{}) string
	colorWarning func(...interface{}) string
	colorOhFuck  func(...interface{}) string
}

func NewView(logger *logrus.Logger) *PandemicView {
	return &PandemicView{
		logger:       logger,
		colorAllGood: color.New(color.FgGreen).Add(color.BgBlack).SprintFunc(),
		colorWarning: color.New(color.FgYellow).Add(color.BgBlack).SprintFunc(),
		colorOhFuck:  color.New(color.FgBlack).Add(color.BgRed).Add(color.BlinkRapid).SprintFunc(),
	}
}

func (p *PandemicView) Start(game *pandemic.GameState) {
	gui := gocui.NewGui()

	if err := gui.Init(); err != nil {
		p.logger.Errorln("Could not init GUI: %v", err)
	}
	defer gui.Close()

	gui.SetLayout(func(gui *gocui.Gui) error {
		width, height := gui.Size()

		p.renderCommandsView(game, gui, width)
		p.renderStriations(game, gui, 2, height/2, width)
		// p.renderTurnStatus(game, gui, 0, height/2+1, width/2, height)
		p.renderConsoleArea(game, gui, width/2+1, height/2+1, width, height)

		p.setUpKeyBindings(game, gui, "Commands")
		gui.Cursor = true
		gui.SetCurrentView("Commands")
		gui.Editor = gocui.DefaultEditor
		return nil
	})

	if err := gui.MainLoop(); err != nil && err != gocui.ErrQuit {
		gui.Close()
		p.logger.Fatalf("Error in game main loop: %v", err)
	}
}

func (p *PandemicView) renderCommandsView(game *pandemic.GameState, gui *gocui.Gui, maxX int) {
	commandView, err := gui.SetView("Commands", 0, 0, maxX, 2)
	if err != nil && err != gocui.ErrUnknownView {
		gui.Close()
		p.logger.Fatalf("Could not render command view")
	}
	commandView.Editable = true
	commandView.Title = "Commands"
}

func (p *PandemicView) terminateIfErr(err error, msg string, gui *gocui.Gui) {
	if err != nil && err != gocui.ErrUnknownView {
		gui.Close()
		p.logger.Fatalf("%v: %v", msg, err)
	}
}

func (p *PandemicView) setUpKeyBindings(game *pandemic.GameState, gui *gocui.Gui, commandView string) {
	err := gui.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {
		// when we get a ctrl-C we exit the game
		gui.Close()
		p.logger.Fatalf("Buh bye") // TODO: save
		return nil
	})
	p.terminateIfErr(err, "could not establish graceful termination keybinding", gui)
	err = gui.SetKeybinding(commandView, gocui.KeyEnter, gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {
		consoleView, err := gui.View("Console")
		if err != nil {
			gui.Close()
			p.logger.Fatalln("Console view not found, game view not set up correctly")
			return nil
		}
		return p.runCommand(game, consoleView, view)
	})
	p.terminateIfErr(err, "could not establish keybinding for command view", gui)
}

func (p *PandemicView) renderConsoleArea(game *pandemic.GameState, gui *gocui.Gui, topX, topY, bottomX, bottomY int) {
	view, err := gui.SetView("Console", topX, topY, bottomX, bottomY)
	p.terminateIfErr(err, "Could not set up console view", gui)
	view.Wrap = true
}

// Creates a series of columns, representing the current infection deck striations. Striations closer
// to the top of the infection deck are further to the right. Cities are colored based on the probability
// of being drawn.
func (p *PandemicView) renderStriations(game *pandemic.GameState, gui *gocui.Gui, topY int, bottomY int, maxX int) error {
	// We know there will never be more than 4 striations, not including drawn.
	// Divide the horizontal space by 4 and make striations that width.
	strWidth := int(math.Floor(float64(maxX) / 4.0))
	p.logger.Infoln(fmt.Sprintf("%+v", len(game.InfectionDeck.Striations)))

	for i := len(game.InfectionDeck.Striations) - 1; i >= 0; i-- {
		widthMultiplier := len(game.InfectionDeck.Striations) - i - 1
		striation := game.InfectionDeck.Striations[i]
		cityNames := striation.Members()
		strName := fmt.Sprintf("Striation %v", i)
		strView, err := gui.SetView(strName, strWidth*widthMultiplier, topY, (widthMultiplier+1)*strWidth, bottomY)
		if err != nil {
			return err
		}
		strView.Clear()
		strView.Title = strName
		for _,city := range cityNames {
			probability := game.ProbabilityOfCity(city)

			text := fmt.Sprintf("%v %.2f", city, probability)
			if probability == 0.0 {
				fmt.Fprintln(strView, p.colorAllGood(text))
			} else if probability > 0.8 {
				fmt.Fprintln(strView, p.colorOhFuck(text))
			} else {
				fmt.Fprintln(strView, p.colorWarning(text))
			}
		}
	}
	return nil
}

func (p *PandemicView) runCommand(gameState *pandemic.GameState, consoleView *gocui.View, commandView *gocui.View) error {
	commandBuffer := strings.Trim(commandView.Buffer(), "\n\t\r ")
	if commandBuffer == "" {
		return nil
	}

	commandArgs := strings.Split(commandBuffer, " ")
	cmd := commandArgs[0]
	args := commandArgs[1]

	switch cmd {
	case "infect", "i":
		err := gameState.InfectionDeck.Draw(args)
		if err != nil {
			fmt.Fprintln(consoleView, p.colorWarning(err))
		} else {
			fmt.Fprintf(consoleView, "Infected %v\n", args)
		}
	default:
		fmt.Fprintf(consoleView, p.colorWarning(fmt.Sprintf("Unrecognized command %v\n", cmd)))
	}

	commandView.Clear()
	return nil
}
