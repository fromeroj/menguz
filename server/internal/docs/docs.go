// Package docs embebe la especificación OpenAPI y los assets de Swagger UI
// dentro del binario, para servir la documentación interactiva en /docs y la
// especificación en /openapi.yaml sin dependencias externas.
//
// Nota: openapi.yaml embebido es una copia de api/openapi.yaml (fuente de
// verdad del repo); regenerar con: cp api/openapi.yaml internal/docs/openapi.yaml
package docs

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/labstack/echo/v4"
)

// swaggerUI contiene los assets estáticos de swagger-ui-dist (v5) con un
// index.html reescrito que apunta a /openapi.yaml.
//
//go:embed all:swagger-ui
var swaggerUI embed.FS

// spec es la especificación OpenAPI 3.0 servida en /openapi.yaml.
//
//go:embed openapi.yaml
var spec []byte

// SpecHandler sirve la especificación OpenAPI embebida.
func SpecHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.Blob(http.StatusOK, "application/yaml", spec)
	}
}

// SwaggerUIHandler sirve la UI en /docs y /docs/ (HTML) y los assets
// estáticos de Swagger UI bajo /docs/* (CSS, JS, favicons).
func SwaggerUIHandler() echo.HandlerFunc {
	sub, err := fs.Sub(swaggerUI, "swagger-ui")
	if err != nil {
		// fs.Sub sobre un embed.FS fijo no puede fallar en la práctica.
		panic(err)
	}
	files := echo.WrapHandler(http.StripPrefix("/docs", http.FileServer(http.FS(sub))))
	return func(c echo.Context) error {
		p := c.Request().URL.Path
		if p == "/docs" || p == "/docs/" {
			return c.HTML(http.StatusOK, uiIndexHTML)
		}
		return files(c)
	}
}

// uiIndexHTML es el documento servido en /docs. Replica al index.html embebido
// pero se devuelve programáticamente para que tanto /docs como /docs/ sirvan
// la UI sin depender del comportamiento de redirección de http.FileServer.
const uiIndexHTML = `<!doctype html>
<html lang="es">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Menguz API · Documentación</title>
  <link rel="icon" type="image/png" href="/docs/favicon-32x32.png" sizes="32x32">
  <link rel="icon" type="image/png" href="/docs/favicon-16x16.png" sizes="16x16">
  <link rel="stylesheet" href="/docs/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="/docs/swagger-ui-bundle.js" charset="UTF-8"></script>
  <script src="/docs/swagger-ui-standalone-preset.js" charset="UTF-8"></script>
  <script>
    window.onload = function () {
      window.ui = SwaggerUIBundle({
        url: "/openapi.yaml",
        dom_id: "#swagger-ui",
        deepLinking: true,
        presets: [
          SwaggerUIBundle.presets.apis,
          SwaggerUIStandalonePreset
        ],
        plugins: [SwaggerUIBundle.plugins.DownloadUrl],
        layout: "StandaloneLayout",
        tryItOutEnabled: true
      });
    };
  </script>
</body>
</html>
`
