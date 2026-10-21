package handlers

import (
	"mime"
	"net/http"
	"strings"

	"go-nx-admin/internal/config"
)

// activeExtensions can run script in the page origin when served inline, so
// they are refused even if an operator lists them by mistake.
var activeExtensions = map[string]bool{
	"html": true, "htm": true, "xhtml": true, "svg": true, "svgz": true, "xml": true,
	"js": true, "mjs": true, "css": true, "swf": true, "php": true, "jsp": true,
	"exe": true, "dll": true, "sh": true, "bat": true,
}

func allowedUploadTypes() map[string]bool {
	list := config.DefaultUploadTypes
	if config.AppConfig != nil && len(config.AppConfig.Upload.AllowTypes) > 0 {
		list = config.AppConfig.Upload.AllowTypes
	}
	allowed := make(map[string]bool, len(list))
	for _, ext := range list {
		ext = strings.ToLower(strings.Trim(ext, "."))
		if ext != "" && !activeExtensions[ext] {
			allowed[ext] = true
		}
	}
	return allowed
}

// checkUpload validates the extension against the allow list and the content
// against the extension. The returned MIME type is detected from the bytes,
// never taken from the client. head is the first bytes of the file.
func checkUpload(filename string, head []byte) (ext, mimeType string, ok bool) {
	dot := strings.LastIndex(filename, ".")
	if dot < 0 || dot == len(filename)-1 {
		return "", "", false
	}
	ext = strings.ToLower(filename[dot+1:])
	if !allowedUploadTypes()[ext] {
		return "", "", false
	}
	detected := http.DetectContentType(head)
	base, _, err := mime.ParseMediaType(detected)
	if err != nil {
		return "", "", false
	}
	// Anything sniffed as markup or script is refused whatever its extension.
	switch base {
	case "text/html", "text/xml", "application/xml", "image/svg+xml", "application/javascript", "text/javascript":
		return "", "", false
	}
	// Binary media must look like what the extension claims.
	if want, strict := strictImageMIME[ext]; strict && base != want {
		return "", "", false
	}
	return ext, base, true
}

var strictImageMIME = map[string]string{
	"jpg": "image/jpeg", "jpeg": "image/jpeg", "png": "image/png",
	"gif": "image/gif", "webp": "image/webp", "bmp": "image/bmp",
	"pdf": "application/pdf",
}
