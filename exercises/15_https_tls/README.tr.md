# 15 – HTTPS (TLS) Quick Lab

## Amaç

Bu egzersizde, NGINX üzerinde temel TLS terminasyonu kuracaksın.

Bu çalışmada odak noktası şunlardır:

- Self-signed bir sertifika üretmek
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

- Docker image build aşamasında bir self-signed sertifika üretmek
- Sertifika dosyalarını `/etc/nginx/certs/` altında oluşturmak
- HTTP için bir `server` bloğu tanımlamak
- Bu blokta gelen istekleri HTTPS’e yönlendirmek
- HTTPS için ayrı bir `server` bloğu tanımlamak
- Bu blokta `listen 443 ssl;` kullanmak
- `ssl_certificate` ve `ssl_certificate_key` direktiflerini doğru dosyalara bağlamak
- HTTPS üzerinden basit bir cevap döndürmek

Bu egzersizde asıl öğrenmen gereken şey, NGINX’in TLS’i terminate etmesi ve düz HTTP trafiğini güvenli HTTPS’e yönlendirmesidir.

---

## Başarı kriterleri

Bu egzersiz başarılı sayılması için:

- Docker image başarıyla build olmalı
- NGINX yapılandırması hatasız olmalı
- HTTP endpoint bir HTTPS yönlendirmesi dönmeli
- HTTPS endpoint geçerli şekilde cevap vermeli
- TLS sertifika ve key dosyaları doğru path’te kullanılmalı
- Nginxlings içinde egzersiz tamamlandı olarak işaretlenmeli

---

## İpuçları

- Self-signed sertifika üretmek için `openssl req -x509` kullanılabilir
- Sertifika ve private key path’leri NGINX config ile birebir uyuşmalıdır
- Redirect için en sade yaklaşım `return 301 https://$host$request_uri;` şeklindedir
- HTTPS sunucusunda `ssl` parametresi doğrudan `listen` üzerinde olmalıdır