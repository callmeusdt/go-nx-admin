package handlers

import (
	"log"

	"github.com/wenlng/go-captcha-assets/resources/imagesv2"
	"github.com/wenlng/go-captcha-assets/resources/tiles"
	"github.com/wenlng/go-captcha/v2/base/option"
	"github.com/wenlng/go-captcha/v2/slide"
)

var slideCapt slide.Captcha

func init() {
	builder := slide.NewBuilder(
		slide.WithImageSize(option.Size{Width: 300, Height: 220}),
	)

	bgs, err := imagesv2.GetImages()
	if err != nil {
		log.Fatalf("captcha background images load failed: %v", err)
	}

	graphs, err := tiles.GetTiles()
	if err != nil {
		log.Fatalf("captcha tile images load failed: %v", err)
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
}
