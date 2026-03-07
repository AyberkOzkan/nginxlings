# 16 – Shared Memory Zone Sizing & Limits

## Amaç

Bu egzersizde, shared memory zone boyutunu belirleyen şeyin doğrudan config olduğunu öğreneceksin.

Odak noktaları:

- `zone=<name>:<size>` ifadesindeki `<size>` değerinin gerçek bir limit olması
- Küçük bir zone ile sistemin zorlanmasını görmek
- Zone büyütüldüğünde aynı senaryonun stabil hale gelmesi

## Görev

`nginx.conf` içinde `limit_req_zone` satırında:

- Zone adını `req_zone` olarak koru
- Zone boyutunu büyüt (ör: 256k, 512k, 1m)

Zone boyutu büyüdüğünde:

- Çok sayıda farklı anahtar (key) tutulabilir
- Shared memory allocation hataları görülmez
- Rate limiting yine çalışır

## Başarı kriterleri

- Zone boyutu en az 256k olmalı
- Yapılandırma geçerli olmalı
- Rate limiting çalışıyor olmalı (bazı istekler 429 dönebilir)