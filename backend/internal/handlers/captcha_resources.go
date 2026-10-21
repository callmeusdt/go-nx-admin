package handlers

import (
	"fmt"
	"sync"

	"github.com/wenlng/go-captcha-assets/resources/imagesv2"
	"github.com/wenlng/go-captcha-assets/resources/tiles"
	"github.com/wenlng/go-captcha/v2/base/option"
	"github.com/wenlng/go-captcha/v2/slide"
)

var (
	slideOnce sync.Once
	slideCapt slide.Captcha
	slideErr  error
)

// slideCaptcha builds the slider captcha on first use. Loading the embedded
// images at package init made any import of this package able to terminate the
// process; a failure is now reported to the one request that needed it.
func slideCaptcha() (slide.Captcha, error) {
	slideOnce.Do(func() {
		builder := slide.NewBuilder(
			slide.WithImageSize(option.Size{Width: 300, Height: 220}),
		)

		bgs, err := imagesv2.GetImages()
		if err != nil {
			slideErr = fmt.Errorf("captcha background images load failed: %w", err)
			return
		}

		graphs, err := tiles.GetTiles()
		if err != nil {
			slideErr = fmt.Errorf("captcha tile images load failed: %w", err)
			return
		}
		var slideGraphs []*slide.GraphImage
		for _, g := range graphs {
			slideGraphs = append(slideGraphs, &slide.GraphImage{
				OverlayImage: g.OverlayImage,
				ShadowImage:  g.ShadowImage,
				MaskImage:    g.MaskImage,
			})
		}

		builder.SetResources(
			slide.WithBackgrounds(bgs),
			slide.WithGraphImages(slideGraphs),
		)
		slideCapt = builder.Make()
	})
	return slideCapt, slideErr
}
