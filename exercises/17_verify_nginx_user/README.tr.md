# 17 – Verify NGINX User & Processes

## Amaç

Bu egzersizde NGINX’in process modelini ve `user` directive’inin neyi etkilediğini öğreneceksin.

Odak noktaları:

- `user` directive’in worker process kullanıcı hesabını belirlemesi
- Master process ile worker process farkı
- Çalışan process listesinden (master/worker) doğru çıkarım yapabilmek

## Görev

`nginx.conf` içinde:

- Main context’e `user nginx;` ekle

Bu değişiklikle:

- Master process genelde root olarak başlar
- Worker process’ler `nginx` kullanıcısı ile çalışır

## Başarı kriterleri

- Konfigürasyon geçerli olmalı
- Effective config içinde `user nginx;` bulunmalı
- Process listesinde:
  - `nginx: master process` (root)
  - `nginx: worker process` (nginx)
  görülmeli