# 16 – Shared Memory Zone Sizing & Limits

## Amaç

Bu egzersizde `zone=<ad>:<boyut>` ifadesindeki `<boyut>`'un gerçek bir limit
olduğunu görecek, bir `limit_req` zone'unda yer kalmadığında NGINX'in ne yaptığını
öğrenecek ve zone'u çok sayıda farklı anahtar altında çalışmaya devam edecek
boyuta getireceksin.

---

## Arka plan – zone'u ne doldurur

- Bir `limit_req_zone` **her farklı anahtar için bir durum** tutar.
  `$binary_remote_addr$arg_id` anahtarıyla bir istemcinin her farklı `?id=` değeri
  yeni bir durumdur. (`id` kısmı yalnızca bu lab'ın tek bir makineden çok sayıda
  anahtar üretebilmesi için var.)
- Zone dolduğunda NGINX isteği başarısız saymaz. **En uzun süredir kullanılmayan
  durumu siler** ve yenisini saklar. İstek ancak bu da yer açmazsa bir hatayla
  reddedilir. Bu yüzden fazla küçük bir zone hiçbir hata göstermez; hâlâ
  sınırlaması gereken istemcileri sessizce "unutur".
- NGINX belgesi boyutlandırma kuralını verir: "One megabyte zone can keep about
  16 thousand 64-byte states or about 8 thousand 128-byte states." Bir durum 32
  bit platformlarda 64, 64 bit platformlarda 128 bayt yer kaplar. Pay bırak: zone
  bir miktar yönetim verisi de tutar.
- NGINX çok küçük bir zone'u `zone "..." is too small` ile reddeder.

Location, `return` kullanmak yerine bir dosya sunar: `return` rewrite fazında
çalışır ve `limit_req` preaccess fazında devreye giremeden yanıt döner; yani
limiti atlatır.

Kaynak: [limit_req_zone](https://nginx.org/en/docs/http/ngx_http_limit_req_module.html#limit_req_zone).

---

## Lab ortamı

- `Dockerfile`, `nginx:1.28.3-alpine` üzerine kurulur ve
  `/usr/share/nginx/html/index.html` dosyasına `zone-sizing-ok` yazar.
- `nginx.conf`, `req_zone`'u `64k` ve `rate=1r/m` ile tanımlar, reddedilen
  isteklere 429 döner ve zone'u dosyayı sunan `location /` içinde uygular. Yavaş
  rate, sınırlanmış bir istemciyi bir dakika boyunca sınırlı tutar; check'in
  zone'un onu hâlâ hatırlayıp hatırlamadığını görmesine yetecek kadar.

Check (Nginxlings'te `c` veya Enter ya da `./check.sh`) image'i build eder, etkin
yapılandırmayı `nginx -T` ile okur ve container'ı başlatır (host portu 18160).
Ardından:

1. `?id=limited`'i iki kez ister: ilki 200, ikincisi 429 almalı;
2. tek bir keep-alive bağlantı üzerinden, farklı `?id=` değerleriyle 1500 istek
   gönderir;
3. `?id=limited`'i yeniden ister: hâlâ 429 almalı;
4. zone'un en az `256k` olduğunu doğrular.

---

## Görev

`nginx.conf` içinde `req_zone`'u, sınırlı istemciyi 3. adıma kadar hatırlayacak
kadar büyüt. Zone adını (`req_zone`), anahtarını, `rate=1r/m` değerini,
`location /` içindeki `limit_req`'i ve dosya sunumunu olduğu gibi bırak. Boyut en
az `256k` olmalı.

---

## Başarı kriterleri

- `nginx -t` yapılandırmayı kabul etmeli.
- Sınırlı istemci ikinci isteğinde 429 almalı.
- 1500 farklı anahtardan sonra da 429 almaya devam etmeli.
- `req_zone` en az `256k` olmalı.

---

## Takıldıysan

Bunları sırayla dene, aralarda check'i yeniden çalıştır:

- `req_zone:64k forgot the limited client`: zone doldu ve NGINX en eski durumu,
  yani sınırlı istemcinin durumunu sildi. 2. adım kaç durum oluşturuyor ve
  zone'una kaç tane sığıyor?
- `held 1500 keys, but the exercise asks for at least 256k`: boyutun ancak
  yetiyor, pay bırakmıyor. Şart 256k ya da daha fazlası.
- `NGINX rejected the configuration: zone "zone=req_zone:..." is too small`: boyutu
  yanlış yöne değiştirdin.
- `Expected HTTP 429 for the second request`: `limit_req` çalışmadı. Hâlâ
  `location /` içinde mi ve location `return` yerine dosyayı mı sunuyor?
- `Keep rate=1r/m on req_zone`: check bu rate'e dayanıyor.

---

## Pekiştirme soruları

- 64 bit bir sistemde 256k'lık bir zone'a yaklaşık kaç durum sığar ve 1500 anahtar
  için en küçük boyutu nasıl tahmin edersin?
- Bir `limit_req` zone'u dolduğunda NGINX önce ne yapar ve istek ne zaman bunun
  yerine bir hatayla sonuçlanır?
- Bu lab'ın anahtarı istemcinin seçtiği `$arg_id`'yi içeriyor. Gerçek bir istemci
  böyle bir anahtarla ne yapabilir ve istemci başına sınırlama için hangi anahtarı
  kullanırdın?
