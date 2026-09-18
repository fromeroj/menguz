// menguz-server — single-binary platform for Menguz Vinos & Licores.
//
// Month 1: B2C storefront API, nightly SAE 8.0 sync, campaign editor, bank-reference checkout, admin panel.
// Month 2: AI chatbot (OpenAI + RAG), WhatsApp Business webhook, B2B portal with SAE price lists.
// Month 3: CRM base — pipeline, customer 360, segmentation, structured export.
//
// Usage:
//
//	menguz-server --port=8080 --badger-path=./data/badger --sqlite-path=./data/analytics.db \
//	  --sae-host=192.168.1.100 --sae-db=/ruta/Empresa01.fdb
package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"menguz/internal/api"
	"menguz/internal/badger"
	"menguz/internal/config"
	"menguz/internal/models"
	"menguz/internal/sae"
	"menguz/internal/sommelier"
	"menguz/internal/sqlite"
)

func main() {
	cfg := config.Load()

	st, err := badger.Open(cfg.BadgerPath)
	if err != nil {
		log.Fatalf("opening badger: %v", err)
	}
	defer st.Close()

	ana, err := sqlite.Open(cfg.SQLitePath)
	if err != nil {
		log.Fatalf("opening sqlite: %v", err)
	}
	defer ana.Close()

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		log.Fatalf("creating data dir: %v", err)
	}

	srv := api.NewServer(cfg, st, ana)

	// bootstrap admin user (logs credentials to stderr on first run)
	if err := srv.EnsureAdmin(); err != nil {
		log.Fatalf("bootstrapping admin: %v", err)
	}

	// seed demo campaigns on first boot (so the storefront shows promos)
	seedCampaigns(srv)

	// seed draft wine profiles on first boot (sommelier enrichment: tipo/uvas
	// detected from the catalog; admin curation happens in /admin/productos)
	if n := countPerfiles(st); n == 0 {
		created, _, err := sommelier.BootstrapDrafts(st)
		if err != nil {
			log.Printf("perfiles: bootstrap: %v", err)
		} else {
			log.Printf("perfiles: %d fichas borrador generadas", created)
		}
	}

	// daily sync: run at boot when stale, then every day at cfg.SyncAt
	syncNow := func(origen string) {
		log.Printf("sync %s: starting", origen)
		slog, err := sae.Sync(cfg, st, ana)
		if err != nil {
			log.Printf("sync %s: ERROR %v", origen, err)
			return
		}
		log.Printf("sync %s: done %d productos, %d clientes, %d facturas (%s)",
			origen, slog.Productos, slog.Clientes, slog.Facturas, slog.Fin.Sub(slog.Inicio).Round(time.Second))
	}
	if cfg.SyncOnBoot {
		last, _ := st.GetString(badger.PrefMeta + badger.MetaLastSync)
		stale := true
		if last != "" {
			if t, err := time.Parse(time.RFC3339, last); err == nil {
				stale = time.Since(t) > 24*time.Hour
			}
		}
		if stale {
			go syncNow("boot")
		}
	}
	go dailyScheduler(cfg.SyncAt, func() { syncNow("cron") })

	// graceful shutdown
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Println("shutting down…")
		if err := srv.ShutdownGracefully(); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}()

	addr := ":" + strconv.Itoa(cfg.Port)
	log.Printf("menguz-server %s listening on %s (sae-mock=%v)", "0.3.0", addr, cfg.SAEMock)
	if cfg.SAEMock {
		log.Printf("SAE mock mode: no Firebird connection; demo catalog active")
	}
	if err := srv.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server: %v", err)
	}
}

// dailyScheduler fires fn once per day at "HH:MM" local time.
func dailyScheduler(at string, fn func()) {
	parts := strings.Split(at, ":")
	if len(parts) != 2 {
		log.Printf("scheduler: invalid sync-at %q, using 03:00", at)
		at = "03:00"
		parts = []string{"03", "00"}
	}
	hh, err1 := strconv.Atoi(parts[0])
	mm, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || hh < 0 || hh > 23 || mm < 0 || mm > 59 {
		log.Printf("scheduler: invalid sync-at %q, using 03:00", at)
		hh, mm = 3, 0
	}
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), hh, mm, 0, 0, now.Location())
		if !next.After(now) {
			next = next.Add(24 * time.Hour)
		}
		time.Sleep(time.Until(next))
		fn()
	}
}

// countPerfiles returns how many wine profiles exist.
func countPerfiles(st *badger.Store) int {
	n := 0
	_ = st.ScanPrefix(badger.PrefPerfilVino, &struct{}{}, func(k string) bool { n++; return false })
	return n
}

// seedCampaigns installs three demo campaigns on the very first boot so the
// storefront promos sections render before the team creates real ones.
func seedCampaigns(srv *api.Server) {
	st := srv.Store
	n := 0
	_ = st.ScanPrefix(badger.PrefCampana, &struct{}{}, func(k string) bool { n++; return false })
	if n > 0 {
		return
	}
	now := time.Now()
	demo := []struct {
		titulo, tipo, cve, desc string
		precio                  float64
	}{
		{"Tequila Añejo Don Julio 750ml", "producto_mes", "TEQ-ANE-001",
			"Nuestro añejo insignia: caramelo, roble y agave asado. Ideal para regalar o cerrar la cena.", 1099.00},
		{"Combo Verano (Absolut + Malibú)", "carousel", "",
			"El clásico del verano: vodka premium + ron de coco a precio especial.", 0},
		{"Mezcal Tobalá Joven 750ml", "carousel", "MEZ-JOV-002",
			"Edición pequeña de agave silvestre. Mineral, tropical, ahumado elegante.", 0},
	}
	for i, d := range demo {
		camp := models.Campana{
			ID: fmt.Sprintf("camp_seed%d", i+1), Titulo: d.titulo, Tipo: d.tipo,
			Descripcion: d.desc, CveArt: d.cve, PrecioOferta: d.precio,
			CtaTexto: "Cotiza ahora", Inicio: now.AddDate(0, 0, -7), Fin: now.AddDate(0, 2, 0),
			Activa: true, Orden: i * 10, CreatedAt: now, UpdatedAt: now,
		}
		_ = st.PutJSON(badger.PrefCampana+camp.ID, camp)
	}
	fmt.Fprintln(os.Stderr, `{"event":"campaigns_seeded","count":3}`)
}
