package badger

// Key schema: hierarchical prefixes for range scans.
//
//	producto:{cve_art}              -> Producto JSON
//	cliente:{id}                    -> Cliente JSON
//	factura:{id}                    -> Factura JSON
//	campana:{id}                    -> Campana JSON
//	orden:{id}                      -> Orden JSON
//	orden_idx:cliente:{cid}:{id}    -> 1 (secondary index)
//	cart:{token}                    -> Cart JSON
//	user:{id}                       -> User JSON
//	user_email:{email}              -> user id
//	refresh:{token}                 -> {user_id, exp}
//	deal:{id}                       -> Deal JSON
//	chat_sess:{id}                  -> ChatSession JSON
//	chat_idx:{yyyymm}               -> chat session id list (newline separated)
//	sync_log:{ts}                   -> SyncLog JSON
//	meta:{k}                        -> misc metadata (last_sync, seq:folio...)
const (
	PrefProducto     = "producto:"
	PrefCliente      = "cliente:"
	PrefFactura      = "factura:"
	PrefCampana      = "campana:"
	PrefOrden        = "orden:"
	PrefOrdenCliente = "orden_idx:cliente:"
	PrefCart         = "cart:"
	PrefUser         = "user:"
	PrefUserEmail    = "user_email:"
	PrefRefresh      = "refresh:"
	PrefDeal         = "deal:"
	PrefChatSess     = "chat_sess:"
	PrefChatIdx      = "chat_idx:"
	PrefSyncLog      = "sync_log:"
	PrefMeta         = "meta:"
)

// Meta keys
const (
	MetaLastSync = "last_sync"
	MetaFolioSeq = "seq:folio"
)
