#!/usr/bin/env bash
# Menjalankan TestEnrollEndpoint_ConcurrentCreditLimit pada N proses BARU
# (masing-masing -count=3) dan meringkas hasil setiap pengulangan.
#
# Pemakaian (dari folder proyek, Git Bash):
#   bash tests/integration/uji_konkurensi.sh        # 10 proses
#   bash tests/integration/uji_konkurensi.sh 15     # 15 proses
#
# Setiap proses baru = pool koneksi baru (dingin), sehingga pengulangan ke-1
# memperlihatkan efek pool dingin (Kesalahan No. 9).
N=${1:-10}

echo "MaxConns   : $(grep -oE 'cfg.MaxConns = [0-9]+' tests/integration/main_test.go)"
if grep -qE '^[[:space:]]*if err := warmPool' tests/integration/main_test.go; then
  echo "warmPool   : AKTIF"
else
  echo "warmPool   : DIMATIKAN (dikomentari)"
fi
if grep -q 'FOR UPDATE`, userID' app/repository/course_repository.go; then
  echo "LockStudent: FOR UPDATE ada (kode benar)"
else
  echo "LockStudent: FOR UPDATE DIHAPUS (mutation T6)"
fi
if grep -q 'taken+adding <= MaxCreditsPerSemester' app/service/course_rules.go; then
  echo "course_rules.go: bersih (<=)"
else
  echo "course_rules.go: MASIH TERMUTASI — jalankan: git checkout -- app/service/course_rules.go"
  exit 1
fi
echo

lolos=0
for i in $(seq 1 "$N"); do
  hasil=$(go test -tags=integration -count=3 -run Concurrent -v ./tests/integration/ 2>&1 | grep -E '^--- ' | awk '{printf "%s%s ", $2, $4}')
  printf 'proses %2d: %s\n' "$i" "$hasil"
  case "$hasil" in PASS:*) lolos=$((lolos + 1)) ;; esac
done
echo
echo "pengulangan ke-1 yang PASS: $lolos dari $N proses"
