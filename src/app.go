package cmd

import (
	"fmt"
	"net/http"
	"os"
	"time"

	focus "github.com/ayoisaiah/focus/src/internal"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"
)

func init() {
	// Override the default help template
	cli.AppHelpTemplate = helpText()

	// Override the default version printer
	oldVersionPrinter := cli.VersionPrinter
	cli.VersionPrinter = func(c *cli.Context) {
		oldVersionPrinter(c)
		checkForUpdates(GetApp())
	}

	// Disable colour output if NO_COLOR is set
	if _, exists := os.LookupEnv("NO_COLOR"); exists {
		disableStyling()
	}

	// Disable colour output if FOCUS_NO_COLOR is set
	if _, exists := os.LookupEnv("FOCUS_NO_COLOR"); exists {
		disableStyling()
	}

	pterm.Error.MessageStyle = pterm.NewStyle(pterm.FgRed)
	pterm.Error.Prefix = pterm.Prefix{
		Text:  "ERROR",
		Style: pterm.NewStyle(pterm.BgRed, pterm.FgBlack),
	}
}

// disableStyling disables all styling provided by pterm.
func disableStyling() {
	pterm.DisableColor()
	pterm.DisableStyling()
	pterm.Debug.Prefix.Text = ""
	pterm.Info.Prefix.Text = ""
	pterm.Success.Prefix.Text = ""
	pterm.Warning.Prefix.Text = ""
	pterm.Error.Prefix.Text = ""
	pterm.Fatal.Prefix.Text = ""
}

// checkForUpdates alerts the user if there is
// an updated version of Focus from the one currently installed.
func checkForUpdates(app *cli.App) {
	spinner, _ := pterm.DefaultSpinner.Start("Checking for updates...")
	c := http.Client{Timeout: 10 * time.Second}

	resp, err := c.Get("https://github.com/ayoisaiah/focus/releases/latest")
	if err != nil {
		pterm.Error.Println("HTTP Error: Failed to check for update")
		return
	}

	defer resp.Body.Close()

	var version string

	_, err = fmt.Sscanf(
		resp.Request.URL.String(),
		"https://github.com/ayoisaiah/focus/releases/tag/%s",
		&version,
	)
	if err != nil {
		pterm.Error.Println("Failed to get latest version")
		return
	}

	if version == app.Version {
		text := pterm.Sprintf(
			"Congratulations, you are using the latest version of %s",
			app.Name,
		)
		spinner.Success(text)
	} else {
		pterm.Warning.Prefix = pterm.Prefix{
			Text:  "UPDATE AVAILABLE",
			Style: pterm.NewStyle(pterm.BgYellow, pterm.FgBlack),
		}
		pterm.Warning.Printfln("A new release of Focus is available: %s at %s", version, resp.Request.URL.String())
	}
}

// GetApp retrieves the focus app instance.
func GetApp() *cli.App {
	return &cli.App{
		Name: "Focus",
		Authors: []*cli.Author{
			{
				Name:  "Ayooluwa Isaiah",
				Email: "ayo@freshman.tech",
			},
		},
		Usage:                "Focus is a cross-platform productivity timer for the command-line. It is based on the Pomodoro Technique,\n\t\ta time management method developed by Francesco Cirillo in the late 1980s.",
		UsageText:            "[COMMAND] [OPTIONS]",
		Version:              "v0.1.0",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			{
				Name:  "stats",
				Usage: "Track your progress with detailed statistics reporting. Defaults to a reporting period of 7 days.",
				Action: func(ctx *cli.Context) error {
					if ctx.Bool("no-color") {
						pterm.DisableColor()
					}

					store, err := focus.NewStore()
					if err != nil {
						return err
					}

					stats, err := focus.NewStats(ctx, store)
					if err != nil {
						return err
					}

					if ctx.Bool("delete") {
						return stats.Delete(os.Stdout, os.Stdin)
					}

					if ctx.Bool("list") {
						return stats.List(os.Stdout)
					}

					return stats.Show(os.Stdout)
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "delete",
						Usage: "Delete the all work sessions within the specified time period.",
					},
					&cli.BoolFlag{
						Name:  "list",
						Usage: "List all the work sessions within the specified time period.",
					},
					&cli.StringFlag{
						Name:    "period",
						Aliases: []string{"p"},
						Usage:   "Specify a time period for (defaults to 7days). Possible values are: today, yesterday, 7days, 14days, 30days, 90days, 180days, 365days, all-time.",
						Value:   "7days",
					},
					&cli.StringFlag{
						Name:    "start",
						Aliases: []string{"s"},
						Usage:   "Specify a start date in the following format: YYYY-MM-DD [HH:MM:SS PM].",
					},
					&cli.StringFlag{
						Name:    "end",
						Aliases: []string{"e"},
						Usage:   "Specify an end date in the following format: YYYY-MM-DD [HH:MM:SS PM] (defaults to the current time).",
					},
					&cli.BoolFlag{
						Name:  "no-color",
						Usage: "Disable coloured output.",
					},
				},
			},
		},
		Flags: []cli.Flag{
			&cli.UintFlag{
				Name:    "long-break",
				Usage:   "Long break duration in minutes (default: 15).",
				Aliases: []string{"l"},
			},
			&cli.UintFlag{
				Name:    "short-break",
				Usage:   "Short break duration in minutes (default: 5).",
				Aliases: []string{"s"},
			},
			&cli.UintFlag{
				Name:    "work",
				Usage:   "Work duration in minutes (default: 25).",
				Aliases: []string{"w"},
			},
			&cli.UintFlag{
				Name:    "long-break-interval",
				Aliases: []string{"int"},
				Usage:   "The number of work sessions before a long break (default: 4).",
			},
			&cli.UintFlag{
				Name:    "max-sessions",
				Aliases: []string{"max"},
				Usage:   "The maximum number of work sessions (unlimited by default).",
			},
			&cli.BoolFlag{
				Name:    "disable-notifications",
				Aliases: []string{"d"},
				Usage:   "Disable the system notification after a session is completed.",
			},
			&cli.BoolFlag{
				Name:    "new",
				Aliases: []string{"n"},
				Usage:   "Start a brand new focus session.",
			},
			&cli.BoolFlag{
				Name:  "no-color",
				Usage: "Disable coloured output.",
			},
			&cli.StringFlag{
				Name:  "sound",
				Usage: "Play ambient sounds continuously during a session. Valid options: coffee_shop, fireplace, rain,\n\t\t\t\twind, summer_night, playground.",
			},
			&cli.BoolFlag{
				Name:    "sound-on-break",
				Aliases: []string{"sob"},
				Usage:   "Play sounds during break session.",
			},
		},
		Action: func(ctx *cli.Context) error {
			if ctx.Bool("no-color") {
				disableStyling()
			}

			store, err := focus.NewStore()
			if err != nil {
				return err
			}

			// Running focus without arguments will attempt
			// to resume an interrupted session
			if ctx.NumFlags() == 0 {
				t := &focus.Timer{
					Store: store,
				}

				_, _, err = t.GetInterrupted()
				if err == nil {
					return t.Resume()
				}
			}

			config, err := focus.NewConfig()
			if err != nil {
				return err
			}

			t := focus.NewTimer(ctx, config, store)

			return t.Run()
		},
	}
}
