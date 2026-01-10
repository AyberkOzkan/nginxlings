# 08 – TCP/UDP Load Balancer (Layer 4, stream)

## Amaç

Bu egzersizde NGINX’i `stream {}` context’i ile **Layer 4 (TCP) load balancer**
olarak yapılandıracaksın.

Hedeflerin:

- `http {}` yerine `stream {}` kullanmak.
- İki TCP backend içeren bir `upstream` tanımlamak:

      127.0.0.1:9001
      127.0.0.1:9002

- Frontend olarak 9000 portunda dinleyip gelen TCP bağlantılarını:

      proxy_pass tcp_backend;

  ile upstream’e dağıtmak.
- Bağlantıların iki backend’e dağıldığını göstermek.


---

## Arkaplan – HTTP mi, stream mi?

NGINX’te protokole göre farklı context’ler vardır:

- `http {}` → HTTP/HTTPS (Layer 7)
- `stream {}` → TCP/UDP (Layer 4)

TCP load balancer kurarken `location` blokları kullanılmaz. Bunun yerine:

    stream {
        upstream tcp_backend {
            server 127.0.0.1:9001;
            server 127.0.0.1:9002;
        }

        server {
            listen 9000;
            proxy_pass tcp_backend;
        }
    }

Varsayılan dengeleme yöntemi round-robin’dir.

---

## Ne değiştireceksin?

Düzenleyeceğin dosya:

    nginx.conf

Container içindeki başlangıç script’i iki basit TCP servisini otomatik başlatır:

- Backend A: 127.0.0.1:9001
- Backend B: 127.0.0.1:9002

Senin görevin, NGINX’in 9000 portunda TCP kabul edip bu bağlantıları upstream’e
dağıtmasını sağlamaktır.

---

## Başarı kriterleri

Otomatik doğrulama başarılı sayılması için:

- Docker imajı build edilebilmeli.
- Container içinde `nginx -t` hatasız geçmeli.
- NGINX (container içinde) 9000 TCP portunda dinlemeli.
- Frontend üzerinden yapılan birden fazla TCP bağlantısında her iki backend’den de
  yanıt görülebilmeli (dağıtım doğrulanmış olmalı).

---

## İpuçları

- `stream {}` bloğu `nginx.conf` içinde üst seviyede olmalı; `http {}` içine yazılmaz.
- Stream tarafında `proxy_pass` şu şekilde kullanılır:

      proxy_pass tcp_backend;

  `http://` yazılmaz.
- `server_name`, `location`, `proxy_set_header` gibi HTTP direktifleri `stream {}` içinde kullanılmaz.
