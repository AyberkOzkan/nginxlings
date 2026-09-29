# 16 – Shared Memory Zone Sizing & Limits

## Amaç

Bu egzersizde `zone=<name>:<size>` ifadesindeki `<size>` değerinin gerçek bir
limit olduğunu ve bir `limit_req` zone'unda yer kalmadığında NGINX'in ne
yaptığını göreceksin.

Odak noktaları:

- Her farklı anahtar (burada `$binary_remote_addr$arg_id`) zone içinde bir durum (state) kaplar
- Zone dolduğunda NGINX hata vermek yerine en uzun süredir kullanılmayan durumu
  siler; bu yüzden küçük bir zone, hâlâ sınırlaması gereken istemcileri sessizce "unutur"
- Daha büyük bir zone aynı yük altında onları hatırlamaya devam eder

## Görev

`nginx.conf` içinde `limit_req_zone` satırında:

- Zone adını `req_zone` olarak koru
- `rate=1r/m` değerini koru
- Zone boyutunu büyüt (ör: 256k, 512k, 1m)

Check önce `?id=limited` olarak bir istek gönderir, ardından 429 ile
reddedilmesi gereken ikinci isteği yollar. Sonra farklı anahtarlarla 1500 istek
gönderir ve tekrar `?id=limited` olarak sorar. Başlangıçtaki 64k zone ile sınırlı
istemci o ana kadar silinmiştir ve yeniden 200 alır; yeterince büyük bir zone
ile hâlâ 429 alır.

Location, `return` kullanmak yerine bir dosya sunar: `return` rewrite fazında
çalışır ve `limit_req` preaccess fazında devreye giremeden yanıt döner; yani
limiti atlatır.

Boyutlandırma ipucu: NGINX belgelerine göre 1 MB'lık bir zone yaklaşık 16 bin
adet 64 baytlık durum tutar. 64 bit sistemlerde bir `limit_req` durumu 128
baytlık bir yuva kullanır; bu yüzden megabayt başına yaklaşık 8 bin durum
hesapla ve pay bırak.

## Başarı kriterleri

- Yapılandırma geçerli olmalı
- Sınırlı istemci ikinci isteğinde 429 almalı
- 1500 farklı anahtardan sonra da 429 almaya devam etmeli
- Zone boyutu en az 256k olmalı
