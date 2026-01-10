#!/usr/bin/env bash
set -e

BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "============================="
echo "      N g i n x l i n g s    "
echo "============================="
echo
echo "Egzersizler:"
echo "  1) 01_static_server"
echo "  q) Çıkış"
echo

read -rp "Seçiminiz: " choice

case "$choice" in
  1)
    EX_DIR="$BASE_DIR/exercises/01_static_server"
    cd "$EX_DIR"
    echo
    echo "[INFO] Egzersiz klasörü: $EX_DIR"
    echo "----------------------------------------"
    if [ -f README.md ]; then
      sed -n '1,80p' README.md || true
      echo "----------------------------------------"
    else
      echo "README.md bulunamadı."
      echo "----------------------------------------"
    fi
    echo
    echo "Çözümünü test etmek için:"
    echo "  ./check.sh"
    echo
    ;;
  q|Q)
    echo "Görüşürüz 👋"
    exit 0
    ;;
  *)
    echo "Geçersiz seçim."
    exit 1
    ;;
esac
