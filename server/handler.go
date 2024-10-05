package server

import (
	"image/png"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/wayjam/tvbox-mixproxy/config"
	"github.com/wayjam/tvbox-mixproxy/pkg/imageutil"
	"github.com/wayjam/tvbox-mixproxy/pkg/mixer"
)

func Home(c fiber.Ctx) error {
	return c.SendString("Hello, TVBox MixProxy 👋!")
}

func Logo(c fiber.Ctx) error {
	svgLogo := `<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100" viewBox="0 0 100 100">
		<rect x="10" y="20" width="80" height="60" rx="5" ry="5" fill="#333"/>
		<rect x="15" y="25" width="70" height="50" rx="3" ry="3" fill="#4CAF50"/>
		<circle cx="50" cy="75" r="5" fill="#333"/>
		<line x1="30" y1="15" x2="40" y2="5" stroke="#333" stroke-width="2"/>
		<line x1="70" y1="15" x2="60" y2="5" stroke="#333" stroke-width="2"/>
	</svg>`
	c.Set("Content-Type", "image/svg+xml")
	return c.SendString(svgLogo)
}

func Wallpaper(c fiber.Ctx) error {
	// Parse query parameters
	width, _ := strconv.Atoi(c.Query("width", "800"))
	height, _ := strconv.Atoi(c.Query("height", "600"))
	pattern := c.Query("pattern", "solid")
	opacity, _ := strconv.ParseFloat(c.Query("opacity", "1.0"), 64)
	borderWidth, _ := strconv.Atoi(c.Query("border_width", "0"))

	// Parse colors
	bgColor := imageutil.ParseColor(c.Query("bg_color", "FFFFFF"))
	borderColor := imageutil.ParseColor(c.Query("border_color", "000000"))

	// Create image parameters
	params := imageutil.ImageParams{
		BackgroundColor: bgColor,
		Width:           width,
		Height:          height,
		Pattern:         pattern,
		Opacity:         opacity,
		BorderWidth:     borderWidth,
		BorderColor:     borderColor,
	}

	// Generate the image
	img := imageutil.GenerateImage(params)

	// Encode the image to PNG
	c.Set("Content-Type", "image/png")
	return png.Encode(c.Response().BodyWriter(), img)
}

func NewRepoHandler(cfg *config.Config, sourceManager *mixer.SourceManager) fiber.Handler {
	return func(c fiber.Ctx) error {
		if cfg.SingleRepoOpt.Disable {
			return c.Status(fiber.StatusNotImplemented).SendString("SingleRepo is disabled")
		}

		result, err := mixer.MixRepo(cfg, sourceManager)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		return c.JSON(result)
	}
}

func NewMultiRepoHandler(cfg *config.Config, sourceManager *mixer.SourceManager) fiber.Handler {
	return func(c fiber.Ctx) error {
		if cfg.MultiRepoOpt.Disable {
			return c.Status(fiber.StatusNotImplemented).SendString("MultiRepo is disabled")
		}

		result, err := mixer.MixMultiRepo(cfg, sourceManager)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		return c.JSON(result)
	}
}

func NewSpiderHandler(cfg *config.Config, sourceManager *mixer.SourceManager) fiber.Handler {
	handler, err := mixer.NewMixURLHandler(cfg.SingleRepoOpt.Spider, sourceManager)
	if err != nil {
		return func(c fiber.Ctx) error {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}
	}

	return handler
}
