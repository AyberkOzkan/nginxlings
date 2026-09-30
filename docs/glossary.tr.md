# Türkçe çeviri kuralları ve terim sözlüğü

Bu belge `README.tr.md` dosyalarının, `i18n/exercises/tr.json` adlarının ve
`i18n/hints/tr.json` ipuçlarının tek bir terim diliyle yazılması içindir
(2026-09-30'da 18 README bu kurallarla çevrildi).

## Kurallar

1. Metin tamamen Türkçe olur. Bir teknik terim dosyada **ilk geçtiği yerde** Türkçesi
   ve parantez içinde İngilizcesiyle yazılır: "yük dengeleyici (load balancer)".
   Sonraki geçişlerde yalnızca Türkçesi kullanılır. Parantezi her geçişte tekrarlama.
2. Şunlar **asla değişmez**: çitli kod blokları (```` ``` ```` ya da 4 boşlukla girintili
   bloklar), ters tırnak (`) içindeki her şey, direktif ve bağlam adları (`upstream`,
   `server`, `http`, `stream`, `location`, `main`, `events`, `user`, `zone` ...), dosya
   yolları, komutlar, URL'ler, port ve adresler, check'in İngilizce çıktı mesajları
   (Takıldıysan bölümündeki ters tırnaklı mesajlar), bağlantı metni olan dokümantasyon
   sayfa adları (ör. `[ngx_http_upstream_module](...)`), `TODO`.
3. Markdown yapısı korunur: aynı başlık sayısı ve seviyesi (README.en.md ile birebir; bir
   test bunu denetler), aynı bölüm sırası, aynı liste maddeleri. Bilgi eklenmez ve
   çıkarılmaz; anlam korunur. Kopyala-yapıştır çözüm eklenmez.
4. Başlık (H1), `i18n/exercises/tr.json`'daki adla birebir aynıdır (`NN - Ad` →
   `# NN – Ad`); `tests/repo` bunu denetler. Bölüm başlıkları:
   `## Amaç`, `## Arka plan – ...`, `## Laboratuvar ortamı`, `## Görev`,
   `## Başarı kriterleri`, `## Takıldıysan`, `## Pekiştirme soruları`.
   Egzersizde farklı bir bölüm varsa onu da Türkçeleştir.
5. Türkçe yazım kurallarına ve ç, ğ, ı, İ, ö, ş, ü harflerine tam uy. Kesme işaretiyle
   ek: `nginx.conf`'u, `upstream`'e, NGINX'in.
6. Satırları bugünkü gibi yaklaşık 84 karakterde sar.
7. Parantez içinde İngilizce terimi yalnızca sözlükte (ya da gerçekten yerleşik bir
   terimde) kullan; sıradan kelimeleri parantezle açıklama.

## H1 başlıkları

| ID | H1 |
| --- | --- |
| 01 | # 01 – Basit Statik NGINX Sunucusu |
| 02 | # 02 – Yapılandırma Düzeni ve Varsayılan Yollar |
| 03 | # 03 – Bağlam Hiyerarşisi (main/http/server/location) |
| 04 | # 04 – İşçi Süreç Kullanıcısı ve İzinler |
| 05 | # 05 – Resmi Depodan Kurulum ve Doğrulama |
| 06 | # 06 – HTTP Ters Vekil Sunucu Temelleri |
| 07 | # 07 – HTTP Yük Dengeleyici (Katman 7) |
| 08 | # 08 – TCP Yük Dengeleyici (Katman 4, stream) |
| 09 | # 09 – Paylaşılan Bellek Bölgelerinin Temelleri |
| 10 | # 10 – Paylaşılan Bellekte Sunucu Grubu Durumu (zone up_mem) |
| 11 | # 11 – Bölge Eşitleme ve API (zone_sync) |
| 12 | # 12 – Günlükler ve Değişkenler |
| 13 | # 13 – Modüler Yapılandırma Tasarımı (include) |
| 14 | # 14 – Birden Fazla Alan Adıyla Statik Siteler |
| 15 | # 15 – HTTPS (TLS) Hızlı Uygulama |
| 16 | # 16 – Paylaşılan Bellek Bölgesi Boyutlandırma ve Sınırlar |
| 17 | # 17 – NGINX Kullanıcısını ve Süreçlerini Doğrulama |
| 18 | # 18 – OSI Katmanları ve Kullanım Senaryoları |

## Sözlük (ilk geçiş biçimi → sonraki geçişler)

| İngilizce | İlk geçiş | Sonra |
| --- | --- | --- |
| backend | arka uç sunucu (backend) | arka uç sunucu / arka uç |
| frontend listener | ön uç dinleyici | ön uç |
| load balancer / load balancing | yük dengeleyici (load balancer) / yük dengeleme | aynı |
| reverse proxy | ters vekil sunucu (reverse proxy) | ters vekil sunucu |
| proxy (fiil) | iletmek | iletmek |
| upstream grubu (kavram) | sunucu grubu (upstream) | sunucu grubu; direktif `upstream` |
| context | bağlam (context) | bağlam; `http` bağlamı |
| directive | direktif | direktif |
| worker process | işçi süreç (worker process) | işçi süreç |
| master process | ana süreç (master process) | ana süreç |
| process | süreç | süreç |
| account | hesap | hesap |
| container | konteyner (container) | konteyner |
| image (Docker) | imaj (image) | imaj |
| build (imaj) | oluşturmak (build) | oluşturmak |
| host (makine) | ana makine (host) | ana makine |
| request / response | istek / yanıt | istek / yanıt |
| header | başlık (header) | başlık |
| body | gövde | gövde |
| status code | durum kodu | durum kodu |
| log | günlük (log) | günlük |
| access log | erişim günlüğü (access log) | erişim günlüğü |
| error log | hata günlüğü (error log) | hata günlüğü |
| log format | günlük biçimi (log format) | günlük biçimi |
| shared memory zone | paylaşılan bellek bölgesi (shared memory zone) | bellek bölgesi / bölge |
| zone | bölge (zone) | bölge; direktif `zone` |
| rate limiting | hız sınırlama (rate limiting) | hız sınırlama |
| key (limit_req) | anahtar | anahtar |
| state | durum | durum |
| LRU eviction | en uzun süredir kullanılmayanı çıkarma (LRU) | çıkarma |
| resolver | ad çözümleyici (resolver) | ad çözümleyici; direktif `resolver` |
| runtime | çalışma zamanı | çalışma zamanı |
| failover | yedeğe geçiş (failover) | yedeğe geçiş |
| retry | yeniden deneme | yeniden deneme |
| round-robin | sırayla dağıtım (round-robin) | sırayla dağıtım |
| virtual server / host | sanal sunucu (virtual server) | sanal sunucu |
| default server | varsayılan sunucu (default server) | varsayılan sunucu |
| domain | alan adı (domain) | alan adı |
| web root / document root | web kök dizini (web root) | kök dizin |
| prefix | önek (prefix) | önek |
| path | yol (path) | yol |
| directory | dizin | dizin |
| permission / owner / group / mode | izin / sahip / grup / izin kipi (mode) | aynı |
| certificate / private key | sertifika / özel anahtar (private key) | sertifika / özel anahtar |
| TLS termination | TLS sonlandırma (TLS termination) | TLS sonlandırma |
| handshake | el sıkışma (handshake) | el sıkışma |
| self-signed | kendinden imzalı (self-signed) | kendinden imzalı |
| redirect | yönlendirme (redirect) | yönlendirme |
| cache | önbellek (cache) | önbellek |
| dynamic module | dinamik modül (dynamic module) | dinamik modül |
| check (Nginxlings denetimi) | kontrol | kontrol; dosya adı `check.sh` |
| placeholder | yer tutucu | yer tutucu |
| lab | laboratuvar | laboratuvar |
| hint | ipucu | ipucu |
| starting state | başlangıç durumu | başlangıç durumu |
| Layer 4 / Layer 7 | Katman 4 / Katman 7 | aynı |
| socket / bind | soket / bağlamak (bind) | soket / bağlamak |
| timeout | zaman aşımı | zaman aşımı |
| scheme (URL) | şema | şema |
| configuration / config | yapılandırma | yapılandırma |
| package / distribution | paket / dağıtım | paket / dağıtım |
| marker (yanıttaki işaret metni) | işaret metni | işaret metni |
| foreground | ön planda | ön planda |
| token | belirteç (token) | belirteç |
| handler | işleyici | işleyici |
| base image | taban imaj | taban imaj |
| signing key | imza anahtarı | imza anahtarı |
| endpoint | uç nokta (endpoint) | uç nokta |
| response header | yanıt başlığı (response header) | yanıt başlığı |
| content type | içerik türü (content type) | içerik türü |
| regular expression | düzenli ifade (regular expression) | düzenli ifade |
| query string | sorgu dizesi (query string) | sorgu dizesi |
| method (HTTP) | yöntem | yöntem |
| cookie | çerez | çerez |
| byte | bayt | bayt |
| replica | kopya | kopya |
| host name | ana makine adı (host name) | ana makine adı |

Kısaltmalar ve adlar olduğu gibi kalır: NGINX, HTTP, HTTPS, TLS, TCP, UDP, DNS, SNI, API,
JSON, MIME, OSI, Docker, Alpine, Debian, BusyBox, PostgreSQL, Redis, IP, URI, URL.
