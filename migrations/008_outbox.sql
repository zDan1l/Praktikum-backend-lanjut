-- Migration 008: transactional outbox dan catatan notifikasi (Modul 11).
-- psql -U postgres -d praktikum_backend -f migrations/008_outbox.sql

-- ---------------------------------------------------------------------
-- outbox — "kotak surat keluar" milik database.
--
-- Setiap event penting (mendaftar, keluar, pindah kelas) dicatat di sini
-- DI DALAM transaction yang sama dengan perubahan datanya. Karena satu
-- transaction, hanya ada dua kemungkinan: perubahan data dan catatan
-- event-nya sama-sama tersimpan, atau sama-sama batal.
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS outbox (
    id           BIGSERIAL PRIMARY KEY,
    event_type   VARCHAR(50) NOT NULL,
    payload      JSONB NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- NULL berarti belum di-publish ke queue.
    published_at TIMESTAMPTZ NULL
);

-- Relay hanya pernah mencari baris yang belum di-publish. Partial index
-- membuatnya tetap kecil walaupun tabel outbox tumbuh jutaan baris.
CREATE INDEX IF NOT EXISTS outbox_unpublished_idx
    ON outbox (id) WHERE published_at IS NULL;

-- ---------------------------------------------------------------------
-- notifications — bukti bahwa sebuah event SUDAH dikirimkan.
--
-- UNIQUE (outbox_id) adalah idempotency key: satu event hanya boleh
-- menghasilkan satu notifikasi, berapa kali pun task-nya dijalankan.
--
-- outbox_id sengaja TIDAK diberi FOREIGN KEY ke outbox: baris outbox yang
-- sudah lama di-publish suatu saat perlu dibersihkan, sedangkan riwayat
-- notifikasi milik mahasiswa harus tetap ada.
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS notifications (
    id        BIGSERIAL PRIMARY KEY,
    outbox_id BIGINT  NOT NULL,
    user_id   INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel   VARCHAR(20)  NOT NULL,
    subject   VARCHAR(200) NOT NULL,
    body      TEXT NOT NULL,
    sent_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT notifications_outbox_unique UNIQUE (outbox_id)
);

CREATE INDEX IF NOT EXISTS notifications_user_idx
    ON notifications (user_id, outbox_id DESC);

-- Permission monitoring queue. Repo ini memodelkan permission sebagai
-- (action, resource) — bukan satu nama "queue:monitor" seperti pada teks
-- modul — jadi di sini dipetakan menjadi action='monitor', resource='queue'.
INSERT INTO permissions (action, resource) VALUES ('monitor', 'queue')
ON CONFLICT (action, resource) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.action = 'monitor' AND p.resource = 'queue'
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;
