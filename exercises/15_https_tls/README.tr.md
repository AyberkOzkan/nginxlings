# 15 – HTTPS (TLS) Quick Lab

## Amaç

Bu egzersizde, NGINX üzerinde temel TLS terminasyonu kuracaksın.

Bu çalışmada odak noktası şunlardır:

- Self-signed bir sertifika kullanmak (`Dockerfile` bunu zaten üretiyor)
- HTTPS için `listen 443 ssl;` kullanmak
- `ssl_certificate` ve `ssl_certificate_key` tanımlamak
- HTTP isteklerini HTTPS’e yönlendirmek

---

## Arkaplan

NGINX, istemciden gelen TLS bağlantısını sonlandırarak HTTPS trafiğini yönetebilir.

Bu yapı genelde iki parçadan oluşur:

1. HTTP sunucusu  
   Gelen 80 portu isteklerini HTTPS’e yönlendirir.

2. HTTPS sunucusu  
   443 portunda SSL/TLS ile dinler ve sertifika kullanır.

Temel mantık şu şekildedir:

    server {
        listen 80;
        return 301 https://$host$request_uri;
    }

    server {
        listen 443 ssl;
        ssl_certificate ...;
        ssl_certificate_key ...;
    }

Bu egzersizde amaç, bu temel yapıyı kurmak ve NGINX’in HTTPS üzerinden içerik sunmasını sağlamaktır.

---

## Bu egzersizde ne yapıyorsun?

Görevin:

- `Dockerfile`'ın build sırasında zaten ürettiği self-signed sertifikayı
  kullanmak: `/etc/nginx/certs/self.crt` ve `/etc/nginx/certs/self.key`
- HTTP için bir `server` bloğu tanımlamak
- Bu blokta gelen istekleri HTTPS’e yönlendirmek
- HTTPS için ayrı bir `server` bloğu tanımlamak
- Bu blokta `listen 443 ssl;` kullanmak
- `ssl_certificate` ve `ssl_certificate_key` direktiflerini doğru dosyalara bağlamak
- HTTPS üzerinden `/` için `tls-ok` içeren bir cevap döndürmek

Bu egzersizde asıl öğrenmen gereken şey, NGINX’in TLS’i terminate etmesi ve düz HTTP trafiğini güvenli HTTPS’e yönlendirmesidir.

---

## Başarı kriterleri

Bu egzersiz başarılı sayılması için:

- Docker image başarıyla build olmalı
- NGINX yapılandırması hatasız olmalı
- HTTP endpoint, `Location` başlığı `https://` ile başlayan bir yönlendirme
  (301, 302, 303, 307 veya 308) dönmeli
- HTTPS endpoint `tls-ok` içeren bir gövde dönmeli
- TLS sertifika ve key dosyaları doğru path’te kullanılmalı
- Nginxlings içinde egzersiz tamamlandı olarak işaretlenmeli

---

## İpuçları

- `Dockerfile` içindeki `openssl req -x509` adımı self-signed sertifikanın
  nasıl üretildiğini gösterir; bu adımı değiştirmen gerekmez
- Sertifika ve private key path’leri NGINX config ile birebir uyuşmalıdır
- Redirect için en sade yaklaşım `return 301 https://$host$request_uri;` şeklindedir
- HTTPS sunucusunda `ssl` parametresi doğrudan `listen` üzerinde olmalıdır