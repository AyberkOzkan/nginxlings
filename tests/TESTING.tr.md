# Nginxlings'i test etmek

Bu belge bütün test katmanlarını, çalıştırılacak komutları ve her katmanın
kapsadığı adımları listeler. İngilizcesi `TESTING.md` dosyasındadır.

## Test katmanları

| Katman | Yer | Neyi sınar | Gereken | Süre | CI'da |
| --- | --- | --- | --- | --- | --- |
| Birim (unit) | `cmd/nginxlings/*_test.go` | Programın iç işleyişi: düzen hesabı, güncelleme mantığı, durum, komut satırı, çeviri yükleme | Go | ~1 sn | evet |
| Depo | `tests/repo` | Dosyalar arası sözleşmeler: manifest, çeviriler, README yapısı ve başlıkları, kontrol betiği kuralları, başlangıç durumları, sürüm ve CI yapılandırması | Go | <1 sn | evet |
| Kontrol kütüphanesi | `tests/checklib` | `exercises/lib/check-lib.sh`, bash ve sahte programlarla | Go, bash | ~1 sn | evet |
| Arayüz (TUI) | `tests/tui` | Derlenmiş programın sahte terminalde (pseudo-terminal) çalışması, ekranın bir terminal öykünücüsüyle (emulator) okunması | Go, Linux pty | ~3 sn | evet |
| Egzersizler | `tests/exercises` | Her egzersizin gerçek `check.sh` betiği: başlangıç dosyaları başarısız, çözüm başarılı | Docker, ana makinede (host) NGINX, çözümler | ~90 sn | hayır (atlanır) |

Birim testleri kodun yanında kalır; çünkü Go'da dışa açılmamış kodu sınayan
testler aynı pakette olmak zorundadır. `tests/` altındaki her şey depoya ve
programa dışarıdan bakar ve ortak yardımcıları `tests/testutil` paketinden alır.

## Her değişiklikte

```sh
gofmt -l .                      # hiçbir şey yazdırmamalı
go vet ./...
go test ./...                   # birim, depo, kontrol kütüphanesi, arayüz
for script in exercises/*/check.sh exercises/lib/*.sh; do bash -n "$script"; done
```

`NGINXLINGS_SOLUTIONS` tanımlı değilse `go test ./...` egzersiz testlerini atlar.

## Sürümden önce

```sh
go test -race ./cmd/nginxlings ./tests/tui
go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
NGINXLINGS_SOLUTIONS=$HOME/projects/nginxlings-solutions/solutions \
  go test ./tests/exercises -v -count=1 -timeout 30m
```

Egzersiz testleri, konteyner (container) egzersizleri için çalışan bir Docker
servisine, 01, 02, 03 ve 06 için de ana makinede `nginx`'e ihtiyaç duyar. Eksik
bir gereksinim ilgili egzersizi atlatır ve sebebini yazar; görmek için `-v` ile
çalıştır. Ana makinede çalışan kontroller, çekirdek izin verdiğinde özel bir ağ
ad alanında (`unshare -rn`) koşar; böylece makinedeki 8080 ve 8081 portlarını
kullanan servisler karışmaz.

Çözümler asla depoya girmez. `NGINXLINGS_SOLUTIONS`, her egzersiz kimliği için
bir alt dizin içeren bir dizini gösterir; bu alt dizinlerde yalnızca başlangıç
dosyalarından farklı olan dosyalar bulunur.

## Test adımları

### Arayüz (`tests/tui`)

| Adım | Test |
| --- | --- |
| 30x6'dan 220x60'a 15 boyutta, İngilizce ve Türkçe başlat; ilk çerçeve eksiksiz olmalı | `TestTUIFrameAtEachSize` |
| Çalışan tek bir programı büyüyen ve küçülen boyutlardan geçir; her çerçeve yerine oturmalı ve bozulmamalı | `TestTUIResizeKeepsFrameIntact` |
| Çerçeve denetimi bozulmuş ekranları reddetmeli | `TestFrameCheckRejectsDamagedScreens` |
| `s`/`w` seçimi taşır; yeniden çizimden hızlı basılan tuşların hepsi sayılır | `TestTUINavigationCountsEveryKeyPress` |
| `h` ipucunu açar ve kapatır | `TestTUIHintTogglesWithInstructions` |
| `l` başlıkların, liste adlarının ve README'nin dilini değiştirir | `TestTUILanguageSwitchTranslatesTheScreen` |
| PgDn/PgUp talimatları kaydırır | `TestTUIScrollMovesTheInstructions` |
| `c` başarılı ve başarısız bir kontrol çalıştırır; çıktı paneli, `Tab`, liste işaretleri, ilerleme ve kaydedilen durum buna uyar | `TestTUICheckFlowShowsResultAndProgress` |
| `r` onay ister; `n` iptal eder, `y` ilerlemeyi sıfırlar | `TestTUIResetClearsProgress` |
| Yeniden başlatma son seçilen egzersizi geri getirir | `TestTUIRestartRestoresTheSelectedExercise` |

Geçerli bir çerçeve şu demektir: ilk satırda başlık, sondan ikinci satırda durum
satırı, son satırda tuş yardımı; sağ kenarı son sütuna ulaşan iki kapalı panel;
80 sütundan itibaren yan yana, daha darda alt alta paneller; terminal 9 satırdan
kısa ya da 80 sütundan dar ve 14 satırdan kısaysa kenarsız en fazla üç satır.

Kontrol akışı, kontrolleri anında yanıt veren iki egzersizlik sahte bir kök
dizin kullanır; bu yüzden Docker'a da NGINX'e de ihtiyaç duymaz.

### Egzersizler (`tests/exercises`)

Her egzersiz dizini için:

1. Egzersizi ve `exercises/lib` dizinini geçici bir dizine kopyala ve `check.sh`
   çalıştır: son satır `[FAIL]` ile başlamalı, çıkış kodu sıfırdan farklı olmalı.
2. Yeniden kopyala, çözümü uygula, `check.sh` çalıştır: `[PASS]` ve çıkış kodu 0.
3. Çözümü uygula; `nginx.conf`, `conf.d/*.conf` ve `server.d/*.conf` dosyalarını
   yorumsuz tek satıra indir, `check.sh` çalıştır: `[PASS]`.

### Depo (`tests/repo`)

Her `go test ./...` çalıştırmasında koşar. Denetledikleri arasında: manifest ile
dizinler uyuşur; her çeviri dosyası her egzersizi kapsar; README başlıkları her
dilde liste adlarıyla aynıdır; README başlık seviyeleri diller arasında aynıdır;
başlangıç dosyaları bitmiş çözümü içermez; kontroller ortak kütüphaneyi ve
sabitlenmiş imajları kullanır; sürüm arşivi her dosyayı içerir.

## Arayüzü canlı izlemek

Elle yapılan bir oturumu izlemek için programı erişebileceğin bir sokette tmux
oturumunda başlat ve başka bir terminalden salt okunur olarak bağlan:

```sh
go build -o /tmp/nginxlings ./cmd/nginxlings
tmux -S /tmp/nginxlings.sock new-session -d -s nlx -x 80 -y 24 \
  "/tmp/nginxlings --state /tmp/nginxlings-state.json"
tmux -S /tmp/nginxlings.sock set-option -t nlx window-size manual
tmux -S /tmp/nginxlings.sock attach -t nlx -r    # başka terminalde; Ctrl+b d ayrılır
```

Test oturumu kendi ilerlemeni değiştirmesin diye ayrı bir `--state` dosyası kullan.

## Test eklemek

- **Yeni egzersiz:** çözümünü çözümler dizinine ekle ve egzersiz testlerini
  çalıştır; başlangıç dosyaları çözüm sanılabilecekse
  `tests/repo/starting_state_test.go` dosyasına bir kayıt ekle.
- **Yeni tuş ya da görünüm:** `tests/tui/flow_test.go` dosyasına bir akış ekle.
  Metni `waitForText`, tam çerçeveyi `waitForFrame` ile bekle; asla sabit süre
  uyuma.
- **Düzen değişikliği:** `tests/tui/frame_test.go` içindeki `frameSizes`
  listesine boyut ekle ve `isCompact` kuralını `cmd/nginxlings/view.go` ile aynı
  tut.
- **Yeni dosya sözleşmesi:** `tests/repo` içindeki ilgili dosyaya bir test ekle.

## Sorun giderme

- `Port ... is already in use`: bir kontrolün portunu başka bir program tutuyor.
  Onu durdur ya da portu kontrolün söylediği değişkenle değiştir.
- `this check needs a running Docker daemon`: Docker'ı başlat; egzersiz başarısız
  sayılmadı, atlandı.
- Portlar boş görünmesine rağmen ana makine kontrolleri bağlama (bind) hatası
  veriyorsa: `unshare` engellenmiş olabilir (örneğin bazı konteynerlerin içinde);
  kontroller o zaman ana makinenin ağında koşar.
