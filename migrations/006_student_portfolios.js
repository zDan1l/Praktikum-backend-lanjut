// Tugas mandiri modul 7: index collection portofolio (MongoDB).
// Aplikasi juga membuat index ini otomatis saat startup (EnsureIndexes).
// Jalankan manual bila perlu: mongosh praktikum_backend migrations/006_student_portfolios.js

const database = db.getSiblingDB("praktikum_backend");

// Satu mahasiswa tepat satu dokumen portofolio.
database.student_portfolios.createIndex(
  { student_id: 1 },
  { unique: true, name: "uniq_student_id" }
);

// Mendukung pencarian portofolio berdasarkan teknologi (multikey).
database.student_portfolios.createIndex(
  { "projects.technologies": 1 },
  { name: "projects_technologies_idx" }
);
