# 14 – Multiple Domains, Static Sites

## Amaç

Bu egzersizde, NGINX ile aynı sunucu üzerinde birden fazla domain için statik içerik sunmayı öğreneceksin.

Bu çalışmada odak noktası şunlardır:

- Birden fazla `server` bloğu kullanmak
- `server_name` ile domain bazlı eşleştirme yapmak
- Her domain için farklı bir `root` dizini tanımlamak
- NGINX’i statik web sunucusu olarak kullanmak

---

## Arkaplan

NGINX, aynı port üzerinde gelen istekleri `Host` başlığına göre farklı sanal host’lara yönlendirebilir.

Bu mantık genelde şu yapı ile kurulur:

    http {
        server {
            listen 8080;
            server_name site1.local;
            root /bir/dizin;
        }

        server {
            listen 8080;
            server_name site2.local;
            root /baska/bir/dizin;
        }
    }

Buradaki temel fikir şudur:

- İstek `site1.local` için geldiyse ilk `server` bloğu kullanılır
- İstek `site2.local` için geldiyse ikinci `server` bloğu kullanılır

Böylece tek bir NGINX instance ile birden fazla statik site sunabilirsin.

---

## Bu egzersizde ne yapıyorsun?

Bu egzersizde `nginx.conf` dosyasını düzenleyeceksin.

Görevin:

- Aynı portu dinleyen iki ayrı `server` bloğu oluşturmak
- Birinci domain için `server_name` değerini `site1.local` yapmak
- İkinci domain için `server_name` değerini `site2.local` yapmak
- Birinci domain için `root` değerini `/usr/share/nginx/site1` olarak ayarlamak
- İkinci domain için `root` değerini `/usr/share/nginx/site2` olarak ayarlamak
- Her iki blokta da `index index.html;` kullanmak
- Statik içeriklerin doğru domain’e göre sunulmasını sağlamak

Bu egzersizde amaç, tek bir genel `server` bloğu yazmak değil; iki ayrı sanal host tanımlamaktır.

---

## Başarı kriterleri

Bu egzersiz başarılı sayılması için:

- Yapılandırma sözdizimi hatasız olmalı
- `site1.local` isteği birinci statik içeriği döndürmeli
- `site2.local` isteği ikinci statik içeriği döndürmeli
- İki domain aynı port üzerinde doğru şekilde ayrıştırılmalı
- Nginxlings içinde egzersiz tamamlandı olarak işaretlenmeli

---

## İpuçları

- Domain bazlı ayrım için `server_name` kullanılır
- Statik içerik sunarken `root` ve `index` birlikte düşünülmelidir
- Aynı portta birden fazla `server` bloğu kullanmak normaldir
- Burada asıl öğrenmen gereken şey, NGINX’in `Host` başlığına göre doğru siteyi seçmesidir