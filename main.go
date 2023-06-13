package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
	"github.com/olekukonko/ts"
)

const localVersion = "1.6.8"

var bold = color.New(color.Bold)
var boldBlue = color.New(color.Bold, color.FgBlue)
var boldViolet = color.New(color.Bold, color.FgMagenta)
var codeText = color.New(color.BgBlack, color.FgGreen, color.Bold)
var stopSpin = false

var programLoop = true
var serverID = ""
var configDir = ""
var userInput = ""
var executablePath = ""

func main() {
	execPath, err := os.Executable()
	if err == nil {
		executablePath = execPath
	}
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-terminate
		os.Exit(0)
	}()

	hasConfig := true
	configDir, err = os.UserConfigDir()

	if err != nil {
		hasConfig = false
	}
	configTxtByte, err := os.ReadFile(configDir + "/tgpt/config.txt")
	if err != nil {
		hasConfig = false
	}
	chatId := ""
	if hasConfig {
		configArr := strings.Split(string(configTxtByte), ":")
		if len(configArr) == 2 {
			chatId = configArr[1]
		}
	}
	args := os.Args

	if len(args) > 1 && len(args[1]) > 0 {
		input := args[1]

		if input == "-s" {
			if len(args) > 2 && len(args[2]) > 1 {
				prompt := args[2]
				go loading(&stopSpin)
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 0 {
					fmt.Println("You need to provide some text")
					fmt.Println(`Example: tgpt -s "How to update system"`)
					os.Exit(0)
				}
				shellCommand(trimmedPrompt)
			} else {
				fmt.Println("You need to provide some text")
				fmt.Println(`Example: tgpt -s "How to update system"`)
				os.Exit(0)
			}

		} else if input == "-c" {
			if len(args) > 2 && len(args[2]) > 1 {
				prompt := args[2]
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 0 {
					fmt.Println("You need to provide some text")
					fmt.Println(`Example: tgpt -c "Hello world in Python"`)
					os.Exit(0)
				}
				codeGenerate(trimmedPrompt)
			} else {
				fmt.Println("You need to provide some text")
				fmt.Println(`Example: tgpt -c "Hello world in Python"`)
				os.Exit(0)
			}
		} else if input == "-i" {
			/////////////////////
			// Normal interactive
			/////////////////////

			reader := bufio.NewReader(os.Stdin)
			serverID = chatId
			for {
				boldBlue.Println("╭─ You")
				boldBlue.Print("╰─> ")

				input, err := reader.ReadString('\n')
				if err != nil {
					fmt.Println("Error reading input:", err)
					break
				}

				if len(input) > 0 {
					input = strings.TrimSpace(input)
					if len(input) > 0 {
						if input == "exit" {
							bold.Println("Exiting...")
							return
						}
						serverID = getData(input, serverID, configDir+"/tgpt", true)
					}

				}

			}

		} else if input == "-m" {
			/////////////////////
			// Multiline interactive
			/////////////////////
			serverID = chatId

			fmt.Print("\nPress Tab to submit and Ctrl + C to exit.\n")

			for programLoop {
				fmt.Print("\n")
				p := tea.NewProgram(initialModel())
				_, err := p.Run()

				if err != nil {
					fmt.Println(err)
					os.Exit(0)
				}
				if len(userInput) > 0 {
					serverID = getData(userInput, serverID, configDir+"/tgpt", true)
				}

			}

		} else {
			go loading(&stopSpin)
			formattedInput := strings.TrimSpace(input)
			getData(formattedInput, chatId, configDir+"/tgpt", false)
		}
		
	} else {
			color.Red("You have to write some text")
			color.Blue(`Example: tgpt "Explain quantum computing in simple terms"`)
	}
}

// Multiline input

type errMsg error

type model struct {
	textarea textarea.Model
	err      error
}

func initialModel() model {
	size, _ := ts.GetSize()
	termWidth := size.Col()
	ti := textarea.New()
	ti.SetWidth(termWidth)
	ti.CharLimit = 200000
	ti.ShowLineNumbers = false
	ti.Placeholder = "Enter your prompt"
	ti.Focus()

	return model{
		textarea: ti,
		err:      nil,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			if m.textarea.Focused() {
				m.textarea.Blur()
			}
		case tea.KeyCtrlC:
			programLoop = false
			userInput = ""
			return m, tea.Quit

		case tea.KeyTab:
			userInput = m.textarea.Value()

			if len(userInput) > 0 {
				m.textarea.Blur()
				return m, tea.Quit
			}

		default:
			if !m.textarea.Focused() {
				cmd = m.textarea.Focus()
				cmds = append(cmds, cmd)
			}
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	return m.textarea.View()
}

//////////////////////////////
