package api

import (
	"net/http"
	"os"

	"github.com/labstack/echo/v4"

	"menguz/internal/middleware"
)

func (s *Server) routes() {
	e := s.Router

	// ---------- health ----------
	e.GET("/healthz", s.health)

	// ---------- public storefront API ----------
	e.GET("/data/products.json", s.dataProductsJSON)
	e.GET("/api/products", s.listProducts)
	e.GET("/api/products/:cve", s.getProduct)
	e.GET("/api/categories", s.listCategories)
	e.GET("/api/promos", s.activePromos)

	// cart & checkout (bank-transfer reference, per proposal PDF)
	e.GET("/api/cart", s.getCart)
	e.POST("/api/cart/items", s.cartAdd)
	e.DELETE("/api/cart/items/:cve", s.cartRemove)
	e.POST("/api/checkout", s.checkout)
	e.GET("/api/orders/lookup", s.orderLookup)

	// chatbot (SSE)
	e.POST("/api/chat", s.chatHandler)

	// webhooks
	e.GET("/webhooks/whatsapp", s.waVerify)
	e.POST("/webhooks/whatsapp", s.waReceive)

	// auth
	e.POST("/api/auth/login", s.adminLogin)
	e.POST("/api/auth/refresh", s.refreshHandler)
	e.POST("/api/auth/logout", s.logout)
	e.POST("/api/b2b/login", s.b2bLogin)

	// ---------- admin API (JWT) ----------
	admin := e.Group("/api/admin", middleware.RequireAuth(s.Cfg.JWTSecret, "admin", "ventas"))
	admin.GET("/orders", s.adminListOrders)
	admin.GET("/orders/:id", s.adminGetOrder)
	admin.PATCH("/orders/:id/status", s.adminSetOrderStatus)
	admin.GET("/campaigns", s.adminListCampaigns)
	admin.POST("/campaigns", s.adminCreateCampaign)
	admin.GET("/campaigns/:id", s.adminGetCampaign)
	admin.PUT("/campaigns/:id", s.adminUpdateCampaign)
	admin.PATCH("/campaigns/:id/toggle", s.adminToggleCampaign)
	admin.DELETE("/campaigns/:id", s.adminDeleteCampaign)
	admin.GET("/clients", s.adminListClients)
	admin.POST("/clients/:id/b2b", s.adminSetB2B)
	admin.POST("/sync", s.adminTriggerSync)
	admin.GET("/sync/logs", s.adminSyncLogs)
	admin.GET("/chat/sessions", s.adminListChatSessions)
	admin.GET("/chat/sessions/:id", s.adminGetChatSession)
	admin.POST("/chat/sessions/:id/resolve", s.adminResolveChatSession)

	// CRM
	admin.GET("/crm/stages", s.crmPipelineStages)
	admin.GET("/crm/deals", s.crmListDeals)
	admin.POST("/crm/deals", s.crmCreateDeal)
	admin.PUT("/crm/deals/:id", s.crmUpdateDeal)
	admin.DELETE("/crm/deals/:id", s.crmDeleteDeal)
	admin.GET("/crm/customers", s.crmListCustomers)
	admin.GET("/crm/customers/:id", s.crmCustomer360)
	admin.GET("/crm/segments", s.crmListSegments)
	admin.GET("/crm/segments/:segmento/members", s.crmSegmentMembers)
	admin.GET("/crm/export/:kind", s.crmExport)

	// ---------- B2B portal API (JWT, rol=b2b) ----------
	b2b := e.Group("/api/b2b", middleware.RequireAuth(s.Cfg.JWTSecret, "b2b"))
	b2b.GET("/me", s.b2bMe)
	b2b.GET("/catalog", s.b2bCatalog)
	b2b.GET("/orders", s.b2bOrders)
	b2b.POST("/orders", s.b2bCreateOrder)

	// ---------- admin panel (HTML, cookie session) ----------
	web := e.Group("/admin", middleware.CSRF())
	web.GET("", s.adminHome)
	web.GET("/login", s.adminLoginPage)
	web.POST("/login", s.adminLoginSubmit)
	web.POST("/logout", s.adminLogoutSubmit)
	web.GET("/campanas", s.adminCampaignsPage)
	web.GET("/campanas/nueva", s.adminCampaignForm)
	web.GET("/campanas/:id", s.adminCampaignForm)
	web.POST("/campanas/guardar", s.adminCampaignSave)
	web.POST("/campanas/:id/toggle", s.adminCampaignToggle)
	web.POST("/campanas/:id/eliminar", s.adminCampaignDelete)
	web.GET("/pedidos", s.adminOrdersPage)
	web.GET("/pedidos/:id", s.adminOrderDetail)
	web.POST("/pedidos/:id/estado", s.adminOrderStatus)
	web.GET("/productos", s.adminProductsPage)
	web.GET("/clientes", s.adminClientsPage)
	web.POST("/clientes/:id/b2b", s.adminClientB2B)
	web.GET("/crm", s.adminCRMPage)
	web.GET("/crm/clientes/:id", s.adminCustomer360)
	web.POST("/crm/deals/guardar", s.adminDealSave)
	web.POST("/crm/deals/:id/eliminar", s.adminDealDelete)
	web.GET("/chat", s.adminChatPage)
	web.POST("/sync", s.adminSyncNow)
	web.GET("/sync-logs", s.adminSyncLogsPage)

	// ---------- static storefront (React build) ----------
	e.GET("/admin/", func(c echo.Context) error { return c.Redirect(http.StatusMovedPermanently, "/admin") })
	e.GET("/*", s.spaStatic)
}

// spaStatic serves the React storefront build if present: real files (JS/CSS
// bundles, images) are served as-is; everything else falls back to
// index.html for client-side routing.
func (s *Server) spaStatic(c echo.Context) error {
	path := c.Request().URL.Path
	if path != "/" && (len(path) >= 4 && path[:4] == "/api" || len(path) >= 5 && path[:5] == "/data" || len(path) >= 5 && path[:5] == "/admin") {
		return echo.NewHTTPError(http.StatusNotFound, "ruta no encontrada")
	}
	if path == "/" {
		return c.File(s.Cfg.StaticDir + "/index.html")
	}
	full := s.Cfg.StaticDir + path
	if info, err := os.Stat(full); err == nil && !info.IsDir() {
		return c.File(full)
	}
	return c.File(s.Cfg.StaticDir + "/index.html")
}
