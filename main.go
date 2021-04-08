package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"time"

	"git.swab.dev/breto/blocks"
	"git.swab.dev/breto/icons"
	"git.swab.dev/breto/stats"
	"git.swab.dev/breto/ui"
)

// CLI flag variables
var dwm, battery, clock, audio, memory, diskSpace, temperature, tray, emoji bool

func init() {
	// Setup and define cli flags
	flag.BoolVar(&dwm, "dwm", false, "Used to enable output for DWM's status bar.\n Example: --dwm=true")
	flag.BoolVar(&battery, "battery", false, "Used to enable battery module.\n Example: --battery=true")
	flag.BoolVar(&clock, "dateTime", true, "Used to disable the date and time module.\n Example: --dateTime=false")
	flag.BoolVar(&audio, "volume", true, "Used to disable the volume module.\n Example: --volume=false")
	flag.BoolVar(&memory, "ram", true, "Used to disable the RAM module.\n Example: --ram=false")
	flag.BoolVar(&diskSpace, "storage", true, "Used to disable the home directory storage module.\n Example: --storage=false")
	flag.BoolVar(&temperature, "temp", true, "Used to disable the temperature module.\n Example: --temp=false")
	flag.BoolVar(&tray, "tray", true, "Used to disable the custom tray module.\n Example: --tray=false")
	flag.BoolVar(&emoji, "emoji", false, "Used to enable Openmoji icons instead of Awesome Font.\n Example: --emoji=true")
}

func formatOutput(status string, stats *stats.Info, ico *icons.Symbols, baty batInfo) string {
	if temperature {
		status = fmt.Sprintf("%s %s%s ", status, icons.Temp(emoji), stats.weather)
	}
	if diskSpace {
		status = fmt.Sprintf("%s %s%s ", status, icons.Dir(emoji), stats.homeSpace)
	}
	if memory {
		status = fmt.Sprintf("%s %s%s ", status, icons.Mem(emoji), stats.ramFree)
	}
	if audio {
		stats.volText, _ = blocks.VolumeText()
		ico.volIcon, _ = icons.Volume(emoji)
		status = fmt.Sprintf("%s %s%s ", status, ico.volIcon, stats.volText)
	}
	if battery {
		if baty.fiveMins == 0 || baty.passed < 10 {
			stats.power, _ = blocks.Battery()
		}
		status = fmt.Sprintf("%s %s%s ", status, icons.Power(emoji), stats.power)
	}
	if clock {
		status = fmt.Sprintf("%s %s ", status, stats.hTime)
	}
	if tray {
		ico.rShift, _ = icons.Redshift(emoji)
		ico.dropbox, _ = icons.Dropbox(emoji)
		ico.syncthing, _ = icons.Syncthing(emoji)
		status = fmt.Sprintf("%s %s%s%s", status, ico.dropbox, ico.syncthing, ico.rShift)
	}
	return status
}

type batInfo struct {
	passed   float64
	fiveMins float64
}

func main() {
	flag.Parse()

	var (
		stats     = new(stats.Info)
		ico       = new(icons.Symbols)
		baty      = batInfo{}
		cWttr     = make(chan string)
		eWttr     = make(chan error)
		cRAM      = make(chan string)
		eRAM      = make(chan error)
		cHomeDisk = make(chan string)
		eHomeDisk = make(chan error)
	)

	// Each Go routine has it's own timer to delay the execution of the command.
	// A Go routine will run unless it's CLI flag is set to false.
	if temperature {
		go blocks.Wttr(cWttr, eWttr)
	}

	if memory {
		go blocks.FreeRam(cRAM, eRAM)
	}

	if diskSpace {
		go blocks.HomeDisk(cHomeDisk, eHomeDisk)
	}

	start := time.Now() // for batter time math
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		// add year & seconds with "Jan 02, 2006 15:04:05"
		stats.hTime = time.Now().Format("Jan 02 15:04")

		if battery {
			baty.passed = time.Since(start).Seconds()
			baty.fiveMins = math.Floor(math.Remainder(baty.passed, 300))
		}

		select { // updates the go routine channels as they send data
		case stats.weather = <-cWttr:
		case stats.wttrErr = <-eWttr:
			log.Println(stats.wttrErr.Error())
		case stats.ramFree = <-cRAM:
		case stats.ramErr = <-eRAM:
			log.Println(stats.ramErr.Error())
		case stats.homeSpace = <-cHomeDisk:
		case stats.homeErr = <-eHomeDisk:
			log.Println(stats.homeErr.Error())
		default:
		}

		// Status bar information as defined by the CLI flags.
		// reset status on every run.
		finalStatus := formatOutput("", stats, ico, baty)

		// Output methods as specified by CLI flags.
		if dwm {
			ui.Dwm(finalStatus)
		} else {
			ui.Default(finalStatus)
		}
	}
}
