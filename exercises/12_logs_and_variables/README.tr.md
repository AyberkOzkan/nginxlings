# 12 – Loglar ve Değişkenler

## Amaç

Bu egzersizde özel bir access log formatı tanımlayıp NGINX değişkenlerinin
access log’a nasıl yazdırıldığını göreceksin.

Hedefler:

- `main` adında özel bir `log_format` tanımlamak,
- `access_log` satırında bu `main` formatını kullanmak,
- Log satırlarında aşağıdaki değişkenlerin değerlerini görmek:
  - `$remote_addr`
  - `$status`
  - `$request`
  - `$http_user_agent`

Doğrulama, birkaç HTTP isteği gönderip access log çıktısında bu alanların
oluştuğunu kontrol eder.

---

## Görev

Bu egzersizin `nginx.conf` dosyasını düzenle ve şunları yap:

1. `http { ... }` context’i içinde `main` adında bir `log_format` tanımla.
2. `access_log` satırında bu `main` formatını kullan.
3. 8080 portunda dinleyen ve `/` isteğine HTTP 200 döndüren bir `server`
   bloğu olduğundan emin ol.

`log_format main` içinde en az şunlar bulunmalı:

    $remote_addr
    $status
    $request
    $http_user_agent

---

## Başarı kriterleri

Bu egzersiz şu koşullarda başarılı sayılır:

- Access log aktif ve `main` formatı ile yazılıyor olmalı.
- Test isteklerinden sonra access log’da şu bilgileri içeren satırlar olmalı:
  - `$remote_addr` kaynak IP değeri
  - `$status` HTTP durum kodu
  - `$request` istek satırı
  - `$http_user_agent` user-agent değeri

---

## İpuçları

- `log_format` mutlaka `http` context’i içinde tanımlanır.
- `access_log` `http` veya `server` içinde tanımlanabilir; bu lab için ikisi de olur.
- Access log hedefini `/dev/stdout` yaparsan container log’larında kolayca görebilirsin.
