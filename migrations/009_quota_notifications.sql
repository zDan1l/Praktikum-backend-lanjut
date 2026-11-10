-- Migration 009: mengganti idempotency key notifikasi (Tugas Mandiri D.2).
-- psql -U postgres -d praktikum_backend -f migrations/009_quota_notifications.sql
--
-- Alasan: event enrollment menghasilkan SATU notifikasi (satu mahasiswa), jadi
-- UNIQUE(outbox_id) cukup. Tetapi event course.quota_changed menghasilkan
-- BANYAK notifikasi — satu untuk setiap peserta kelas — sehingga satu outbox_id
-- muncul di banyak baris notifications. UNIQUE(outbox_id) akan menolak peserta
-- kedua dan seterusnya.
--
-- Idempotency key yang benar untuk kedua kasus adalah pasangan
-- (outbox_id, user_id): "satu mahasiswa hanya menerima satu notifikasi untuk
-- satu event, berapa kali pun task-nya dijalankan". Untuk event enrollment
-- (satu user per event) perilakunya tidak berubah; untuk event quota, setiap
-- peserta tetap dijamin menerima tepat satu email meski task di-retry.

ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notifications_outbox_unique;

ALTER TABLE notifications
    ADD CONSTRAINT notifications_outbox_user_unique UNIQUE (outbox_id, user_id);
