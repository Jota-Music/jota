package main

import (
	"embed"
	"log"
	"os"

	"jota/server/internal/app"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
)

//go:embed all:client/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	a := app.New()

	if os.Getenv("WEBKIT_DISABLE_DMABUF_RENDERER") == "" {
		_ = os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")
	}

	err := wails.Run(&options.App{
		Title:            "Jota " + version,
		Width:            1100,
		Height:           720,
		MinWidth:         640,
		MinHeight:        480,
		Frameless:        true,
		CSSDragProperty:  "--wails-draggable",
		CSSDragValue:     "drag",
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 12, G: 10, B: 9, A: 1},
		OnStartup:        a.Startup,
		OnShutdown:       a.Shutdown,
		Bind: []any{
			a,
		},
		Linux: &linux.Options{
			ProgramName:         "jota",
			Icon:                icon,
			WindowIsTranslucent: true,
			WebviewGpuPolicy:    linux.WebviewGpuPolicyOnDemand,
		},
	})

	if err != nil {
		log.Fatal(err)
	}
}
