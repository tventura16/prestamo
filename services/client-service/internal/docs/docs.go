// Package docs expone el contrato OpenAPI 3.1 del servicio y una UI Swagger.
// El spec (openapi.yaml) es la fuente de verdad del contrato; se sirve embebido
// para no depender del filesystem en runtime.
package docs

import (
	_ "embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed openapi.yaml
var openapiSpec []byte

// swaggerUI carga Swagger UI desde CDN y apunta al spec del propio servicio.
// (Requiere acceso a internet en el navegador; el spec en sí es self-hosted.)
const swaggerUI = `<!DOCTYPE html>
<html lang="es">
<head>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1"/>
  <title>API · Documentación</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css"/>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js" crossorigin></script>
  <script>
    window.ui = SwaggerUIBundle({ url: 'openapi.yaml', dom_id: '#swagger-ui', deepLinking: true });
  </script>
</body>
</html>`

// Register monta las rutas públicas de documentación: el spec OpenAPI crudo en
// /openapi.yaml y la UI Swagger en /docs. Debe montarse fuera del grupo
// autenticado (la documentación es pública dentro de la red).
func Register(r gin.IRouter) {
	r.GET("/openapi.yaml", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/yaml; charset=utf-8", openapiSpec)
	})
	r.GET("/docs", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerUI))
	})
}
